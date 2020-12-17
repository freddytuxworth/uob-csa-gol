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
func log(msg string) {
	fmt.Printf("%s%s\n",
		bold("worker %d @ turn %d: ", workerId, currentTurn),
		msg)
	//fmt.Sprintf("worker %d @ turn %d: %s\n", workerId, currentTurn, msg)
}

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
	log("SRA: did put setrowabove req")
	if workerId > 0 {
		bottomRow := <-bottomRowOut
		logf("SRA: did get from bottomRowOut, starting workerBelow call with %v", bottomRow)
		var response stubs.RowUpdate
		workerBelow.Call(stubs.SetRowAbove, bottomRow, &response)
		logf("SRA: done workerBelow call, got %v", response)
		rowBelowIn <- response
	}
	*res = <-topRowOut
	logf("SRA: did put setrowabove res: %v", *res)
	return
}

func (W *Worker) GetState(req bool, res *bool) (err error) {
	log("got state request")
	stateRequestChan <- true
	return
}

func sendEdges(wrappedGrid stubs.Grid, workerAbove, workerBelow *rpc.Client, makeStateRequest bool, ) {

}

func sendState(wrappedGrid stubs.Grid, distributor *rpc.Client) {
	log(color.GreenString("sending state"))
	distributor.Go(stubs.SetWorkerState, stubs.WorkerStateUpdate{
		WorkerId: workerId,
		Turn:     currentTurn,
		State:    wrappedGrid.Cells[1 : wrappedGrid.Height-1],
	}, nil, nil)
}
//
//func sendTurnUpdate(currentTurn int, distributor *rpc.Client) bool {
//	var shouldSendState bool
//	distributor.Call(stubs.WorkerTurnUpdate, stubs.WorkerStateUpdate{
//		WorkerId: workerId,
//		Turn:     currentTurn,
//	}, &shouldSendState)
//	return shouldSendState
//}

func startEdgeExchange(wrappedGrid stubs.Grid, stateRequest bool) {
	log("LEX: start")
	topRowOut <- stubs.RowUpdate{
		Row: wrappedGrid.Cells[1],
	}

	log("LEX: filled channel")
	bottomRow := stubs.RowUpdate{
		Row:             wrappedGrid.Cells[wrappedGrid.Height-2],
		ShouldSendState: stateRequest,
	}

	logf("LEX: starting workerBelow call with %v", bottomRow)
	var response stubs.RowUpdate
	workerBelow.Call(stubs.SetRowAbove, bottomRow, &response)
	logf("LEX: done workerBelow call, got %v", response)
	rowBelowIn <- response
}

//func exchangeEdges(workerBelow *rpc.Client, wrappedGrid stubs.Grid) {
//	//if initialSetup {
//	fmt.Println("beginning edge exchange")
//	topRowOut <- stubs.RowUpdate{
//		Row:             wrappedGrid.Cells[1],
//		//ShouldSendState: stateRequest,
//	}
//	fmt.Println("top row out sent")
//
//	rowAboveUpdate := <-rowAboveIn
//	fmt.Println("got row above", rowAboveUpdate)
//
//	//<-done
//	//Now we have row below
//
//	wrappedGrid.Cells[1] = rowAboveUpdate.Row
//	wrappedGrid.Cells[wrappedGrid.Height-1] = response.Row
//	fmt.Println("finished row exchange")
//}

func setupInitialState(initialState stubs.WorkerInitialState) stubs.Grid {
	logf("got initial state from distributor: %v", initialState)
	wrappedGrid := stubs.Grid{
		Width:  initialState.Grid.Width,
		Height: initialState.Grid.Height + 2,
		Cells:  make([][]byte, initialState.Grid.Height+2),
	}

	for y := 0; y < wrappedGrid.Height; y++ {
		wrappedGrid.Cells[y] = make([]byte, initialState.Grid.Width)
		if y > 0 && y < wrappedGrid.Height-1 {
			copy(wrappedGrid.Cells[y], initialState.Grid.Cells[y-1])
		}
	}
	workerBelow, _ = rpc.Dial("tcp", initialState.WorkerBelowAddr)
	distributor, _ = rpc.Dial("tcp", initialState.DistributorAddr)

	workerId = initialState.WorkerId
	currentTurn = 0
	if workerId == 0 {
		startEdgeExchange(wrappedGrid, false)
	} else {
		topRowOut <- stubs.RowUpdate{Row: wrappedGrid.Cells[1]}
		bottomRowOut <- stubs.RowUpdate{Row: wrappedGrid.Cells[wrappedGrid.Height-2]}
	}

	return wrappedGrid
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
			log("received state request in main loop")
		default:
		}

		rowAboveUpdate := <-rowAboveIn
		wrappedGrid.Cells[0] = rowAboveUpdate.Row
		if rowAboveUpdate.ShouldSendState {
			sendState(wrappedGrid, distributor)
		}

		rowBelowUpdate := <-rowBelowIn
		wrappedGrid.Cells[wrappedGrid.Height-1] = rowBelowUpdate.Row

		log("begin computing turn")
		logf("current channel states: %d, %d, %d, %d", len(topRowOut), len(bottomRowOut), len(rowAboveIn), len(rowBelowIn))
		logf("wrapped grid before computing:\n%v", wrappedGrid)
		for y := 1; y < wrappedGrid.Height-1; y++ {
			for x := 0; x < wrappedGrid.Width; x++ {
				wrappedGrid.Cells[y][x] = gol.ShouldSurvive(x, y, wrappedGrid)
			}
		}
		time.Sleep(500 * time.Millisecond)
		fmt.Println("completed computing turn", currentTurn)
		currentTurn++
		logf("wrapped grid after computing:\n%v", wrappedGrid)

		if workerId == 0 {
			if gotStateRequest {
				sendState(wrappedGrid, distributor)
			}
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
