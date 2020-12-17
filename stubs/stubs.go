package stubs

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"
	"uk.ac.bris.cs/gameoflife/util"
)

type Remote struct {
	*rpc.Client
	Addr   string
}

func (c *Remote) Connect() {
	client, err := rpc.Dial("tcp", c.Addr)
	util.Check(err)
	c.Client = client
}

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
var GetWorkerState = "Worker.GetState"

// distributor methods
var SetWorkerState = "Distributor.WorkerState"
var GetState = "Distributor.GetState"
var SetInitialState = "Distributor.SetInitialState"

type InstructionResult struct {
	WorkerId        int
	CurrentTurn     int
	AliveCellsCount int
	State           Grid
}

type Instruction uint8

const (
	GetCurrentTurn     Instruction = 1
	GetWholeState      Instruction = 2
	GetAliveCellsCount Instruction = 4
	Pause              Instruction = 8
	Resume             Instruction = 16
	Shutdown           Instruction = 32
)

func (s Instruction) HasFlag(flag Instruction) bool {
	return s&flag != 0
}

type RowUpdate struct {
	Row          []byte
	StateRequest Instruction
}

func (r RowUpdate) String() string {
	return fmt.Sprintf("%v, %v", r.Row, r.StateRequest)
}

type WorkerInitialState struct {
	WorkerId        int
	MaxTurns        int
	Grid            Grid
	WorkerBelowAddr string
	DistributorAddr string
}

func (s WorkerInitialState) String() string {
	return fmt.Sprintf("id: %d, size: %dx%d, worker below: %s, distributor: %s\n%v", s.WorkerId, s.Grid.Width, s.Grid.Height, s.WorkerBelowAddr, s.DistributorAddr, s.Grid)
}

type DistributorInitialState struct {
	Grid           Grid
	MaxTurns       int
	ControllerAddr string
}

func (s DistributorInitialState) String() string {
	return fmt.Sprintf("size: %dx%d, controller: %s\n%v", s.Grid.Width, s.Grid.Height, s.Grid)
}
