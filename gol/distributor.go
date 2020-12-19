package gol

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

func (g Grid) make() {
	g.Cells = make([][]byte, g.Height)
	for y := 0; y < g.Height; y++ {
		g.Cells[y] = make([]byte, g.Width)
	}
}

func (g Grid) countAlive() int {
	aliveCells := 0
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			aliveCells += int(g.Cells[y][x])
		}
	}

	return aliveCells
}

func (g Grid) getRow(y int) []byte {
	if y < 0 {
		y += g.Height
	}
	y %= g.Height
	return g.Cells[y]
}

func (g Grid) get(x, y int) byte {
	if x < 0 {
		x += g.Width
	}
	if y < 0 {
		y += g.Height
	}
	x %= g.Width
	y %= g.Height
	return g.Cells[y][x]
}

func (g Grid) set(x, y int, val byte) bool {
	if x < 0 {
		x += g.Width
	}
	if y < 0 {
		y += g.Height
	}
	x %= g.Width
	y %= g.Height

	if g.Cells[y][x] == val {
		return false
	}

	g.Cells[y][x] = val
	return true
}

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool

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
	topEdge := make([]byte, wrappedGrid.Width)
	copy(topEdge, wrappedGrid.Cells[1])
	c.topEdgeOut <- topEdge

	bottomEdge := make([]byte, wrappedGrid.Width)
	copy(bottomEdge, wrappedGrid.Cells[wrappedGrid.Height-2])
	c.bottomEdgeOut <- bottomEdge
}

func workerThread(threadNum int, initialState Grid, offsetY int, c workerChannels) {
	wrappedGrid := Grid{
		Width:  initialState.Width,
		Height: initialState.Height + 2,
		Cells:  make([][]byte, initialState.Height+2),
	}

	for y := 0; y < wrappedGrid.Height; y++ {
		wrappedGrid.Cells[y] = make([]byte, initialState.Width)
		if y > 0 && y < wrappedGrid.Height-1 {
			copy(wrappedGrid.Cells[y], initialState.Cells[y-1])
		}
	}

	sendEdges(wrappedGrid, c)

	for turn := 1; ; turn++ {
		// get top and bottom edges from workers above and below
		wrappedGrid.Cells[0] = <-c.topEdgeIn
		wrappedGrid.Cells[initialState.Height+1] = <-c.bottomEdgeIn

		cellFlips := make([]util.Cell, 0)
		for y := 1; y < initialState.Height+1; y++ {
			for x := 0; x < initialState.Width; x++ {
				nextCellState := shouldSurvive(x, y, wrappedGrid)
				if nextCellState != wrappedGrid.Cells[y][x] {
					cellFlips = append(cellFlips, util.Cell{
						X: x,
						Y: y + offsetY - 1,
					})
				}
			}
		}
		for _, cellFlip := range cellFlips {
			wrappedGrid.Cells[cellFlip.Y-offsetY+1][cellFlip.X] = 1 - wrappedGrid.Cells[cellFlip.Y-offsetY+1][cellFlip.X]
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
		fmt.Printf("worker %d: %#v\n\n", thread, workers[thread])
	}

	n := p.Threads
	thread := 0
	for i := 0; i < p.ImageHeight; {
		// calculate (p.ImageHeight - i) / n and round up
		size := int(math.Ceil(float64(p.ImageHeight-i) / float64(n)))
		n--
		go workerThread(thread, Grid{
			Width:  p.ImageWidth,
			Height: size,
			Cells:  currentState[i : i+size],
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
	io := StartIo()
	currentState := io.readImageToSlice(fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight))

	processor := GolProcessor{
		p:            Params{},
		currentState: &currentState,
	}
	stopProcessor := make(chan bool)
	processor.start(c.events, stopProcessor)

	ticker := time.NewTicker(2 * time.Second)

	select {
	case <-ticker.C:
		stopProcessor <- true
		c.events <- AliveCellsCount{
			CompletedTurns: processor.currentTurn,
			CellsCount:     processor.currentState.countAlive(),
		}
		stopProcessor <- true
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
		//fmt.Println(turn, countAliveCells(p, currentState))
		turn++
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, currentState),
	}
	writeStateToImage(p, currentState, c, p.Turns)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
