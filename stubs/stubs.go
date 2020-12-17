package stubs

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"
)

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

func (g Grid) String() string {
	var output []string
	output = append(output, strings.Repeat("──", g.Width))
	for y := 0; y < g.Height; y++ {
		rowSum := 0
		line := ""
		for x := 0; x < g.Width; x++ {
			if g.Cells[y][x] > 0 {
				line += "██"
			} else {
				line += "  "
			}
			rowSum += int(1 << uint8(x) * g.Cells[y][x])
		}
		output = append(output, fmt.Sprintf("%s %d", line, rowSum))
	}
	output = append(output, strings.Repeat("──", g.Width), "")
	return strings.Join(output, "\n")
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
var PauseWorker = "Worker.Pause"
var GetWorkerState = "Worker.GetState"

// distributor methods
var SetWorkerState = "Distributor.WorkerState"
var GetState = "Distributor.GetState"
var PauseDistributor = "Distributor.Pause"
var SetInitialState = "Distributor.SetInitialState"
var GetCellsAlive = "Distributor.GetCellsAlive"

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

func (r RowUpdate) String() string {
	return fmt.Sprintf("%v, %v", r.Row, r.ShouldSendState)
}

//type RowAboveResponse struct {
//	RowBelow []byte
//}

type WorkerInitialState struct {
	WorkerId        int
	Grid            Grid
	WorkerAboveAddr string
	WorkerBelowAddr string
	DistributorAddr string
}

func (s WorkerInitialState) String() string {
	return fmt.Sprintf("id: %d, size: %dx%d, worker below: %s, distributor: %s\n%v", s.WorkerId, s.Grid.Width, s.Grid.Height, s.WorkerBelowAddr, s.DistributorAddr, s.Grid)
}

type DistributorInitialState struct {
	Grid           Grid
	ControllerAddr string
}

func (s DistributorInitialState) String() string {
	return fmt.Sprintf("size: %dx%d, controller: %s\n%v", s.Grid.Width, s.Grid.Height, s.ControllerAddr, s.Grid)
}

type OperationResult struct {
	success bool
}
