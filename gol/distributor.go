package gol

import (
	"fmt"
	"os"
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

func printGrid(id int, offset int, grid Grid) {
	output := ""
	for y := 0; y < grid.height; y++ {
		row := ""
		for x := 0; x < grid.width; x++ {
			if grid.cells[y][x] > 0 {
				row += "#"
			} else {
				row += "."
			}
		}
		output += fmt.Sprintf("%02d | %02d: %s\n", id, y + offset, row)
	}
	fmt.Println(output)
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
		cells:  make([][]byte, initialState.height + 2),
	}

	for y:=0; y < wrappedGrid.height; y++ {
		wrappedGrid.cells[y] = make([]byte, initialState.width)
		if y > 0 && y < wrappedGrid.height - 1 {
			copy(wrappedGrid.cells[y], initialState.cells[y-1])
		}
	}

	sendEdges(wrappedGrid, c)

	for turn := 1;; turn++ {
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
			wrappedGrid.cells[cellFlip.Y - offsetY + 1][cellFlip.X] = 1 - wrappedGrid.cells[cellFlip.Y - offsetY + 1][cellFlip.X]
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
	stripHeight := p.ImageHeight / p.Threads
	for thread := 0; thread < p.Threads; thread++ {
		go workerThread(thread, Grid{
			width:  p.ImageWidth,
			height: stripHeight,
			cells:  currentState[thread*stripHeight : (thread+1)*stripHeight],
		}, thread*stripHeight, workers[thread])
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

func writeStateToImage(p Params, currentState [][]byte, c distributorChannels, turn int) {
	c.ioCommand <- ioOutput
	c.filename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageHeight; x++ {
			c.output <- currentState[y][x]
		}
	}
}

type turnUpdate struct {
	thread int
	turn int
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	currentState := readImageToSlice(p, c)
	//resultsChan := make(chan []util.Cell, p.Threads)


	//turnChan := make(chan turnUpdate, p.Threads)
	//turnStats := make([]int, p.Threads)
	//printGrid(-1, 0, Grid{
	//	width: p.ImageWidth,
	//	height: p.ImageHeight,
	//	cells: currentState,
	//})
	workers := startWorkers(p, currentState)
	//for turn := 0; turn < p.Turns; turn++ {
	//	collateResults(p, currentState, resultsChan, c.events, turn)
	//}
	ticker := time.NewTicker(2 * time.Second)

	for turn := 0; turn < p.Turns; {
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{
				CompletedTurns: turn,
				CellsCount:     countAliveCells(p, currentState),
			}
		//case t := <-turnChan:
		//	turnStats[t.thread] = t.turn
		//	out := ""
		//	for i, t := range turnStats {
		//		out += fmt.Sprintf("%02d %s\n", i, strings.Repeat(".", t))
		//	}
		//	fmt.Print(out)
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
