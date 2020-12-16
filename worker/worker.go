package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Worker struct{}

var stateChan = make(chan stubs.WorkerInitialState, 1)
var rowAboveIn = make(chan stubs.RowAboveUpdate, 1)
var topRowOut = make(chan stubs.RowAboveResponse, 1)
var stateRequestChan = make(chan bool, 1)

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowAboveUpdate, res *stubs.RowAboveResponse) (err error) {
	rowAboveIn <- req
	*res = <-topRowOut
	fmt.Println("responding", res)
	return
}

func (W *Worker) GetState(req bool, res *bool) (err error) {
	stateRequestChan <- true
	return
}

func sendEdges(workerId int, wrappedGrid stubs.Grid, workerAbove, workerBelow *rpc.Client, makeStateRequest bool, ) {

}

func sendState(workerId, currentTurn int, wrappedGrid stubs.Grid, distributor *rpc.Client) {
	fmt.Println("Sending state")
	distributor.Go(stubs.SetWorkerState, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
		State:    wrappedGrid.Cells[1 : wrappedGrid.Height-1],
	}, nil, nil)
}

func exchangeEdges(workerBelow *rpc.Client, wrappedGrid stubs.Grid, nextWorkerStateRequestWorker int) int {
	fmt.Println("beginning edge exchange")
	topRowOut <- stubs.RowAboveResponse{
		RowBelow: wrappedGrid.Cells[1],
	}
	fmt.Println("top row out sent")
	var response stubs.RowAboveResponse
	workerBelow.Call(stubs.SetRowAbove, stubs.RowAboveUpdate{
		RowAbove:           wrappedGrid.Cells[wrappedGrid.Height-2],
		StateRequestWorker: nextWorkerStateRequestWorker,
	}, &response)
	fmt.Println("done workerBelow call, got", response.RowBelow)

	rowAboveUpdate := <-rowAboveIn
	fmt.Println("got row above", rowAboveUpdate)

	wrappedGrid.Cells[1] = rowAboveUpdate.RowAbove
	wrappedGrid.Cells[wrappedGrid.Height-1] = response.RowBelow
	fmt.Println("finished row exchange")
	return rowAboveUpdate.StateRequestWorker
}

func setupInitialState(initialState stubs.WorkerInitialState) (workerId int, wrappedGrid stubs.Grid, workerBelow, distributor *rpc.Client) {
	wrappedGrid = stubs.Grid{
		Width:  initialState.Width,
		Height: initialState.Height + 2,
		Cells:  make([][]byte, initialState.Height+2),
	}

	for y := 0; y < wrappedGrid.Height; y++ {
		wrappedGrid.Cells[y] = make([]byte, initialState.Width)
		if y > 0 && y < wrappedGrid.Height-1 {
			copy(wrappedGrid.Cells[y], initialState.Cells[y-1])
		}
	}
	workerBelow, _ = rpc.Dial("tcp", initialState.WorkerBelowAddr)
	distributor, _ = rpc.Dial("tcp", initialState.DistributorAddr)

	return initialState.WorkerId, wrappedGrid, workerBelow, distributor
}

func workerServer() {
	//distributor, _ := rpc.Dial("tcp", distributorAddr)

	var workerId int
	var wrappedGrid stubs.Grid

	var distributor *rpc.Client
	var workerBelow *rpc.Client

	gotStateRequest := false
	stateRequestWorker := -1

	currentTurn := 0

	workerId, wrappedGrid, workerBelow, distributor = setupInitialState(<-stateChan)
	exchangeEdges(workerBelow, wrappedGrid, -1)

	for {
		select {
		case initialState := <-stateChan:
			workerId, wrappedGrid, workerBelow, distributor = setupInitialState(initialState)
			currentTurn = 0
			exchangeEdges(workerBelow, wrappedGrid, -1)
		case <-stateRequestChan:
			gotStateRequest = true
		default:
		}
		fmt.Println("begin computing turn", currentTurn)
		for y := 1; y < wrappedGrid.Height-1; y++ {
			for x := 0; x < wrappedGrid.Width; x++ {
				wrappedGrid.Cells[y][x] = gol.ShouldSurvive(x, y, wrappedGrid)
			}
		}
		time.Sleep(500 * time.Millisecond)
		fmt.Println("completed computing turn", currentTurn)

		//nextWorkerStateRequestWorker := -1
		//if gotStateRequest {
		//	sendState(workerId, currentTurn, wrappedGrid, distributor)
		//	nextWorkerStateRequestWorker = workerId
		//} else if stateRequestWorker > -1 {
		//	if stateRequestWorker != workerId {
		//		sendState(workerId, currentTurn, wrappedGrid, distributor)
		//		nextWorkerStateRequestWorker = stateRequestWorker
		//	}
		//}
		if gotStateRequest {
			exchangeEdges(workerBelow, wrappedGrid, workerId)
			sendState(workerId, currentTurn, wrappedGrid, distributor)
		} else {
			stateRequestWorker = exchangeEdges(workerBelow, wrappedGrid, -1)
			fmt.Println("STATEREQWORKER", stateRequestWorker)
			if stateRequestWorker > -1 && stateRequestWorker != workerId {
				sendState(workerId, currentTurn, wrappedGrid, distributor)
			} else {
				stateRequestWorker = -1
			}
		}

		gotStateRequest = false
		currentTurn++
		fmt.Println(currentTurn)
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
