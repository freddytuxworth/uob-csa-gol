package gol

import (
	"fmt"
	"github.com/fatih/color"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct {
	id          int
	currentTurn int
	maxTurns    int
	distributor stubs.Remote
	workerBelow stubs.Remote

	wrappedGrid stubs.Grid

	stateChan        chan stubs.WorkerInitialState
	stateRequestChan chan stubs.Instruction

	rowAboveIn   chan stubs.RowUpdate
	rowBelowIn   chan stubs.RowUpdate
	topRowOut    chan stubs.RowUpdate
	bottomRowOut chan []byte
}

var bold = color.New(color.Bold).SprintfFunc()

func (w *Worker) logf(format string, obj ...interface{}) {
	fmt.Printf("%s	%s\n",
		bold("worker %d @ turn %d: ", w.id, w.currentTurn),
		fmt.Sprintf(format, obj...))
}

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	w.stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowUpdate, res *stubs.RowUpdate) (err error) {
	w.logf("SRA: got setrowabove: %v", req)
	w.rowAboveIn <- req
	w.logf("SRA: did put setrowabove req")
	*res = <-w.topRowOut
	w.logf("SRA: did put setrowabove res: %v", *res)
	return
}

func (w *Worker) GetState(req stubs.Instruction, res *bool) (err error) {
	w.logf("got state request %v", req)
	w.stateRequestChan <- req
	return
}

//func (w *Worker) sendBottomRow(bottomRow []byte, stateRequest stubs.Instruction) stubs.RowUpdate {
//	w.logf("SBR %#v", w.workerBelow)
//	bottomRowUpdate := stubs.RowUpdate{
//		Row:          bottomRow,
//		StateRequest: stateRequest,
//	}
//	w.logf("starting workerBelow call with %v", bottomRow)
//	var response stubs.RowUpdate
//	w.workerBelow.Call(stubs.SetRowAbove, bottomRowUpdate, &response)
//	w.logf("done workerBelow call, got %v", response)
//	return response
//}

func (w *Worker) handleStateRequests(request stubs.Instruction) {
	if request == 0 {
		return
	}
	w.logf(color.GreenString("sending state"))
	stateUpdate := stubs.InstructionResult{WorkerId: w.id}
	if request.HasFlag(stubs.GetCurrentTurn) {
		stateUpdate.CurrentTurn = w.currentTurn
	}
	if request.HasFlag(stubs.GetWholeState) {
		stateUpdate.State = stubs.Grid{
			Width:  w.wrappedGrid.Width,
			Height: w.wrappedGrid.Height - 2,
			Cells:  w.wrappedGrid.Cells[1 : w.wrappedGrid.Height-1],
		}
	}
	if request.HasFlag(stubs.GetAliveCellsCount) {
		stateUpdate.AliveCellsCount = CountAliveCells(w.wrappedGrid.Width, w.wrappedGrid.Height-2, w.wrappedGrid.Cells[1:w.wrappedGrid.Height-1])
	}
	w.distributor.Go(stubs.SetWorkerState, stateUpdate, nil, nil)
	if request.HasFlag(stubs.Pause) {
		//TODO
	}
}

//func (w *Worker) startEdgeExchange(stateRequest stubs.Instruction) {
//	w.topRowOut <- stubs.RowUpdate{Row: w.wrappedGrid.Cells[1]}
//	w.rowBelowIn <- w.sendBottomRow(w.wrappedGrid.Cells[w.wrappedGrid.Height-2], stateRequest)
//}

func (w *Worker) setupInitialState(initialState stubs.WorkerInitialState) {
	w.logf("got initial state from distributor: %v", initialState)

	w.stateRequestChan = make(chan stubs.Instruction, 1)

	w.rowAboveIn = make(chan stubs.RowUpdate, 1)
	w.rowBelowIn = make(chan stubs.RowUpdate, 1)
	w.topRowOut = make(chan stubs.RowUpdate, 1)
	w.bottomRowOut = make(chan []byte, 1)

	w.wrappedGrid = stubs.Grid{
		Width:  initialState.Grid.Width,
		Height: initialState.Grid.Height + 2,
		Cells: append(
			append(make([][]byte, 1), initialState.Grid.Cells...),
			make([]byte, initialState.Grid.Width)),
	}

	w.workerBelow = stubs.Remote{Addr: initialState.WorkerBelowAddr}
	w.workerBelow.Connect()
	w.logf("connected to worker below")
	w.distributor = stubs.Remote{Addr: initialState.DistributorAddr}
	w.distributor.Connect()
	w.logf("connected to distributor")

	w.id = initialState.WorkerId
	w.currentTurn = 0
	w.maxTurns = initialState.MaxTurns

	//if w.id == 0 {
	//	w.startEdgeExchange(0)
	//} else {
	//	w.topRowOut <- stubs.RowUpdate{Row: w.wrappedGrid.Cells[1]}
	//	w.bottomRowOut <- w.wrappedGrid.Cells[w.wrappedGrid.Height-2]
	//}

	w.logf("finished setting up state")
	//return w.wrappedGrid
}

func (w *Worker) computeTurn() {
	w.logf("begin computing turn")
	w.logf("wrapped grid before computing:\n%v", w.wrappedGrid)

	var cellFlips []util.Cell
	for y := 1; y < w.wrappedGrid.Height-1; y++ {
		for x := 0; x < w.wrappedGrid.Width; x++ {
			shouldSurvive := shouldSurvive(x, y, w.wrappedGrid)
			if shouldSurvive != w.wrappedGrid.Cells[y][x] {
				cellFlips = append(cellFlips, util.Cell{X: x, Y: y})
			}
		}
	}
	for _, flip := range cellFlips {
		w.wrappedGrid.Cells[flip.Y][flip.X] ^= 1
	}
	//time.Sleep(1 * time.Millisecond)
	fmt.Println("completed computing turn", w.currentTurn)
	w.currentTurn++
	w.logf("w.wrapped grid after computing:\n%v", w.wrappedGrid)
	time.Sleep(310 * time.Millisecond)
}

func (w *Worker) run() {
	w.id = -1
	w.currentTurn = -1
	w.logf("starting worker")
	w.setupInitialState(<-w.stateChan)

	for {
		var newStateRequest stubs.Instruction = 0
		select {
		case initialState := <-w.stateChan:
			w.setupInitialState(initialState)
		case newStateRequest = <-w.stateRequestChan:
			w.logf("received state request in main loop")
		default:
		}

		w.topRowOut <- stubs.RowUpdate{
			Row: w.wrappedGrid.Cells[1],
		}

		var rowBelowUpdate stubs.RowUpdate
		var rowAboveUpdate stubs.RowUpdate

		if w.id > 0 {
			rowAboveUpdate = <-w.rowAboveIn
			if w.currentTurn == w.maxTurns {
				w.logf("game finished, sending state to distributor\n")
				newStateRequest = stubs.GetWholeState | stubs.GetCurrentTurn | stubs.GetAliveCellsCount
			//} else if w.currentTurn > w.maxTurns {
			//	break
			}
		}

		bottomRowUpdate := stubs.RowUpdate{
			Row:          w.wrappedGrid.Cells[w.wrappedGrid.Height-2],
			StateRequest: rowAboveUpdate.StateRequest | newStateRequest,
		}
		w.workerBelow.Call(stubs.SetRowAbove, bottomRowUpdate, &rowBelowUpdate)

		if w.id == 0 {
			rowAboveUpdate = <-w.rowAboveIn
		}

		w.wrappedGrid.Cells[0] = rowAboveUpdate.Row
		w.wrappedGrid.Cells[w.wrappedGrid.Height-1] = rowBelowUpdate.Row

		w.logf("HSR for %d", bottomRowUpdate.StateRequest)
		w.handleStateRequests(bottomRowUpdate.StateRequest)
		w.computeTurn()

		//if w.id == 0 {
		//	if w.currentTurn == w.maxTurns {
		//		w.logf("game finished, sending state to distributor\n")
		//		//w.distributor.Call("Distributor.GameFinished", false, nil)
		//		w.startEdgeExchange(stubs.GetWholeState | stubs.GetCurrentTurn | stubs.GetAliveCellsCount)
		//		//} else if w.currentTurn > w.maxTurns {
		//		//	break
		//	} else {
		//		w.startEdgeExchange(newStateRequest)
		//	}
		//} else {
		//	//if rowAboveUpdate.StateRequest.HasFlag(stubs.Pause) {
		//	//	continue
		//	//}
		//	//w.topRowOut <- stubs.RowUpdate{
		//	//	Row: w.wrappedGrid.Cells[1],
		//	//}
		//}
	}
	w.run()
}

func RunWorker(thisAddr string) {
	thisWorker := Worker{
		//id:               -1,
		//currentTurn:      -1,
		stateChan: make(chan stubs.WorkerInitialState, 1),
		//stateRequestChan: make(chan stubs.Instruction, 1),
		//
		//rowAboveIn:   make(chan stubs.RowUpdate, 1),
		//rowBelowIn:   make(chan stubs.RowUpdate, 1),
		//topRowOut:    make(chan stubs.RowUpdate, 1),
		//bottomRowOut: make(chan []byte, 1),
	}

	//go stubs.Serve(&thisWorker, thisAddr)
	util.Check(rpc.Register(&thisWorker))
	listener, _ := net.Listen("tcp", thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	thisWorker.run()
}

func shouldSurvive(x int, y int, grid stubs.Grid) byte {
	leftX := util.WrapNum(x-1, grid.Width)
	rightX := (x + 1) % grid.Width

	livingNeighbors :=
		grid.Cells[y-1][leftX] +
			grid.Cells[y-1][x] +
			grid.Cells[y-1][rightX] +
			grid.Cells[y][leftX] +
			grid.Cells[y][rightX] +
			grid.Cells[y+1][leftX] +
			grid.Cells[y+1][x] +
			grid.Cells[y+1][rightX]

	if livingNeighbors == 2 {
		return grid.Cells[y][x]
	} else if livingNeighbors == 3 {
		return 1
	}

	return 0
}

func CountAliveCells(w, h int, state [][]byte) int {
	aliveCells := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			aliveCells += int(state[y][x])
		}
	}

	return aliveCells
}
