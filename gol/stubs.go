package gol

import (
	"net"
	"net/rpc"
)

func serve(receiver interface{}, addr string) {
	rpc.Register(receiver)
	listener, _ := net.Listen("tcp", addr)
	defer listener.Close()
	rpc.Accept(listener)
}

// worker methods
var SetState = "Worker.SetState"
var SetRowAbove = "Worker.SetRowAbove"
var SetRowBelow = "Worker.SetRowBelow"
var PauseWorker = "Worker.Pause"
var GetWorkerState = "Worker.GetState"

// distributor methods
var SetWorkerState = "Distributor.SetWorkerState"
var GetImage = "Distributor.GetImage"
var PauseDistributor = "Distributor.Pause"
var SetInitialState = "Distributor.SetInitialState"

// controller methods
var CellsAlive = "Controller.CellsAlive"

type WorkerStateUpdate struct {
	workerId int
	turn     int
	state    [][]byte
}

type RowAboveUpdate struct {
	rowAbove           []byte
	stateRequestWorker int
}

type WorkerInitialState struct {
	workerId        int
	width           int
	height          int
	cells           [][]byte
	workerAboveAddr string
	workerBelowAddr string
	distributorAddr string
}

type DistributorInitialState struct {
	width          int
	height         int
	cells          [][]byte
	controllerAddr string
}
