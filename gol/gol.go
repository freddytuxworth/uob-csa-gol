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

func neighbors(x int, y int, p Params) []byte {
	fmt.Print("get neighbors for", x, y)
	fmt.Println(currentState)

	leftX := x - 1
	if x == 0 {
		leftX += p.ImageWidth
	}

	rightX := (x + 1) % p.ImageWidth

	upY := y - 1
	if y == 0 {
		upY += p.ImageHeight
	}

	downY := (y + 1) % p.ImageHeight

	return []byte{
		currentState[upY][leftX],
		currentState[upY][x],
		currentState[upY][rightX],

		currentState[y][leftX],
		currentState[y][rightX],

		currentState[downY][leftX],
		currentState[downY][x],
		currentState[downY][rightX]}
}

func shouldSurvive(x int, y int, p Params) byte {
	fmt.Println(currentState)
	return 0
	// var livingNeighbors byte = 0
	// for _, v := range neighbors(x, y, p) {
	// 	livingNeighbors += v / 255
	// }

	// if livingNeighbors < 2 {
	// 	return 0
	// } else if livingNeighbors == 2 {
	// 	return currentState[y][x]
	// } else if livingNeighbors == 3 {
	// 	return 255
	// } else {
	// 	return 0
	// }
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

func calculateAliveCells(p Params) []util.Cell {
	aliveCells := []util.Cell{}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if currentState[y][x] == 255 {
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
}
