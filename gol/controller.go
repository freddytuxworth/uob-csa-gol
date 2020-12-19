package gol

import (
	"fmt"
	"os"
	"strings"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Grid struct {
	Width  int
	Height int
	Cells  [][]byte
}

func (g *Grid) String() string {
	//return fmt.Sprintf("Grid(%dx%d)", g.Width, g.Height)
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

func (g *Grid) nextState(x, y int) byte {
	leftX := util.WrapNum(x-1, g.Width)
	rightX := (x + 1) % g.Width
	upY := util.WrapNum(y-1, g.Height)
	downY := (y + 1) % g.Height

	livingNeighbors :=
		g.Cells[upY][leftX] +
			g.Cells[upY][x] +
			g.Cells[upY][rightX] +
			g.Cells[y][leftX] +
			g.Cells[y][rightX] +
			g.Cells[downY][leftX] +
			g.Cells[downY][x] +
			g.Cells[downY][rightX]

	if livingNeighbors == 2 {
		return g.Cells[y][x]
	} else if livingNeighbors == 3 {
		return 1
	}

	return 0
}

func (g *Grid) make() {
	g.Cells = make([][]byte, g.Height)
	for y := 0; y < g.Height; y++ {
		g.Cells[y] = make([]byte, g.Width)
	}
}

func (g *Grid) countAlive() int {
	aliveCells := 0
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			aliveCells += int(g.Cells[y][x])
		}
	}

	return aliveCells
}

func (g *Grid) getAlive() []util.Cell {
	var aliveCells []util.Cell
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			if g.Cells[y][x] > 0 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

func (g *Grid) getRow(y int) []byte {
	if y < 0 {
		y += g.Height
	}
	y %= g.Height
	return g.Cells[y]
}

func (g *Grid) get(x, y int) byte {
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
	events     chan<- Event
	keypresses <-chan rune
}

func controller(p Params, c distributorChannels) {
	io := StartIo()
	initialState := io.readImageToSlice(fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight))
	for _, cell := range initialState.getAlive() {
		c.events <- CellFlipped{
			CompletedTurns: 0,
			Cell:           cell,
		}
	}

	processor := GolProcessor{
		p:            p,
		currentState: &initialState,
	}

	processorFinished := make(chan bool, 1)
	go processor.start(c.events, processorFinished)

	ticker := time.NewTicker(2 * time.Second)

	for {
		isFinished := false
		select {
		case isFinished = <-processorFinished:
			fmt.Println("procfin")
		case <-ticker.C:
			processor.pause()
			c.events <- AliveCellsCount{
				CompletedTurns: processor.currentTurn,
				CellsCount:     processor.currentState.countAlive(),
			}
			processor.pause()
		case key := <-c.keypresses:
			switch key {
			case 'p':
				processor.pause()
				fmt.Println("pausing, current turn:", processor.currentTurn)
				for {
					if <-c.keypresses == 'p' {
						processor.pause()
						break
					}
				}
			case 'q':
				processor.pause()
				io.writeStateToImage(*processor.currentState, fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, processor.currentTurn))
				os.Exit(0)
			case 's':
				io.writeStateToImage(*processor.currentState, fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, processor.currentTurn))
			}
		}
		if isFinished {
			break
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          processor.currentState.getAlive(),
	}
	io.writeStateToImage(*processor.currentState, fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns))

	io.waitUntilFinished()

	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
