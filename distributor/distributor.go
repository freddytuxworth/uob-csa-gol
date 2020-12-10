package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"net/rpc"
	"strings"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)


type workerConnection struct {
	client *rpc.Client
	addr   string
}

type strip struct {
	top    int
	height int
}

var imageChan = make(chan []byte, 1)
var imageRequestChan = make(chan bool, 1)
var initialStateChan = make(chan stubs.DistributorInitialState, 2)
var workerStateUpdates = make(chan stubs.WorkerStateUpdate, 2)

type Distributor struct{}

func (d *Distributor) GetImage(req bool, res *[]byte) (err error) {
	imageRequestChan <- true
	*res = <-imageChan
	return
}

func (d *Distributor) SetInitialState(req stubs.DistributorInitialState, res *bool) (err error) {
	initialStateChan <- req
	return
}

func (d *Distributor) WorkerState(req stubs.WorkerStateUpdate, res *bool) (err error) {
	fmt.Println("Got worker state", req)
	workerStateUpdates <- req
	return
}

func makeStrips(totalHeight, numStrips int) []strip {
	result := make([]strip, 0, numStrips)
	thread := 0
	for i := 0; i < totalHeight; thread++ {
		// calculate (p.ImageHeight - i) / n and round up
		size := int(math.Ceil(float64(totalHeight-i) / float64(numStrips-thread)))
		result = append(result, strip{top: i, height: size})
		i += size
	}
	return result
}

func startWorkers(p gol.Params, currentState [][]byte, thisAddr string, workers []workerConnection) []strip {
	numWorkers := len(workers)
	strips := makeStrips(p.ImageHeight, numWorkers)
	for i, worker := range workers {
		worker.client.Go(stubs.SetState, stubs.WorkerInitialState{
			WorkerId:        i,
			Width:           p.ImageWidth,
			Height:          strips[i].height,
			Cells:           currentState[strips[i].top : strips[i].top+strips[i].height],
			WorkerAboveAddr: workers[util.WrapNum(i-1, numWorkers)].addr,
			WorkerBelowAddr: workers[util.WrapNum(i+1, numWorkers)].addr,
			DistributorAddr: thisAddr,
		}, nil, nil)
		fmt.Println("sent state to worker", i)
	}
	return strips
}

func fetchState(workerStrips []strip, currentState [][]byte, workers []workerConnection) {
	fmt.Println("Fetching state")
	workers[0].client.Call(stubs.GetWorkerState, false, nil)
	for i:=0;i<len(workers);i++ {
		workerState := <-workerStateUpdates
		fmt.Println(workerState.Turn)
		workerTop := workerStrips[workerState.WorkerId].top
		workerBottom := workerTop + workerStrips[workerState.WorkerId].height
		workerSection := currentState[workerTop:workerBottom]
		copy(workerSection, workerState.State)
	}
	fmt.Println("GOT COMPLETE STATE")
	fmt.Println(currentState)
	//imageChan <- make([]byte, 2)
}

func runDistributor(thisAddr string, workerAddrs []string) {
	var p gol.Params

	p.Threads = len(workerAddrs)
	workers := make([]workerConnection, p.Threads)
	for i, workerAddr := range workerAddrs {
		client, err := rpc.Dial("tcp", workerAddr)
		if err != nil {
			panic(fmt.Sprintf("could not connect to workerConnection at %s", workerAddr))
		}
		workers[i] = workerConnection{
			client: client,
			addr:   workerAddr,
		}
	}

	fmt.Println(workers)

	//var width, height int

	var workerStrips []strip
	var currentState [][]byte
	for {
		select {
		case initialState := <-initialStateChan:
			fmt.Println("Setting Initial State")
			fmt.Println(initialState)
			currentState = initialState.Cells
			p.ImageWidth = initialState.Width
			p.ImageHeight = initialState.Height
			workerStrips = startWorkers(p, initialState.Cells, thisAddr, workers)
		case <-imageRequestChan:
			fetchState(workerStrips, currentState, workers)
			//sendImage(state)
		}
	}
}

func main() {
	thisAddr := flag.String("ip", "127.0.0.1:8020", "IP and port to listen on")
	workerAddrs := flag.String("workers", "127.0.0.1:8030", "Address of broker instance")
	flag.Parse()

	rpc.Register(&Distributor{})
	listener, _ := net.Listen("tcp", *thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	//go stubs.Serve(Distributor{}, *thisAddr)
	runDistributor(*thisAddr, strings.Split(*workerAddrs, ","))
}

//
//type distributorChannels struct {
//	events    chan<- Event
//	ioCommand chan<- ioCommand
//	ioIdle    <-chan bool
//
//	filename   chan string
//	output     chan<- uint8
//	input      <-chan uint8
//	keypresses <-chan rune
//}
//
//type workerChannels struct {
//	topEdgeIn  chan []byte
//	topEdgeOut chan []byte
//
//	bottomEdgeIn  chan []byte
//	bottomEdgeOut chan []byte
//
//	results chan []util.Cell
//}
//
//func startWorkers(p Params, currentState [][]byte) []workerChannels {
//	workers := make([]workerChannels, p.Threads)
//	for thread := 0; thread < p.Threads; thread++ {
//		workers[thread] = workerChannels{
//			topEdgeOut:    make(chan []byte, p.Threads),
//			bottomEdgeOut: make(chan []byte, p.Threads),
//			results:       make(chan []util.Cell, p.Threads),
//		}
//	}
//
//	for thread := 0; thread < p.Threads; thread++ {
//		workers[thread].topEdgeIn = workers[util.WrapNum(thread-1, p.Threads)].bottomEdgeOut
//		workers[thread].bottomEdgeIn = workers[util.WrapNum(thread+1, p.Threads)].topEdgeOut
//		fmt.Printf("workerConnection %d: %#v\n\n", thread, workers[thread])
//	}
//
//	n := p.Threads
//	thread := 0
//	for i := 0; i < p.ImageHeight; {
//		// calculate (p.ImageHeight - i) / n and round up
//		size := int(math.Ceil(float64(p.ImageHeight-i) / float64(n)))
//		n--
//		go workerThread(thread, Grid{
//			width:  p.ImageWidth,
//			height: size,
//			cells:  currentState[i : i+size],
//		}, i, workers[thread])
//		i += size
//		thread++
//	}
//
//	return workers
//}
//
//func collateResults(p Params, currentState [][]byte, workers []workerChannels, events chan<- Event, turn int) {
//	for i := 0; i < p.Threads; i++ {
//		flippedCells := <-workers[i].results
//		for _, cell := range flippedCells {
//			events <- CellFlipped{
//				CompletedTurns: turn,
//				Cell:           cell,
//			}
//			currentState[cell.Y][cell.X] = 1 - currentState[cell.Y][cell.X]
//		}
//	}
//}
//
//func readImageToSlice(p Params, c distributorChannels) [][]byte {
//	c.ioCommand <- ioInput // send ioInput command to io goroutine
//	c.filename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
//
//	loadedCells := make([][]byte, p.ImageHeight)
//	for y := range loadedCells {
//		loadedCells[y] = make([]byte, p.ImageWidth)
//		for x := range loadedCells[y] {
//			if <-c.input > 0 {
//				loadedCells[y][x] = 1
//				c.events <- CellFlipped{
//					CompletedTurns: 0,
//					Cell:           util.Cell{X: x, Y: y},
//				}
//			}
//		}
//	}
//	return loadedCells
//}
//
//// send the current board data to the IO goroutine for output to an image
//func writeStateToImage(p Params, currentState [][]byte, c distributorChannels, turn int) {
//	c.ioCommand <- ioOutput
//	c.filename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageHeight; x++ {
//			c.output <- currentState[y][x]
//		}
//	}
//}
//
//// distributor divides the work between workers and interacts with other goroutines.
//func distributor(p Params, c distributorChannels) {
//	currentState := readImageToSlice(p, c)
//
//	workers := startWorkers(p, currentState)
//	ticker := time.NewTicker(2 * time.Second)
//
//	for turn := 0; turn < p.Turns; {
//		select {
//		case <-ticker.C:
//			c.events <- AliveCellsCount{
//				CompletedTurns: turn,
//				CellsCount:     countAliveCells(p, currentState),
//			}
//		case key := <-c.keypresses:
//			switch key {
//			case 'p':
//				fmt.Println("Current turn:", turn)
//				for {
//					if <-c.keypresses == 'p' {
//						break
//					}
//				}
//			case 'q':
//				writeStateToImage(p, currentState, c, turn)
//				os.Exit(0)
//			case 's':
//				writeStateToImage(p, currentState, c, turn)
//			}
//		default:
//			collateResults(p, currentState, workers, c.events, turn)
//			c.events <- TurnComplete{turn + 1}
//			//fmt.Println(turn, countAliveCells(p, currentState))
//			turn++
//		}
//	}
//
//	c.events <- FinalTurnComplete{
//		CompletedTurns: p.Turns,
//		Alive:          calculateAliveCells(p, currentState),
//	}
//	writeStateToImage(p, currentState, c, p.Turns)
//
//	// Make sure that the Io has finished any output before exiting.
//	c.ioCommand <- ioCheckIdle
//	<-c.ioIdle
//
//	c.events <- StateChange{p.Turns, Quitting}
//	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
//	close(c.events)
//}
