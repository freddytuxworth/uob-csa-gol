package gol

import (
	"fmt"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
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