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
	RunController(
		os.Getenv("THIS_ADDR"),
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

//// Run starts the processing of Game of Life. It should initialise channels and goroutines.
//func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
//	IoCommand := make(chan IoCommand)
//	ioIdle := make(chan bool)
//	ioFilename := make(chan string)
//	ioInput := make(chan uint8)
//	ioOutput := make(chan uint8)
//
//	distributorChannels := distributorChannels{
//		events,
//		IoCommand,
//		ioIdle,
//		ioFilename,
//		ioOutput,
//		ioInput,
//		keyPresses,
//	}
//	go distributor(p, distributorChannels)
//
//	ioChannels := IoChannels{
//		Command:  IoCommand,
//		idle:     ioIdle,
//		filename: ioFilename,
//		output:   ioOutput,
//		input:    ioInput,
//	}
//	go startIo(p, IoChannels)
//	//time.Sleep(100 * time.Second)
//}
