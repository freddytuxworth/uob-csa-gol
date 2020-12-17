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

var workerId int
var currentTurn int
var workerBelow *rpc.Client

var stateChan = make(chan stubs.WorkerInitialState, 1)
var stateRequestChan = make(chan bool, 1)

var rowAboveIn = make(chan stubs.RowUpdate, 1)
var rowBelowIn = make(chan stubs.RowUpdate, 1)
var topRowOut = make(chan stubs.RowUpdate, 1)
var bottomRowOut = make(chan stubs.RowUpdate, 1)

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowUpdate, res *stubs.RowUpdate) (err error) {
	rowAboveIn <- req
	*res = <-topRowOut
	if workerId > 0 {
		bottomRow := <-bottomRowOut
		var response stubs.RowUpdate
		workerBelow.Call(stubs.SetRowAbove, bottomRow, &response)
		fmt.Println("done workerBelow call, got", response.Row)
		rowBelowIn <- response
	}
	return
}

func (W *Worker) GetState(req bool, res *bool) (err error) {
	stateRequestChan <- true
	return
}

func sendEdges(wrappedGrid stubs.Grid, workerAbove, workerBelow *rpc.Client, makeStateRequest bool, ) {

}

func sendState(wrappedGrid stubs.Grid, distributor *rpc.Client) {
	fmt.Println("Sending state")
	distributor.Go(stubs.SetWorkerState, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
		State:    wrappedGrid.Cells[1 : wrappedGrid.Height-1],
	}, nil, nil)
}

func sendTurnUpdate(currentTurn int, distributor *rpc.Client) bool {
	var shouldSendState bool
	distributor.Call(stubs.WorkerTurnUpdate, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
	}, &shouldSendState)
	return shouldSendState
}

func exchangeEdges(workerBelow *rpc.Client, wrappedGrid stubs.Grid) {
	//if initialSetup {
	fmt.Println("beginning edge exchange")
	topRowOut <- stubs.RowUpdate{
		Row:             wrappedGrid.Cells[1],
		//ShouldSendState: stateRequest,
	}
	fmt.Println("top row out sent")


	rowAboveUpdate := <-rowAboveIn
	fmt.Println("got row above", rowAboveUpdate)

	//<-done
	//Now we have row below

	wrappedGrid.Cells[1] = rowAboveUpdate.Row
	wrappedGrid.Cells[wrappedGrid.Height-1] = response.Row
	fmt.Println("finished row exchange")
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

	var wrappedGrid stubs.Grid

	var distributor *rpc.Client
	//var workerBelow *rpc.Client

	currentTurn := 0

	workerId, wrappedGrid, workerBelow, distributor = setupInitialState(<-stateChan)
	exchangeEdges(workerBelow, wrappedGrid)

	for {
		select {
		case initialState := <-stateChan:
			workerId, wrappedGrid, workerBelow, distributor = setupInitialState(initialState)
			currentTurn = 0
			exchangeEdges(workerBelow, wrappedGrid)
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

		exchangeEdges(workerBelow, wrappedGrid)

		//if sendTurnUpdate(workerId, currentTurn, distributor) {
			//sendState(workerId, currentTurn, wrappedGrid, distributor)
		//}
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
