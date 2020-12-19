package gol

import (
	"fmt"
	"github.com/fatih/color"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/errors"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct {
	startTime int64
	lastLogTime int64
	id          int
	jobName     string
	currentTurn int
	turns       int
	distributor stubs.Remote
	workerBelow stubs.Remote

	wrappedGrid *stubs.Grid

	initialStates         chan *stubs.WorkerInitialState
	instructionChan       chan stubs.Instruction
	workerInstructionChan chan stubs.Instruction

	rowAboveIn chan *stubs.RowUpdate
	rowBelowIn chan *stubs.RowUpdate
	topRowOut  chan *stubs.RowUpdate
}

var bold = color.New(color.Bold).SprintfFunc()

func (w *Worker) logf(format string, obj ...interface{}) {
	timeNow := time.Now().UnixNano() / 1000000
	fmt.Printf("%s	%s\n",
		bold("%d (+%d) | worker %d @ turn %d [%s]: ", timeNow - w.startTime, timeNow - w.lastLogTime, w.id, w.currentTurn, w.jobName),
		fmt.Sprintf(format, obj...))
	w.lastLogTime = timeNow
}

func (w *Worker) SetInitialState(req stubs.WorkerInitialState, res *bool) (err error) {
	w.initialStates <- &req
	return
}

func (w *Worker) SetRowAbove(req stubs.EncodedRowUpdate, res *stubs.EncodedRowUpdate) (err error) {
	instruction := req.DecodeInto(w.wrappedGrid.Cells[0])
	*res = stubs.RowUpdate{Row: w.wrappedGrid.Cells[1]}.Encode()
	if w.id > 0 {
		w.exchangeBottom(instruction)
	}
	w.workerInstructionChan <- instruction
	return
}

func (w *Worker) DoInstruction(req stubs.Instruction, res *bool) (err error) {
	if w.id < 0 {
		w.logf("discarded state request %v, game already finished", req)
		return errors.GameAlreadyFinished
	}
	w.instructionChan <- req
	return
}

func (w *Worker) handleInstructions(instruction stubs.Instruction) {
	if instruction == 0 {
		return
	}
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
	go func() {
		w.logf(color.GreenString("begin sending state"))
		w.distributor.Call("Distributor.WorkerState", stateUpdate.Encode(), nil)
		w.logf(color.GreenString("finished sending state"))
	}()
	if instruction.HasFlag(stubs.Pause) {
		w.logf("pausing")
		for i := stubs.Instruction(0); i & stubs.Resume == 0; i = <-w.instructionChan {
			w.logf("ignoring instruction %v while paused", i)
		}
	}
	if instruction.HasFlag(stubs.Shutdown) {
		os.Exit(0)
	}
}

func (w *Worker) setupInitialState(initialState *stubs.WorkerInitialState) {
	w.logf("got initial state from distributor")

	w.id = initialState.WorkerId
	w.startTime = time.Now().UnixNano() / 1000000
	w.currentTurn = 0
	w.turns = initialState.Turns
	w.jobName = initialState.JobName

	w.instructionChan = make(chan stubs.Instruction, 1)
	w.workerInstructionChan = make(chan stubs.Instruction, 1)

	initialGrid := initialState.Grid.Decode()
	w.wrappedGrid = &stubs.Grid{
		Width:  initialGrid.Width,
		Height: initialGrid.Height + 2,
		Cells: append(
			append([][]byte{make([]byte, initialGrid.Width)}, initialGrid.Cells...),
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

func (w *Worker) computeTurn() {
	//w.gridMu.Lock()
	if w.currentTurn%10 == 0 {
		w.logf("begin computing turn %d", w.currentTurn)
	}
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
	//time.Sleep(500 * time.Millisecond)
	//w.gridMu.Unlock()
}

func (w *Worker) exchangeBottom(instruction stubs.Instruction) {
	//w.gridMu.Lock()
	bottomRowUpdate := stubs.RowUpdate{
		Row:         w.wrappedGrid.Cells[w.wrappedGrid.Height-2],
		Instruction: instruction,
	}.Encode()

	var encodedRowBelowUpdate stubs.EncodedRowUpdate
	util.Check(w.workerBelow.Call("Worker.SetRowAbove", bottomRowUpdate, &encodedRowBelowUpdate))
	//copy(w.wrappedGrid.Cells[w.wrappedGrid.Height-1], encodedRowBelowUpdate.Decode().Row)
	//out := make([]byte, encodedRowBelowUpdate.Length)
	//encodedRowBelowUpdate.DecodeInto(out)
	//w.logf("row below: %v", stubs.Grid{
	//	Width:  encodedRowBelowUpdate.Length,
	//	Height: 1,
	//	Cells:  [][]byte{out},
	//})
	encodedRowBelowUpdate.DecodeInto(w.wrappedGrid.Cells[w.wrappedGrid.Height-1])
	//w.logf("exchanged down")
	//w.gridMu.Unlock()
	//w.logf("wg %#v", w.wrappedGrid)
	//w.logf("unlocked grid")
}

func (w *Worker) run() {
	w.id = -1
	w.currentTurn = -1
	w.logf("starting worker")
	w.setupInitialState(<-w.initialStates)

	for {
		select {
		case initialState := <-w.initialStates:
			w.setupInitialState(initialState)
		default:
		}

		//w.gridMu.Lock()
		if w.id == 0 {
			select {
			case newInstruction := <-w.instructionChan:
				w.exchangeBottom(newInstruction)
			default:
				w.exchangeBottom(0)
			}
		}

		w.handleInstructions(<-w.workerInstructionChan)

		w.computeTurn()
		if w.currentTurn == w.turns {
			w.handleInstructions(stubs.GetCurrentTurn | stubs.GetWholeState)
			break
		}
	}
	w.run()
}

func RunWorker(port int) {
	thisWorker := Worker{
		initialStates: make(chan *stubs.WorkerInitialState, 1),
	}

	util.Check(rpc.Register(&thisWorker))
	//stubs.ServeHTTP(port)

	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	defer listener.Close()
	go rpc.Accept(listener)
	thisWorker.run()
}
