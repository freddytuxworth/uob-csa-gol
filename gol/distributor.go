package gol

import "fmt"

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool

	filename chan string
	output   <-chan uint8
	input    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	c.filename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- 1 // send ioInput command to io goroutine
	//worldSlice := make([][]uint8)
	worldSlice := make([][]uint8, p.ImageHeight)
	for y := range worldSlice {
		worldSlice[y] = make([]uint8, p.ImageWidth)
		for x := range worldSlice[y] {
			worldSlice[y][x] = <-c.input
		}
	}

	fmt.Println(worldSlice)
	// TODO: Create a 2D slice to store the world.
	// TODO: For all initially alive cells send a CellFlipped Event.

	turn := 0

	// TODO: Execute all turns of the Game of Life.
	// TODO: Send correct Events when required, e.g. CellFlipped, TurnComplete and FinalTurnComplete.
	//		 See event.go for a list of all events.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
