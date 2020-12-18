package gol

import (
	"fmt"
	"math"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/errors"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type worker struct {
	stubs.Remote
	strip              strip
	instructionResults chan stubs.InstructionResult
}

type strip struct {
	top    int
	height int
}

func (d *Distributor) logf(format string, obj ...interface{}) {
	fmt.Printf("%s	%s\n",
		bold("distributor (%s):", d.thisAddr),
		fmt.Sprintf(format, obj...))
}

type Distributor struct {
	thisAddr   string
	controller stubs.Remote
	p          Params
	workers    []worker
	//currentState stubs.Grid
	stateUpdateChan chan stubs.InstructionResult
	//stateRequestChan chan stubs.Instruction
	initialStateChan chan stubs.DistributorInitialState
	//gameEndChan      chan bool
	//workerStateUpdates chan stubs.InstructionResult
}

func (d *Distributor) GetState(req stubs.Instruction, res *stubs.InstructionResult) (err error) {
	*res = d.fetchState(req)
	if res.CurrentTurn == d.p.Turns {

		d.controller.Go("Controller.GameFinished", *res, nil, nil)
		return errors.GameAlreadyFinished
	}
	return
}

func (d *Distributor) SetInitialState(req stubs.DistributorInitialState, res *bool) (err error) {
	d.initialStateChan <- req
	return
}

func (d *Distributor) WorkerState(req stubs.InstructionResult, res *bool) (err error) {
	d.workers[req.WorkerId].instructionResults <- req
	return
}

//func (d *Distributor) GameFinished(req bool, res *bool) (err error) {
//	fmt.Println("game finished!")
//	d.gameEndChan <- req
//	return
//}

func makeStrips(totalHeight, numStrips int) []strip {
	result := make([]strip, 0, numStrips)
	thread := 0
	for i := 0; i < totalHeight; thread++ {
		// calculate (p.ImageHeight - i) / n and round up
		size := int(math.Ceil(float64(totalHeight-i) / float64(numStrips-thread)))
		result = append(result, strip{top: i, height: size})
		i += size
	}
	return result
}

func (d *Distributor) startWorkers(state stubs.Grid) {
	d.logf("starting workers")
	strips := makeStrips(state.Height, d.p.Threads)
	for i := 0; i < d.p.Threads; i++ {
		d.workers[i].strip = strips[i]
		d.workers[i].Call(stubs.SetState, stubs.WorkerInitialState{
			WorkerId: i,
			MaxTurns: d.p.Turns,
			Grid: stubs.Grid{
				Width:  state.Width,
				Height: d.workers[i].strip.height,
				Cells:  state.Cells[d.workers[i].strip.top : d.workers[i].strip.top+d.workers[i].strip.height],
			},
			WorkerBelowAddr: d.workers[(i+1)%d.p.Threads].Addr,
			DistributorAddr: d.thisAddr,
		}, nil)
		d.logf("started worker %d/%d", i, d.p.Threads)
	}
}

func (d *Distributor) combineStateUpdates() stubs.InstructionResult {
	result := stubs.InstructionResult{
		State: stubs.Grid{
			Width:  d.p.ImageWidth,
			Height: d.p.ImageHeight,
		},
	}
	for i := 0; i < len(d.workers); i++ {
		workerState := <-d.workers[i].instructionResults
		if workerState.State.Cells != nil {
			result.State.Cells = append(result.State.Cells, workerState.State.Cells...)
		}
		result.CurrentTurn = workerState.CurrentTurn
		result.AliveCellsCount += workerState.AliveCellsCount
	}

	return result
}

func (d *Distributor) fetchState(request stubs.Instruction) stubs.InstructionResult {
	d.logf("sending state request (%v)", request)
	d.workers[0].Call(stubs.GetWorkerState, request, nil)

	return d.combineStateUpdates()
}

func (d *Distributor) run() {
	d.logf("starting distributor")

	for i := 0; i < len(d.workers); i++ {
		d.workers[i].Connect()
		d.workers[i].instructionResults = make(chan stubs.InstructionResult, 2)
	}

	for {
		select {
		case initialState := <-d.initialStateChan:
			d.logf("setting initial state")
			d.p = Params{
				Turns:       initialState.MaxTurns,
				Threads:     len(d.workers),
				ImageWidth:  initialState.Grid.Width,
				ImageHeight: initialState.Grid.Height,
			}
			d.controller = stubs.Remote{Addr: initialState.ControllerAddr}
			d.controller.Connect()
			d.logf("connected to controller")
			d.startWorkers(initialState.Grid)
			//case stateRequest := <-d.stateRequestChan:
			//	d.stateUpdateChan <- d.fetchState(stateRequest)
			//case stateUpdate := <-d.stateUpdateChan:
			//	if stateUpdate.CurrentTurn == d.p.Turns {
			//		d.controller.Call("Controller.GameFinished", stateUpdate, nil)
			//	} else {
			//		d.controller.Call("Controller.GameFinished", stateUpdate, nil)
			//	}
			//	go d.combineStateUpdates()
			//	//d.controller.Call("Controller.GameFinished",
			//	//	d.combineStateUpdates(stubs.GetWholeState|stubs.GetCurrentTurn|stubs.GetAliveCellsCount), nil)
		}
	}
}

func RunDistributor(thisAddr string, workerAddrs []string) {
	workers := make([]worker, len(workerAddrs))
	for i, addr := range workerAddrs {
		workers[i] = worker{
			Remote: stubs.Remote{Addr: addr},
		}
	}
	thisDistributor := Distributor{
		thisAddr: thisAddr,
		workers:  workers,
		//stateUpdateChan:    make(chan stubs.InstructionResult, 1),
		//stateRequestChan:   make(chan stubs.Instruction, 1),
		initialStateChan: make(chan stubs.DistributorInitialState, 1),
		//gameEndChan:      make(chan bool, 1),
	}
	util.Check(rpc.Register(&thisDistributor))
	listener, _ := net.Listen("tcp", thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	//go stubs.Serve(Distributor{}, *thisAddr)
	thisDistributor.run()
}
