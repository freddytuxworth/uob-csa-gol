package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct{}

var workerId = -1
var currentTurn int
var distributor *rpc.Client
var workerBelow *rpc.Client

var stateChan = make(chan stubs.WorkerInitialState, 1)
var stateRequestChan = make(chan bool, 1)

var rowAboveIn = make(chan stubs.RowUpdate, 1)
var rowBelowIn = make(chan stubs.RowUpdate, 1)
var topRowOut = make(chan stubs.RowUpdate, 1)
var bottomRowOut = make(chan []byte, 1)
var bold = color.New(color.Bold).SprintfFunc()

func logf(format string, obj ...interface{}) {
	fmt.Printf("%s%s\n",
		bold("worker %d @ turn %d: ", workerId, currentTurn),
		fmt.Sprintf(format, obj...))
}

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowUpdate, res *stubs.RowUpdate) (err error) {
	logf("SRA: got setrowabove: %v", req)
	rowAboveIn <- req
	logf("SRA: did put setrowabove req")
	if workerId > 0 {
		bottomRow := <-bottomRowOut
		rowBelowIn <- sendBottomRow(bottomRow, req.ShouldSendState)
	}
	*res = <-topRowOut
	logf("SRA: did put setrowabove res: %v", *res)
	return
}

func (w *Worker) GetState(req bool, res *bool) (err error) {
	logf("got state request")
	stateRequestChan <- true
	return
}

func sendBottomRow(bottomRow []byte, shouldSendState bool) stubs.RowUpdate {
	bottomRowUpdate := stubs.RowUpdate{
		Row:             bottomRow,
		ShouldSendState: shouldSendState,
	}
	logf("starting workerBelow call with %v", bottomRow)
	var response stubs.RowUpdate
	workerBelow.Call(stubs.SetRowAbove, bottomRowUpdate, &response)
	logf("done workerBelow call, got %v", response)
	return response
}

func sendState(wrappedGrid stubs.Grid, distributor *rpc.Client) {
	logf(color.GreenString("sending state"))
	distributor.Go(stubs.SetWorkerState, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
		State:    wrappedGrid.Cells[1 : wrappedGrid.Height-1],
	}, nil, nil)
}

func startEdgeExchange(wrappedGrid stubs.Grid, stateRequest bool) {
	topRowOut <- stubs.RowUpdate{Row: wrappedGrid.Cells[1]}
	rowBelowIn <- sendBottomRow(wrappedGrid.Cells[wrappedGrid.Height-2], stateRequest)
}

func setupInitialState(initialState stubs.WorkerInitialState) stubs.Grid {
	logf("got initial state from distributor: %v", initialState)
	wrappedGrid := stubs.Grid{
		Width:  initialState.Grid.Width,
		Height: initialState.Grid.Height + 2,
		Cells:  append(
			append(make([][]byte, 1), initialState.Grid.Cells...),
			make([]byte, initialState.Grid.Width)),
	}

	workerBelow, _ = rpc.Dial("tcp", initialState.WorkerBelowAddr)
	distributor, _ = rpc.Dial("tcp", initialState.DistributorAddr)

	workerId = initialState.WorkerId
	currentTurn = 0
	if workerId == 0 {
		startEdgeExchange(wrappedGrid, false)
	} else {
		topRowOut <- stubs.RowUpdate{Row: wrappedGrid.Cells[1]}
		bottomRowOut <- wrappedGrid.Cells[wrappedGrid.Height-2]
	}

	return wrappedGrid
}

func computeTurn(wrappedGrid stubs.Grid) {
	logf("begin computing turn")
	logf("wrapped grid before computing:\n%v", wrappedGrid)

	var cellFlips []util.Cell
	for y := 1; y < wrappedGrid.Height-1; y++ {
		for x := 0; x < wrappedGrid.Width; x++ {
			shouldSurvive := gol.ShouldSurvive(x, y, wrappedGrid)
			if shouldSurvive != wrappedGrid.Cells[y][x] {
				cellFlips = append(cellFlips, util.Cell{X: x, Y: y})
			}
		}
	}
	for _, flip := range cellFlips {
		wrappedGrid.Cells[flip.Y][flip.X] ^= 1
	}
	//time.Sleep(1 * time.Millisecond)
	fmt.Println("completed computing turn", currentTurn)
	currentTurn++
	logf("wrapped grid after computing:\n%v", wrappedGrid)
	time.Sleep(500 * time.Millisecond)
}

func workerServer() {
	//distributor, _ := rpc.Dial("tcp", distributorAddr)

	var wrappedGrid stubs.Grid

	//var workerBelow *rpc.Client

	wrappedGrid = setupInitialState(<-stateChan)
	//exchangeEdges(workerBelow, wrappedGrid)

	for {
		gotStateRequest := false
		select {
		case initialState := <-stateChan:
			wrappedGrid = setupInitialState(initialState)
		//exchangeEdges(workerBelow, wrappedGrid)
		case <-stateRequestChan:
			gotStateRequest = true
			logf("received state request in main loop")
		default:
		}

		rowAboveUpdate := <-rowAboveIn
		wrappedGrid.Cells[0] = rowAboveUpdate.Row

		rowBelowUpdate := <-rowBelowIn
		wrappedGrid.Cells[wrappedGrid.Height-1] = rowBelowUpdate.Row

		if rowAboveUpdate.ShouldSendState {
			sendState(wrappedGrid, distributor)
		}

		computeTurn(wrappedGrid)

		if workerId == 0 {
			startEdgeExchange(wrappedGrid, gotStateRequest)
		} else {
			topRowOut <- stubs.RowUpdate{
				Row: wrappedGrid.Cells[1],
			}

			bottomRowOut <- wrappedGrid.Cells[wrappedGrid.Height-2]
		}

		//exchangeEdges(workerBelow, wrappedGrid)

		//if sendTurnUpdate(workerId, currentTurn, distributor) {
		//}
	}
}

func main() {
	thisAddr := flag.String("ip", "127.0.0.1:8020", "IP and port to listen on")
	//workerAddrs := flag.String("workers", "127.0.0.1:8030", "Address of broker instance")
	flag.Parse()

	rpc.Register(&Worker{})
	listener, _ := net.Listen("tcp", *thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	workerServer()
}
