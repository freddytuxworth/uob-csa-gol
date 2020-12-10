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

var stateChan  = make(chan stubs.WorkerInitialState, 1)
var rowAboveChan = make(chan stubs.RowAboveUpdate, 1)
var rowBelowChan = make(chan []byte, 1)
var stateRequestChan = make(chan bool, 1)

func (w *Worker) SetState(req stubs.WorkerInitialState, res *bool) (err error) {
	fmt.Println("Got initial state", req)
	stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req stubs.RowAboveUpdate, res *bool) (err error) {
	rowAboveChan <- req
	return
}

func (w *Worker) SetRowBelow(req []byte, res *bool) (err error) {
	rowBelowChan <- req
	return
}

func (W *Worker) GetState(req bool, res *bool) (err error) {
	stateRequestChan <- true
	return
}

func sendEdges(workerId int, wrappedGrid stubs.Grid, workerAbove, workerBelow *rpc.Client, makeStateRequest bool,) {

}

func sendState(workerId, currentTurn int, wrappedGrid stubs.Grid, distributor *rpc.Client) {
	fmt.Println("Sending state")
	distributor.Go(stubs.SetWorkerState, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
		State:    wrappedGrid.Cells[1:wrappedGrid.Height-1],
	}, nil, nil)
}

func workerServer() {
	//distributor, _ := rpc.Dial("tcp", distributorAddr)

	var workerId int
	var wrappedGrid stubs.Grid

	var distributor *rpc.Client
	var workerAbove *rpc.Client
	var workerBelow *rpc.Client

	gotRowAbove := false
	gotRowBelow := false
	gotStateRequest := false
	stateRequestWorker := -1

	currentTurn := 0

	for {
		select {
		case initialState := <-stateChan:
			fmt.Println("Setting state")
			fmt.Println(initialState)
			workerId = initialState.WorkerId
			wrappedGrid = stubs.Grid{
				Width:  initialState.Width,
				Height: initialState.Height + 2,
				Cells:  make([][]byte, initialState.Height+2),
			}
			currentTurn = 0

			for y := 0; y < wrappedGrid.Height; y++ {
				wrappedGrid.Cells[y] = make([]byte, initialState.Width)
				if y > 0 && y < wrappedGrid.Height-1 {
					copy(wrappedGrid.Cells[y], initialState.Cells[y-1])
				}
			}
			workerAbove, _ = rpc.Dial("tcp", initialState.WorkerAboveAddr)
			workerBelow, _ = rpc.Dial("tcp", initialState.WorkerBelowAddr)
			distributor, _ = rpc.Dial("tcp", initialState.DistributorAddr)
			workerAbove.Go(stubs.SetRowBelow, wrappedGrid.Cells[1], nil, nil)
			workerBelow.Go(stubs.SetRowAbove, stubs.RowAboveUpdate{
				RowAbove:           wrappedGrid.Cells[wrappedGrid.Height - 2],
				StateRequestWorker: -1,
			}, nil, nil)

		case rowAbove := <-rowAboveChan:
			wrappedGrid.Cells[0] = rowAbove.RowAbove
			stateRequestWorker = rowAbove.StateRequestWorker
			gotRowAbove = true
		case rowBelow := <-rowBelowChan:
			wrappedGrid.Cells[wrappedGrid.Height-1] = rowBelow
			gotRowBelow = true
		case <-stateRequestChan:
			gotStateRequest = true
		}
		if gotRowAbove && gotRowBelow {
			for y := 1; y < wrappedGrid.Height-1; y++ {
				for x := 0; x < wrappedGrid.Width; x++ {
					wrappedGrid.Cells[y][x] = gol.ShouldSurvive(x, y, wrappedGrid)
				}
			}

			nextWorkerStateRequestWorker := -1
			if gotStateRequest {
				sendState(workerId, currentTurn, wrappedGrid, distributor)
				nextWorkerStateRequestWorker = workerId
			} else if stateRequestWorker > -1 {
				if stateRequestWorker != workerId {
					sendState(workerId, currentTurn, wrappedGrid, distributor)
					nextWorkerStateRequestWorker = stateRequestWorker
				}
			}
			workerAbove.Go(stubs.SetRowBelow, wrappedGrid.Cells[1], nil, nil)
			workerBelow.Go(stubs.SetRowAbove, stubs.RowAboveUpdate{
				RowAbove:           wrappedGrid.Cells[wrappedGrid.Height - 2],
				StateRequestWorker: nextWorkerStateRequestWorker,
			}, nil, nil)
			//distributor.Go(ChangeCells, cellFlips, nil, nil)
				//c.results <- cellFlips
			gotRowAbove = false
			gotRowBelow = false
			gotStateRequest = false
			currentTurn++
			time.Sleep(500 * time.Millisecond)
			fmt.Println(currentTurn)
		}
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