package gol

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
	distributorChannels := distributorChannels{
		events,
		keyPresses,
	}
	go controller(p, distributorChannels)
	//time.Sleep(100 * time.Second)
}
