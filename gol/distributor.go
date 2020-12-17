package gol

import (
	"fmt"
	"math"
	"net"
	"net/rpc"
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

//var stateUpdateChan = make(chan stubs.InstructionResult, 1)
//var stateRequestChan = make(chan stubs.Instruction, 1)
//var initialStateChan = make(chan stubs.DistributorInitialState, 1)
//var workerStateUpdates = make(chan stubs.InstructionResult, 1)

type Distributor struct {
	stateUpdateChan    chan stubs.InstructionResult
	stateRequestChan   chan stubs.Instruction
	initialStateChan   chan stubs.DistributorInitialState
	workerStateUpdates chan stubs.InstructionResult
}

func (d *Distributor) GetState(req stubs.Instruction, res *stubs.InstructionResult) (err error) {
	d.stateRequestChan <- req
	*res = <-d.stateUpdateChan
	return
}

func (d *Distributor) SetInitialState(req stubs.DistributorInitialState, res *bool) (err error) {
	d.initialStateChan <- req
	return
}

func (d *Distributor) WorkerState(req stubs.InstructionResult, res *bool) (err error) {
	d.workerStateUpdates <- req
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

func startWorkers(p Params, currentState [][]byte, thisAddr string, workers []workerConnection) []strip {
	numWorkers := len(workers)
	strips := makeStrips(p.ImageHeight, numWorkers)
	for i, worker := range workers {
		worker.client.Go(stubs.SetState, stubs.WorkerInitialState{
			WorkerId: i,
			Grid: stubs.Grid{
				Width:  p.ImageWidth,
				Height: strips[i].height,
				Cells:  currentState[strips[i].top : strips[i].top+strips[i].height],
			},
			WorkerBelowAddr: workers[(i+1)%numWorkers].addr,
			DistributorAddr: thisAddr,
		}, nil, nil)
		fmt.Println("sent state to worker", i)
	}
	return strips
}

func (d *Distributor) fetchState(workerStrips []strip, currentState [][]byte, workers []workerConnection, request stubs.Instruction) stubs.InstructionResult {
	fmt.Println("Fetching state", request)
	workers[0].client.Call(stubs.GetWorkerState, request, nil)

	result := stubs.InstructionResult{
		CurrentTurn:     0,
		AliveCellsCount: 0,
	}
	for i := 0; i < len(workers); i++ {
		workerState := <-d.workerStateUpdates
		//fmt.Printf("worker %d state: %#v\n", i, workerState)
		if request.HasFlag(stubs.GetWholeState) {
			workerTop := workerStrips[workerState.WorkerId].top
			workerBottom := workerTop + workerStrips[workerState.WorkerId].height
			workerSection := currentState[workerTop:workerBottom]
			copy(workerSection, workerState.State.Cells)
		}
		result.CurrentTurn = workerState.CurrentTurn
		result.AliveCellsCount += workerState.AliveCellsCount
	}

	if request.HasFlag(stubs.GetWholeState) {
		result.State = stubs.Grid{
			Width:  len(currentState),
			Height: len(currentState[0]),
			Cells:  currentState,
		}
		//fmt.Printf("got complete state:\n%v", result.State)
	}
	//fmt.Printf("Compiled state update: %#v\n", result)
	return result
}

func connectToWorkers(workerAddrs []string) (workers []workerConnection) {
	for _, workerAddr := range workerAddrs {
		client, err := rpc.Dial("tcp", workerAddr)
		if err != nil {
			panic(fmt.Sprintf("could not connect to workerConnection at %s", workerAddr))
		}
		workers = append(workers, workerConnection{
			client: client,
			addr:   workerAddr,
		})
	}
	return workers
}

func (d *Distributor) run(thisAddr string, workerAddrs []string) {
	fmt.Println("starting distributor")
	p := Params{
		Threads: len(workerAddrs),
	}

	workers := connectToWorkers(workerAddrs)

	var workerStrips []strip
	var currentState [][]byte
	for {
		select {
		case initialState := <-d.initialStateChan:
			fmt.Println("Setting Initial State")
			fmt.Println(initialState)
			currentState = initialState.Grid.Cells
			p.ImageWidth = initialState.Grid.Width
			p.ImageHeight = initialState.Grid.Height
			workerStrips = startWorkers(p, initialState.Grid.Cells, thisAddr, workers)
		case stateRequest := <-d.stateRequestChan:
			d.stateUpdateChan <- d.fetchState(workerStrips, currentState, workers, stateRequest)
		}
	}
}

func RunDistributor(thisAddr string, workerAddrs []string) {
	thisDistributor := Distributor{
		stateUpdateChan:    make(chan stubs.InstructionResult, 1),
		stateRequestChan:   make(chan stubs.Instruction, 1),
		initialStateChan:   make(chan stubs.DistributorInitialState, 1),
		workerStateUpdates: make(chan stubs.InstructionResult, 1),
	}
	util.Check(rpc.Register(&thisDistributor))
	listener, _ := net.Listen("tcp", thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	//go stubs.Serve(Distributor{}, *thisAddr)
	thisDistributor.run(thisAddr, workerAddrs)
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
