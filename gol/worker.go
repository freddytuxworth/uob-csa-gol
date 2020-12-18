package gol

import (
	"fmt"
	"github.com/fatih/color"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/errors"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct {
	id          int
	jobName     string
	currentTurn int
	turns       int
	distributor stubs.Remote
	workerBelow stubs.Remote

	wrappedGrid stubs.Grid

	initialStates    chan stubs.WorkerInitialState
	stateRequestChan chan stubs.Instruction

	rowAboveIn chan stubs.RowUpdate
	rowBelowIn chan stubs.RowUpdate
	topRowOut  chan stubs.RowUpdate
}

var bold = color.New(color.Bold).SprintfFunc()

func (w *Worker) logf(format string, obj ...interface{}) {
	fmt.Printf("%s	%s\n",
		bold("[%s] worker %d @ turn %d: ", w.jobName, w.id, w.currentTurn),
		fmt.Sprintf(format, obj...))
}

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	w.initialStates <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowUpdate, res *stubs.RowUpdate) (err error) {
	//w.logf("SRA: got setrowabove: %v", req)
	//w.logf("SRA: did put setrowabove req")
	*res = <-w.topRowOut
	w.rowAboveIn <- req
	//w.logf("SRA: did put setrowabove res: %v", *res)
	return
}

func (w *Worker) GetState(req stubs.Instruction, res *bool) (err error) {
	w.logf("got state request %v", req)
	if w.id < 0 {
		return errors.GameAlreadyFinished
	}
	w.stateRequestChan <- req
	w.logf("finished state request %v", req)
	return
}

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
		stateUpdate.AliveCellsCount = countAliveCells(w.wrappedGrid.Width, w.wrappedGrid.Height-2, w.wrappedGrid.Cells[1:w.wrappedGrid.Height-1])
	}
	w.distributor.Call(stubs.SetWorkerState, stateUpdate, nil)
	if request.HasFlag(stubs.Pause) {
		//TODO
	}
}

func (w *Worker) setupInitialState(initialState stubs.WorkerInitialState) {
	w.logf("got initial state from distributor: %v", initialState)

	w.id = initialState.WorkerId
	w.currentTurn = 0
	w.turns = initialState.Turns
	w.jobName = initialState.JobName

	w.stateRequestChan = make(chan stubs.Instruction, 1)

	w.rowAboveIn = make(chan stubs.RowUpdate, 1)
	w.rowBelowIn = make(chan stubs.RowUpdate, 1)
	w.topRowOut = make(chan stubs.RowUpdate, 1)

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

	w.logf("finished setting up state")
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

func countAliveCells(w, h int, state [][]byte) int {
	aliveCells := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			aliveCells += int(state[y][x])
		}
	}

	return aliveCells
}

func (w *Worker) computeTurn() {
	w.logf("begin computing turn %d", w.currentTurn)
	//w.logf("wrapped grid before computing:\n%v", w.wrappedGrid)

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
	//w.logf("completed computing turn %d", w.currentTurn)
	w.currentTurn++
	//w.logf("wrapped grid after computing:\n%v", w.wrappedGrid)
}

func (w *Worker) run() {
	w.id = -1
	w.currentTurn = -1
	w.logf("starting worker")
	w.setupInitialState(<-w.initialStates)

	for {
		var newStateRequest stubs.Instruction = 0
		select {
		case initialState := <-w.initialStates:
			w.setupInitialState(initialState)
		case newStateRequest = <-w.stateRequestChan:
		default:
		}

		w.topRowOut <- stubs.RowUpdate{
			//Turn: w.currentTurn,
			Row: w.wrappedGrid.Cells[1],
		}

		var rowBelowUpdate stubs.RowUpdate
		var rowAboveUpdate stubs.RowUpdate

		if w.currentTurn > w.turns {
			break
		}
		if w.id > 0 {
			rowAboveUpdate = <-w.rowAboveIn
		} else if w.currentTurn == w.turns {
			newStateRequest = stubs.GetWholeState | stubs.GetCurrentTurn | stubs.GetAliveCellsCount
		}

		bottomRowUpdate := stubs.RowUpdate{
			//Turn:         w.currentTurn,
			Row:          w.wrappedGrid.Cells[w.wrappedGrid.Height-2],
			StateRequest: rowAboveUpdate.StateRequest | newStateRequest,
		}

		err := w.workerBelow.Call(stubs.SetRowAbove, bottomRowUpdate, &rowBelowUpdate)
		util.Check(err)
		w.handleStateRequests(bottomRowUpdate.StateRequest)
		if w.id == 0 {
			rowAboveUpdate = <-w.rowAboveIn
		}

		w.wrappedGrid.Cells[0] = rowAboveUpdate.Row
		w.wrappedGrid.Cells[w.wrappedGrid.Height-1] = rowBelowUpdate.Row

		w.computeTurn()
	}
	w.run()
}

func RunWorker(thisAddr string) {
	thisWorker := Worker{initialStates: make(chan stubs.WorkerInitialState, 1)}

	util.Check(rpc.Register(&thisWorker))
	stubs.ServeHTTP(thisAddr)

	//listener, _ := net.Listen("tcp", thisAddr)
	//defer listener.Close()
	//go rpc.Accept(listener)

	thisWorker.run()
	//l.Close()
}
