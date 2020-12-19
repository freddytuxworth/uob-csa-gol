package gol

import (
	"fmt"
	"os"
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

func shouldSurvive(x int, y int, grid *stubs.Grid) byte {
	leftX := util.WrapNum(x-1, grid.Width)
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

func getAliveCells(w, h int, state [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if state[y][x] > 0 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

func countAliveCells(w, h int, state [][]byte) int {
	aliveCells := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			aliveCells += int(state[y][x])
		}
	}

	return aliveCells
}


func Run(p Params, events chan Event, keyPresses chan rune) {
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	go RunController(
		//os.Getenv("THIS_ADDR"),
		os.Getenv("DISTRIBUTOR_ADDR"),
		stubs.GolJob{
			Name:     filename,
			Filename: filename,
			Turns:    p.Turns,
		},
		keyPresses,
		events,
	)
}