package gol

import (
	"fmt"
	"math"
	"net/rpc"
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
		bold("[%s] distributor (%s):", d.jobName, d.thisAddr),
		fmt.Sprintf(format, obj...))
}

type Distributor struct {
	thisAddr          string
	jobName           string
	controller        stubs.Remote
	p                 Params
	workers           []worker
	combinedStateChan chan stubs.InstructionResult
	getStateChan      chan stubs.InstructionResult
	endStateChan      chan stubs.InstructionResult
	initialStateChan  chan stubs.DistributorInitialState
}

func (d *Distributor) GetEndState(req bool, res *stubs.InstructionResult) (err error) {
	*res = <-d.endStateChan
	d.logf("sending back endstate %v", *res)
	return
}

func (d *Distributor) GetState(req stubs.Instruction, res *stubs.InstructionResult) (err error) {
	d.sendStateRequest(req)
	*res = <-d.getStateChan
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
			JobName:  d.jobName,
			Turns:    d.p.Turns,
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

func (d *Distributor) combineStateUpdates() {
	d.logf("expecting results from %d workers", d.p.Threads)
	result := stubs.InstructionResult{
		State: stubs.Grid{
			Width:  d.p.ImageWidth,
			Height: d.p.ImageHeight,
		},
	}
	for i := 0; i < d.p.Threads; i++ {
		workerState := <-d.workers[i].instructionResults
		//d.logf("got elem for channel %d", i)
		if workerState.State.Cells != nil {
			result.State.Cells = append(result.State.Cells, workerState.State.Cells...)
		}
		result.CurrentTurn = workerState.CurrentTurn
		result.AliveCellsCount += workerState.AliveCellsCount
	}
	//d.logf("finished collecting, result: %#v", result)
	d.combinedStateChan <- result
	//return result
}

func (d *Distributor) sendStateRequest(request stubs.Instruction) {
	//d.logf("sending state request (%v)", request)
	d.workers[0].Go(stubs.GetWorkerState, request, nil, nil)
}

func (d *Distributor) run() {
	d.logf("starting distributor")

	for i := 0; i < len(d.workers); i++ {
		d.workers[i].Connect()
		d.workers[i].instructionResults = make(chan stubs.InstructionResult, 2)
	}
	d.logf("connected to %d workers", len(d.workers))
	for {
		select {
		case initialState := <-d.initialStateChan:
			d.jobName = initialState.JobName
			d.logf("setting initial state")
			for i := 0; i < len(d.workers); i++ {
				d.workers[i].instructionResults = make(chan stubs.InstructionResult, 2)
			}
			d.combinedStateChan = make(chan stubs.InstructionResult, 1)
			d.getStateChan = make(chan stubs.InstructionResult, 2)
			d.endStateChan = make(chan stubs.InstructionResult, 1)
			d.p.Turns = initialState.Turns
			d.p.Threads = len(d.workers)
			d.p.ImageWidth = initialState.Grid.Width
			d.p.ImageHeight = initialState.Grid.Height

			d.startWorkers(initialState.Grid)
			go d.combineStateUpdates()
		case state := <-d.combinedStateChan:
			if state.CurrentTurn == d.p.Turns {
				d.logf("pushing endstate %v", state)
				d.endStateChan <- state
			}
			d.getStateChan <- state
			go d.combineStateUpdates()
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
		thisAddr: fmt.Sprintf("%s:%d", thisAddr, 8000),
		workers:  workers,
		//combinedStateChan: make(chan stubs.InstructionResult, 2),
		//getStateChan:      make(chan stubs.InstructionResult, 2),
		initialStateChan: make(chan stubs.DistributorInitialState, 1),
	}
	util.Check(rpc.Register(&thisDistributor))
	//rpc.HandleHTTP()
	//l, e := net.Listen("tcp", thisAddr)
	//if e != nil {
	//	log.Fatal("listen error:", e)
	//}
	//go http.Serve(l, nil)
	stubs.ServeHTTP(8000)
	//listener, _ := net.Listen("tcp", thisAddr)
	//defer listener.Close()
	//go rpc.Accept(listener)

	//go stubs.Serve(Distributor{}, *thisAddr)
	thisDistributor.run()
	//l.Close()
}
