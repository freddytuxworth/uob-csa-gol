package stubs

import (
	"fmt"
	"log"
	"math"
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
	client, err := rpc.Dial("tcp", c.Addr)
	util.Check(err)
	c.Client = client
}

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

type EncodedGrid struct {
	Width  int
	Height int
	Data   []byte
}

func (g Grid) Encode() EncodedGrid {
	bytes := make([]byte, 0)
	for _, row := range g.Cells {
		encodedRow := make([]byte, g.Width / 8)
		BitsToBytes(g.Width, row, encodedRow)
		bytes = append(bytes, encodedRow...)
	}
	return EncodedGrid{
		Width:  g.Width,
		Height: g.Height,
		Data:   bytes,
	}
}

func (encoded EncodedGrid) Decode() Grid {
	//dataBits := BytesToBits(encoded.Data)
	cells := make([][]byte, encoded.Height)
	for y := 0; y < encoded.Height; y++ {
		cells[y] = make([]byte, encoded.Width)
		BytesToBits(encoded.Width, encoded.Data[y*encoded.Width/8:(y+1)*encoded.Width/8], cells[y])
		//cells[y] = (dataBits)[y*encoded.Width : (y+1)*encoded.Width]
	}
	return Grid{
		Width:  encoded.Width,
		Height: encoded.Height,
		Cells:  cells,
	}
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
	WorkerId        int
	CurrentTurn     int
	AliveCellsCount int
	State           Grid
}

func (i InstructionResult) Encode() EncodedInstructionResult {
	return EncodedInstructionResult{
		WorkerId:        i.WorkerId,
		CurrentTurn:     i.CurrentTurn,
		AliveCellsCount: i.AliveCellsCount,
		State:           i.State.Encode(),
	}
}

type EncodedInstructionResult struct {
	WorkerId        int
	CurrentTurn     int
	AliveCellsCount int
	State           EncodedGrid
}

func (encoded EncodedInstructionResult) Decode() InstructionResult {
	return InstructionResult{
		WorkerId:        encoded.WorkerId,
		CurrentTurn:     encoded.CurrentTurn,
		AliveCellsCount: encoded.AliveCellsCount,
		State:           encoded.State.Decode(),
	}
}

type Instruction byte

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

type EncodedRowUpdate struct {
	Length int
	Data   []byte
	Instruction Instruction
}

type RowUpdate struct {
	//Turn         int
	Row         []byte
	Instruction Instruction
}

func (r RowUpdate) String() string {
	return fmt.Sprintf("%v, %v", r.Row, r.Instruction)
}

func BitsToBytes(n int, bits, out []byte) {
	numBytes := int(math.Ceil(float64(n)/8))
	for i := 0; i < numBytes; i++ {
		out[i] = 0
	}
	//out = make([]byte, length/8)
	for i := 0; i < n; i += 1 {
		out[i/8] += bits[i] << byte(i%8)
	}
}

func BytesToBits(n int, bytes, out []byte) {
	//out := make([]byte, length)
	for i := 0; i < n; i ++ {
		out[i] = (bytes[i/8] >> byte(i%8)) & 1
	}
	//return out
}

func (r RowUpdate) Encode() EncodedRowUpdate {
	length := len(r.Row)
	bytes := make([]byte, length)
	BitsToBytes(length, r.Row, bytes)
	return EncodedRowUpdate{
		Length: len(r.Row),
		Data:   append(bytes),
		Instruction: r.Instruction,
	}
}

//func (encoded EncodedRowUpdate) Decode() *RowUpdate {
//	return &RowUpdate{
//		Instruction: Instruction(encoded.Data[encoded.Length/8]),
//		Row:         BytesToBits(encoded.Data[:encoded.Length/8]),
//	}
//}

func (encoded EncodedRowUpdate) DecodeInto(out []byte) Instruction {
	BytesToBits(encoded.Length, encoded.Data, out)
	return encoded.Instruction
	//return &RowUpdate{
	//	Instruction: Instruction(encoded.Data[encoded.Length/8]),
	//	Row:         BytesToBits(encoded.Data[:encoded.Length/8]),
	//}
}

type WorkerInitialState struct {
	WorkerId        int
	JobName         string
	Turns           int
	Grid            EncodedGrid
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
	JobName string
	Grid    EncodedGrid
	Turns   int
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
