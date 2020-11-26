package gol

import (
	"fmt"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type sliceChan chan [][]byte

type workToDo struct {
	topY        int
	stripHeight int
	turn        int
}
type workChan chan workToDo

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool

	filename chan string
	output   <-chan uint8
	input    <-chan uint8
}

var currentState [][]byte
var nextState [][]byte

func workerThread(threadNumber int, p Params, workChannel workChan, events chan<- Event) {
	work := <-workChannel

	fmt.Println(threadNumber, "Work received: top", work.topY, "height", work.stripHeight)

	for y := work.topY; y < work.topY+work.stripHeight; y++ {
		// nextState[y] = make([]byte, p.ImageWidth)
		for x := 0; x < p.ImageWidth; x++ {
			nextCellState := shouldSurvive(x, y, p)
			if nextCellState != nextState[y][x] {
				// fmt.Print("cell changed state:", x, y)
				events <- CellFlipped{
					CompletedTurns: work.turn,
					Cell: util.Cell{
						X: x,
						Y: y,
					},
				}
			}
			nextState[y][x] = nextCellState
		}
	}

	// for y := 0; y < len(work); y++ {
	// 	fmt.Print(threadNumber, "")
	// 	for x := 0; x < len(work[y]); x++ {
	// 		if work[y][x] > 0 {
	// 			fmt.Print(".")
	// 		} else {
	// 			fmt.Print(" ")
	// 		}
	// 	}
	// 	fmt.Println("")
	// }
}

// divide worldSlice into 'thread' amount of strips
// add the row above and below the strip to the slice to allow for state changes at bottom and top of strip
// send each new slice down a threads channel, to be sent to workers
func splitAndSend(p Params, threads []workChan, turn int) {
	stripHeight := p.ImageHeight / p.Threads
	for strip := 0; strip < p.Threads; strip++ {
		threads[strip] <- workToDo{
			stripHeight: stripHeight,
			topY:        stripHeight * strip,
			turn:        turn,
		}
	}

	// topSlice := append(
	// 	[][]byte{worldSlice[p.ImageHeight-1]},
	// 	worldSlice[0:stripHeight+1]...)
	// threads[0] <- topSlice

	// time.Sleep(1 * time.Second)
	// for thread := 1; thread < p.Threads-1; thread++ {
	// 	stripTopY := stripHeight*thread - 1
	// 	threadSlice := worldSlice[stripTopY : stripTopY+stripHeight+2]
	// 	threads[thread] <- threadSlice
	// 	time.Sleep(1 * time.Second)
	// }

	// bottomSliceTopY := stripHeight*(p.Threads-1) - 1
	// bottomSlice := append(
	// 	worldSlice[bottomSliceTopY:bottomSliceTopY+stripHeight+1],
	// 	worldSlice[0])

	// threads[p.Threads-1] <- bottomSlice
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	fmt.Println("Distributor started")
	c.ioCommand <- 1 // send ioInput command to io goroutine
	c.filename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	//worldSlice := make([][]uint8)

	currentState := make([][]byte, p.ImageHeight)
	for y := range currentState {
		currentState[y] = make([]byte, p.ImageWidth)
		for x := range currentState[y] {
			currentState[y][x] = <-c.input
			if currentState[y][x] == 255 {
				// fmt.Print("(", x, y, ")")
				c.events <- CellFlipped{
					CompletedTurns: 1,
					Cell: util.Cell{
						X: x,
						Y: y,
					},
				}
			}
		}
	}

	//fmt.Println(worldSlice)

	nextState := make([][]byte, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		nextState[y] = make([]byte, p.ImageWidth)
		for x := 0; x < p.ImageWidth; x++ {
			nextState[y][x] = 0
		}
	}

	workerThreads := make([]workChan, p.Threads)
	for t := 0; t < p.Threads; t++ {
		workerThreads[t] = make(workChan, 0)
		go workerThread(t, p, workerThreads[t], c.events)
	}

	for turn := 0; turn < p.Turns; turn++ {
		splitAndSend(p, workerThreads, turn)
		// worldSlice = calculateNextState(p, worldSlice, c.events, turn)
		c.events <- TurnComplete{
			CompletedTurns: turn + 1,
		}
		// time.Sleep(1 * time.Second)
	}

	time.Sleep(2 * time.Second)

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p),
	}
	// TODO: Execute all turns of the Game of Life.
	// TODO: Send correct Events when required, e.g. CellFlipped, TurnComplete and FinalTurnComplete.
	//		 See event.go for a list of all events.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
