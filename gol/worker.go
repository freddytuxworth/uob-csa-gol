package gol

import (
	"fmt"
	"github.com/fatih/color"
	"net"
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

	wrappedGrid *stubs.Grid

	initialStates   chan *stubs.WorkerInitialState
	instructionChan chan stubs.Instruction

	rowAboveIn chan *stubs.RowUpdate
	rowBelowIn chan *stubs.RowUpdate
	topRowOut  chan *stubs.RowUpdate
}

var bold = color.New(color.Bold).SprintfFunc()

func (w *Worker) logf(format string, obj ...interface{}) {
	fmt.Printf("%s	%s\n",
		bold("[%s] worker %d @ turn %d: ", w.jobName, w.id, w.currentTurn),
		fmt.Sprintf(format, obj...))
}

func (w *Worker) SetInitialState(req stubs.WorkerInitialState, res *bool) (err error) {
	w.logf("Nitial?!?")
	w.initialStates <- &req
	return
}

func (w *Worker) SetRowAbove(req stubs.EncodedRowUpdate, res *stubs.EncodedRowUpdate) (err error) {
	//w.logf("SRA: got setrowabove: %v", req)
	//w.logf("SRA: did put setrowabove req")
	*res = (<-w.topRowOut).Encode()
	//w.logf("SRA: replying with %#v", res)
	w.rowAboveIn <- req.Decode()
	//w.logf("SRA: did put setrowabove res: %v", *res)
	return
}

func (w *Worker) DoInstruction(req stubs.Instruction, res *bool) (err error) {
	w.logf("got state request %v", req)
	if w.id < 0 {
		return errors.GameAlreadyFinished
	}
	w.instructionChan <- req
	w.logf("finished state request %v", req)
	return
}

func (w *Worker) handleInstructions(instruction stubs.Instruction) {
	if instruction == 0 {
		return
	}
	w.logf(color.GreenString("sending state"))
	stateUpdate := stubs.InstructionResult{WorkerId: w.id}
	if instruction.HasFlag(stubs.GetCurrentTurn) {
		stateUpdate.CurrentTurn = w.currentTurn
	}
	if instruction.HasFlag(stubs.GetWholeState) {
		stateUpdate.State = stubs.Grid{
			Width:  w.wrappedGrid.Width,
			Height: w.wrappedGrid.Height - 2,
			Cells:  w.wrappedGrid.Cells[1 : w.wrappedGrid.Height-1],
		}
	}
	if instruction.HasFlag(stubs.GetAliveCellsCount) {
		stateUpdate.AliveCellsCount = countAliveCells(w.wrappedGrid.Width, w.wrappedGrid.Height-2, w.wrappedGrid.Cells[1:w.wrappedGrid.Height-1])
	}
	w.distributor.Call("Distributor.WorkerState", stateUpdate, nil)
	if instruction.HasFlag(stubs.Pause) {
		//TODO
	}
}

func (w *Worker) setupInitialState(initialState *stubs.WorkerInitialState) {
	w.logf("got initial state from distributor: %v", initialState)

	w.id = initialState.WorkerId
	w.currentTurn = 0
	w.turns = initialState.Turns
	w.jobName = initialState.JobName

	w.instructionChan = make(chan stubs.Instruction, 1)

	w.rowAboveIn = make(chan *stubs.RowUpdate, 1)
	w.rowBelowIn = make(chan *stubs.RowUpdate, 1)
	w.topRowOut = make(chan *stubs.RowUpdate, 1)

	initialGrid := initialState.Grid.Decode()
	w.wrappedGrid = &stubs.Grid{
		Width:  initialGrid.Width,
		Height: initialGrid.Height + 2,
		Cells: append(
			append(make([][]byte, 1), initialGrid.Cells...),
			make([]byte, initialGrid.Width)),
	}

	w.workerBelow = stubs.Remote{Addr: initialState.WorkerBelowAddr}
	w.workerBelow.Connect()
	w.logf("connected to worker below")
	w.distributor = stubs.Remote{Addr: initialState.DistributorAddr}
	w.distributor.Connect()
	w.logf("connected to distributor")

	w.logf("finished setting up state")
}

func shouldSurvive(x int, y int, grid *stubs.Grid) byte {
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

func getAliveCells(w, h int, state [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if state[y][x] > 0 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
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
		var newInstruction stubs.Instruction = 0
		select {
		case initialState := <-w.initialStates:
			w.setupInitialState(initialState)
		case newInstruction = <-w.instructionChan:
		default:
		}

		w.topRowOut <- &stubs.RowUpdate{
			//Turn: w.currentTurn,
			Row: w.wrappedGrid.Cells[1],
		}

		var rowAboveUpdate *stubs.RowUpdate

		if w.currentTurn > w.turns {
			break
		}

		bottomRowUpdate := stubs.RowUpdate{
			Row:         w.wrappedGrid.Cells[w.wrappedGrid.Height-2],
			Instruction: newInstruction,
		}

		if w.id > 0 {
			rowAboveUpdate = <-w.rowAboveIn
			bottomRowUpdate.Instruction = rowAboveUpdate.Instruction
		} else if w.currentTurn == w.turns {
			bottomRowUpdate.Instruction =
				stubs.GetWholeState | stubs.GetCurrentTurn
		}

		//bottomRowUpdate := stubs.RowUpdate{
		//	//Turn:         w.currentTurn,
		//	Row:         w.wrappedGrid.Cells[w.wrappedGrid.Height-2],
		//	Instruction: rowAboveUpdate.Instruction | newInstruction,
		//}

		var encodedRowBelowUpdate stubs.EncodedRowUpdate
		err := w.workerBelow.Call("Worker.SetRowAbove", bottomRowUpdate.Encode(), &encodedRowBelowUpdate)
		util.Check(err)
		w.handleInstructions(bottomRowUpdate.Instruction)
		if w.id == 0 {
			rowAboveUpdate = <-w.rowAboveIn
		}

		w.wrappedGrid.Cells[0] = rowAboveUpdate.Row
		w.wrappedGrid.Cells[w.wrappedGrid.Height-1] = encodedRowBelowUpdate.Decode().Row

		w.computeTurn()
	}
	w.run()
}

func RunWorker(port int) {
	thisWorker := Worker{initialStates: make(chan *stubs.WorkerInitialState, 1)}

	util.Check(rpc.Register(&thisWorker))
	//stubs.ServeHTTP(port)

	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	defer listener.Close()
	go rpc.Accept(listener)

	thisWorker.run()
	//l.Close()
}
