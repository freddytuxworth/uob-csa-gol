package gol

import (
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

func ShouldSurvive(x int, y int, grid stubs.Grid) byte {
	leftX := x - 1
	if x == 0 {
		leftX += grid.Width
	}

	rightX := (x + 1) % grid.Width

	livingNeighbors :=
		grid.Cells[y-1][leftX] +
			grid.Cells[y-1][x] +
			grid.Cells[y-1][rightX] +
			grid.Cells[y][leftX] +
			grid.Cells[y][rightX] +
			grid.Cells[y+1][leftX] +
			grid.Cells[y+1][x] +
			grid.Cells[y+1][rightX]

	if livingNeighbors == 2 {
		return grid.Cells[y][x]
	} else if livingNeighbors == 3 {
		return 1
	}

	return 0
}

func countAliveCells(p Params, state [][]byte) int {
	aliveCells := 0
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			aliveCells += int(state[y][x])
		}
	}

	return aliveCells
}

func calculateAliveCells(p Params, state [][]byte) []util.Cell {
	aliveCells := []util.Cell{}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if state[y][x] > 0 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

//// Run starts the processing of Game of Life. It should initialise channels and goroutines.
//func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
//	ioCommand := make(chan ioCommand)
//	ioIdle := make(chan bool)
//	ioFilename := make(chan string)
//	ioInput := make(chan uint8)
//	ioOutput := make(chan uint8)
//
//	distributorChannels := distributorChannels{
//		events,
//		ioCommand,
//		ioIdle,
//		ioFilename,
//		ioOutput,
//		ioInput,
//		keyPresses,
//	}
//	go distributor(p, distributorChannels)
//
//	ioChannels := ioChannels{
//		command:  ioCommand,
//		idle:     ioIdle,
//		filename: ioFilename,
//		output:   ioOutput,
//		input:    ioInput,
//	}
//	go startIo(p, ioChannels)
//	//time.Sleep(100 * time.Second)
//}
