package stubs

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"uk.ac.bris.cs/gameoflife/util"
)

type GolJob struct {
	Name     string
	Filename string
	Turns    int
}

func ServeHTTP(port int) {
	fmt.Printf("listening on :%d\n", port)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

type Remote struct {
	*rpc.Client
	Addr string
}

func (c *Remote) Connect() {
	client, err := rpc.DialHTTP("tcp", c.Addr)
	util.Check(err)
	c.Client = client
}

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

func (g Grid) String() string {
	return fmt.Sprintf("Grid(%dx%d)", g.Width, g.Height)
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

//func Serve(receiver interface{}, addr string) {
//	rpc.Register(&receiver)
//	listener, _ := net.Listen("tcp", addr)
//	defer listener.Close()
//	rpc.Accept(listener)
//}

// worker methods
//var SetState = "Worker.SetState"
//var SetRowAbove = "Worker.SetRowAbove"

// distributor methods
//var SetWorkerState = "Distributor.WorkerState"
//var GetState = "Distributor.GetState"
//var SetInitialState = "Distributor.SetInitialState"

type InstructionResult struct {
	//Instruction     Instruction
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
	//Turn         int
	Row         []byte
	Instruction Instruction
}

func (r RowUpdate) String() string {
	return fmt.Sprintf("%v, %v", r.Row, r.Instruction)
}

type WorkerInitialState struct {
	WorkerId        int
	JobName         string
	Turns           int
	Grid            Grid
	WorkerBelowAddr string
	DistributorAddr string
}

func (s WorkerInitialState) String() string {
	return fmt.Sprintf(
		"worker id: %d, job: %s, size: %dx%d, turns: %d, worker below: %s, distributor: %s\n%v",
		s.WorkerId,
		s.JobName,
		s.Grid.Width,
		s.Grid.Height,
		s.Turns,
		s.WorkerBelowAddr,
		s.DistributorAddr,
		s.Grid,
	)
}

type DistributorInitialState struct {
	JobName        string
	Grid           Grid
	Turns          int
	//ControllerAddr string
}

func (s DistributorInitialState) String() string {
	return fmt.Sprintf(
		"job: %s, size: %dx%d, turns: %d\n%v",
		s.JobName,
		s.Grid.Width,
		s.Grid.Height,
		s.Turns,
		//s.ControllerAddr,
		s.Grid,
	)
}
