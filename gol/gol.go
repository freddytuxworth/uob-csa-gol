package gol

import (
	"fmt"
	"uk.ac.bris.cs/gameoflife/util"
)

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

func shouldSurvive(x int, y int, grid Grid) byte {
	leftX := x - 1
	if x == 0 {
		leftX += grid.width
	}

	rightX := (x + 1) % grid.width

	livingNeighbors :=
		grid.cells[y-1][leftX] +
			grid.cells[y-1][x] +
			grid.cells[y-1][rightX] +
			grid.cells[y][leftX] +
			grid.cells[y][rightX] +
			grid.cells[y+1][leftX] +
			grid.cells[y+1][x] +
			grid.cells[y+1][rightX]

	if livingNeighbors == 2 {
		return grid.cells[y][x]
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

func SetupIO(events chan<- Event, keyPresses <-chan rune) distributorChannels {
	fmt.Println("IO")
	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	ioParams := make(chan Params)
	ioFilename := make(chan string)
	ioInput := make(chan uint8)
	ioOutput := make(chan uint8)

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		params:   ioParams,
		filename: ioFilename,
		output:   ioOutput,
		input:    ioInput,
	}
	go startIo(ioChannels)

	return distributorChannels{
		events:     events,
		ioCommand:  ioCommand,
		ioIdle:     ioIdle,
		ioParams:   ioParams,
		filename:   ioFilename,
		output:     ioOutput,
		input:      ioInput,
		keypresses: keyPresses,
	}
}

func RunWithIO(p Params, channels distributorChannels) {
	go distributor(p, channels)
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
	distributorChannels := SetupIO(events, keyPresses)
	go distributor(p, distributorChannels)
	//time.Sleep(100 * time.Second)
}
