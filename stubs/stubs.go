package stubs

import (
	"net"
	"net/rpc"
)

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

func Serve(receiver interface{}, addr string) {
	rpc.Register(&receiver)
	listener, _ := net.Listen("tcp", addr)
	defer listener.Close()
	rpc.Accept(listener)
}

// worker methods
var SetState = "Worker.SetState"
var SetRowAbove = "Worker.SetRowAbove"

//var SetRowBelow = "Worker.SetRowBelow"
var PauseWorker = "Worker.Pause"
var GetWorkerState = "Worker.GetState"

// distributor methods
var WorkerTurnUpdate = "Distributor.WorkerTurn"
var SetWorkerState = "Distributor.WorkerState"
var GetState = "Distributor.GetState"
var PauseDistributor = "Distributor.Pause"
var SetInitialState = "Distributor.SetInitialState"

// controller methods
var CellsAlive = "Controller.CellsAlive"

type WorkerStateUpdate struct {
	WorkerId int
	Turn     int
	State    [][]byte
}

type RowUpdate struct {
	Row             []byte
	ShouldSendState bool
}

//type RowAboveResponse struct {
//	RowBelow []byte
//}

type WorkerInitialState struct {
	WorkerId        int
	Width           int
	Height          int
	Cells           [][]byte
	WorkerAboveAddr string
	WorkerBelowAddr string
	DistributorAddr string
}

type DistributorInitialState struct {
	Width          int
	Height         int
	Cells          [][]byte
	ControllerAddr string
}

type OperationResult struct {
	success bool
}
