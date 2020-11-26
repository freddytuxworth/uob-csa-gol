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

func neighbors(x int, y int, grid Grid) []byte {
	leftX := x - 1
	if x == 0 {
		leftX += grid.width
	}

	rightX := (x + 1) % grid.width

	upY := y - 1
	if y == 0 {
		upY += grid.height
	}

	downY := (y + 1) % grid.height

	return []byte{
		grid.cells[upY][leftX],
		grid.cells[upY][x],
		grid.cells[upY][rightX],

		grid.cells[y][leftX],
		grid.cells[y][rightX],

		grid.cells[downY][leftX],
		grid.cells[downY][x],
		grid.cells[downY][rightX]}
}

func shouldSurvive(x int, y int, grid Grid) byte {
	var livingNeighbors byte = 0
	for _, v := range neighbors(x, y, grid) {
		livingNeighbors += v / 255
	}

	if livingNeighbors < 2 {
		return 0
	} else if livingNeighbors == 2 {
		return grid.cells[y][x]
	} else if livingNeighbors == 3 {
		return 255
	} else {
		return 0
	}
}

// func calculateNextState(p Params, world [][]byte, events chan<- Event, turn int) [][]byte {
// 	nextState := make([][]byte, p.ImageHeight)

// 	for y := 0; y < p.ImageHeight; y++ {
// 		nextState[y] = make([]byte, p.ImageWidth)
// 		for x := 0; x < p.ImageWidth; x++ {
// 			nextCellState := shouldSurvive(x, y, world, p)
// 			if nextCellState != nextState[y][x] {
// 				// fmt.Print("cell changed state:", x, y)
// 				events <- CellFlipped{
// 					CompletedTurns: turn,
// 					Cell: util.Cell{
// 						X: x,
// 						Y: y,
// 					},
// 				}
// 			}
// 			nextState[y][x] = nextCellState
// 		}
// 	}

// 	return nextState
// }

// func stripNextState(p Params, strip [][]byte, stripHeight int, events chan<- Event, turn int) [][]byte {
// 	nextState := make([][]byte, stripHeight)

// 	for y := 0; y < stripHeight; y++ {
// 		nextState[y] = make([]byte, p.ImageWidth)
// 		for x := 0; x < p.ImageWidth; x++ {
// 			nextCellState := shouldSurvive(x, y+1, strip, p)
// 			if nextCellState != nextState[y][x] {
// 				// fmt.Print("cell changed state:", x, y)
// 				events <- CellFlipped{
// 					CompletedTurns: turn,
// 					Cell: util.Cell{
// 						X: x,
// 						Y: y,
// 					},
// 				}
// 			}
// 			nextState[y][x] = nextCellState
// 		}
// 	}

// 	return nextState
// }

func calculateAliveCells(p Params, state [][]byte) []util.Cell {
	aliveCells := []util.Cell{}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if state[y][x] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
	fmt.Println("TEST")
	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	ioFilename := make(chan string)
	ioInput := make(chan uint8)
	ioOutput := make(chan uint8)

	distributorChannels := distributorChannels{
		events,
		ioCommand,
		ioIdle,
		ioFilename,
		ioOutput,
		ioInput,
	}
	go distributor(p, distributorChannels)

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: ioFilename,
		output:   ioOutput,
		input:    ioInput,
	}
	go startIo(p, ioChannels)
	//time.Sleep(100 * time.Second)
}
