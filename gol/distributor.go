package gol

import (
	"fmt"
	"math"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Grid struct {
	width  int
	height int
	cells  [][]byte
}

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool
	ioParams  chan<- Params

	filename   chan string
	output     chan<- uint8
	input      <-chan uint8
	keypresses <-chan rune
}

type workerChannels struct {
	topEdgeIn  chan []byte
	topEdgeOut chan []byte

	bottomEdgeIn  chan []byte
	bottomEdgeOut chan []byte

	results chan []util.Cell
}

func sendEdges(wrappedGrid Grid, c workerChannels) {
	topEdge := make([]byte, wrappedGrid.width)
	copy(topEdge, wrappedGrid.cells[1])
	c.topEdgeOut <- topEdge

	bottomEdge := make([]byte, wrappedGrid.width)
	copy(bottomEdge, wrappedGrid.cells[wrappedGrid.height-2])
	c.bottomEdgeOut <- bottomEdge
}

func workerThread(threadNum int, initialState Grid, offsetY int, c workerChannels) {
	wrappedGrid := Grid{
		width:  initialState.width,
		height: initialState.height + 2,
		cells:  make([][]byte, initialState.height+2),
	}

	for y := 0; y < wrappedGrid.height; y++ {
		wrappedGrid.cells[y] = make([]byte, initialState.width)
		if y > 0 && y < wrappedGrid.height-1 {
			copy(wrappedGrid.cells[y], initialState.cells[y-1])
		}
	}

	sendEdges(wrappedGrid, c)

	for turn := 1; ; turn++ {
		// get top and bottom edges from workers above and below
		wrappedGrid.cells[0] = <-c.topEdgeIn
		wrappedGrid.cells[initialState.height+1] = <-c.bottomEdgeIn

		cellFlips := make([]util.Cell, 0)
		for y := 1; y < initialState.height+1; y++ {
			for x := 0; x < initialState.width; x++ {
				nextCellState := shouldSurvive(x, y, wrappedGrid)
				if nextCellState != wrappedGrid.cells[y][x] {
					cellFlips = append(cellFlips, util.Cell{
						X: x,
						Y: y + offsetY - 1,
					})
				}
			}
		}
		for _, cellFlip := range cellFlips {
			wrappedGrid.cells[cellFlip.Y-offsetY+1][cellFlip.X] = 1 - wrappedGrid.cells[cellFlip.Y-offsetY+1][cellFlip.X]
		}
		sendEdges(wrappedGrid, c)
		c.results <- cellFlips
	}

}

func startWorkers(p Params, currentState [][]byte) []workerChannels {
	workers := make([]workerChannels, p.Threads)
	for thread := 0; thread < p.Threads; thread++ {
		workers[thread] = workerChannels{
			topEdgeOut:    make(chan []byte, p.Threads),
			bottomEdgeOut: make(chan []byte, p.Threads),
			results:       make(chan []util.Cell, p.Threads),
		}
	}

	for thread := 0; thread < p.Threads; thread++ {
		workers[thread].topEdgeIn = workers[util.WrapNum(thread-1, p.Threads)].bottomEdgeOut
		workers[thread].bottomEdgeIn = workers[util.WrapNum(thread+1, p.Threads)].topEdgeOut
	}

	n := p.Threads
	thread := 0
	for i := 0; i < p.ImageHeight; {
		// calculate (p.ImageHeight - i) / n and round up
		size := int(math.Ceil(float64(p.ImageHeight-i) / float64(n)))
		n--
		go workerThread(thread, Grid{
			width:  p.ImageWidth,
			height: size,
			cells:  currentState[i : i+size],
		}, i, workers[thread])
		i += size
		thread++
	}

	return workers
}

func collateResults(p Params, currentState [][]byte, workers []workerChannels, events chan<- Event, turn int) {
	for i := 0; i < p.Threads; i++ {
		flippedCells := <-workers[i].results
		for _, cell := range flippedCells {
			events <- CellFlipped{
				CompletedTurns: turn,
				Cell:           cell,
			}
			currentState[cell.Y][cell.X] = 1 - currentState[cell.Y][cell.X]
		}
	}
}

func readImageToSlice(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- ioInput // send ioInput command to io goroutine
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

// send the current board data to the IO goroutine for output to an image
func writeStateToImage(p Params, currentState [][]byte, c distributorChannels, turn int) {
	c.ioCommand <- ioOutput
	c.filename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageHeight; x++ {
			c.output <- currentState[y][x]
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	fmt.Println("Distributing")
	c.ioParams <- p
	currentState := readImageToSlice(p, c)

	workers := startWorkers(p, currentState)
	ticker := time.NewTicker(2 * time.Second)

	for turn := 0; turn < p.Turns; {
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{
				CompletedTurns: turn,
				CellsCount:     countAliveCells(p, currentState),
			}
		case key := <-c.keypresses:
			switch key {
			case 'p':
				fmt.Println("Current turn:", turn)
				for {
					if <-c.keypresses == 'p' {
						break
					}
				}
			case 'q':
				writeStateToImage(p, currentState, c, turn)
				os.Exit(0)
			case 's':
				writeStateToImage(p, currentState, c, turn)
			}
		default:
			collateResults(p, currentState, workers, c.events, turn)
			c.events <- TurnComplete{turn + 1}
			fmt.Println(turn, countAliveCells(p, currentState))
			turn++
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, currentState),
	}
	//writeStateToImage(p, currentState, c, p.Turns)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
