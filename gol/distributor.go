package gol

import (
	"fmt"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Grid struct {
	width  int
	height int
	cells  [][]byte
}

type Job struct {
	grid    Grid
	offsetY int
}

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool

	filename chan string
	output   <-chan uint8
	input    <-chan uint8
}

func workerThread(jobs <-chan Job, results chan<- []util.Cell) {
	for {
		job := <-jobs

		cellFlips := make([]util.Cell, 0)
		for y := 0; y < job.grid.height-2; y++ {
			for x := 0; x < job.grid.width; x++ {
				nextCellState := shouldSurvive(x, y+1, job.grid)
				if nextCellState != job.grid.cells[y+1][x] {
					cellFlips = append(cellFlips, util.Cell{
						X: x,
						Y: y + job.offsetY,
					})
				}
			}
		}
		results <- cellFlips
	}

}

// divide worldSlice into 'thread' amount of strips
// add the row above and below the strip to the slice to allow for state changes at bottom and top of strip
// send each new slice down a threads channel, to be sent to workers
func splitAndSend(p Params, currentState [][]byte, workers []chan Job) {
	stripHeight := p.ImageHeight / p.Threads

	for thread := 0; thread < p.Threads; thread++ {
		stripCells := make([][]byte, 0, stripHeight+2)
		stripTop := thread*stripHeight - 1
		for y := stripTop; y < stripTop+stripHeight+2; y++ {
			wrappedY := y % p.ImageHeight
			if wrappedY < 0 {
				wrappedY += p.ImageHeight
			}
			stripCells = append(stripCells, currentState[wrappedY])
		}

		workers[thread] <- Job{
			grid: Grid{
				width:  p.ImageWidth,
				height: stripHeight + 2,
				cells:  stripCells,
			},
			offsetY: thread * stripHeight,
		}
	}

}

func collateResults(p Params, currentState [][]byte, resultsChan chan []util.Cell, events chan<- Event, turn int) {
	flippedCells := make([]util.Cell, 0)
	for i := 0; i < p.Threads; i++ {
		flippedCells = append(flippedCells, <-resultsChan...)
	}

	for _, cell := range flippedCells {
		events <- CellFlipped{
			CompletedTurns: turn,
			Cell:           cell,
		}
		currentState[cell.Y][cell.X] = 1 - currentState[cell.Y][cell.X]
	}
}

func readImageToSlice(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- 1 // send ioInput command to io goroutine
	c.filename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	loadedCells := make([][]byte, p.ImageHeight)
	for y := range loadedCells {
		loadedCells[y] = make([]byte, p.ImageWidth)
		for x := range loadedCells[y] {
			if <-c.input > 0 {
				loadedCells[y][x] = 1
				c.events <- CellFlipped{
					CompletedTurns: 0,
					Cell:           util.Cell{X: x, Y: y},
				}
			}
		}
	}
	return loadedCells
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	currentState := readImageToSlice(p, c)
	workers := make([]chan Job, p.Threads)
	resultsChan := make(chan []util.Cell, p.Threads)

	for t := 0; t < p.Threads; t++ {
		workers[t] = make(chan Job)
		go workerThread(workers[t], resultsChan)
	}

	ticker := time.NewTicker(2 * time.Second)

	for turn := 0; turn < p.Turns; {
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{
				CompletedTurns: turn,
				CellsCount:     countAliveCells(p, currentState),
			}
		default:
			splitAndSend(p, currentState, workers)
			collateResults(p, currentState, resultsChan, c.events, turn)
			c.events <- TurnComplete{turn+1}
			//fmt.Println(turn, countAliveCells(p, currentState))
			turn++
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, currentState),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
