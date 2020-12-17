package gol

import (
	"fmt"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Controller struct {
	thisAddr    string
	distributor stubs.Remote
	filename    string
	io          ioState
	keyPresses  chan rune
	events      chan Event
}

func (c *Controller) startGame(grid stubs.Grid) {
	err := c.distributor.Call(stubs.SetInitialState, stubs.DistributorInitialState{
		Grid: grid,
		//ControllerAddr: thisAddr,
	}, nil)
	if err != nil {
		panic(err)
	}
}

func runInstruction(distributor stubs.Remote, instruction stubs.Instruction) (result stubs.InstructionResult) {
	distributor.Call(stubs.GetState, instruction, &result)
	return result
}

func getCurrentTurn(distributor stubs.Remote) int {
	return runInstruction(distributor, stubs.GetCurrentTurn).CurrentTurn
}

func setProcessingPaused(distributor stubs.Remote, paused bool) int {
	p := stubs.GetCurrentTurn
	if paused {
		p |= stubs.Pause
	} else {
		p |= stubs.Resume
	}
	return runInstruction(distributor, p).CurrentTurn
}

func (c *Controller) run() {
	c.distributor.Connect()

	if c.filename != "" {
		state := c.io.readImageToSlice(c.filename)
		c.startGame(state)
	}

	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ticker.C:
			aliveCells := runInstruction(c.distributor, stubs.GetAliveCellsCount|stubs.GetCurrentTurn)
			fmt.Printf("turn %d, %d alive cells\n", aliveCells.CurrentTurn, aliveCells.AliveCellsCount)
			c.events <- AliveCellsCount{
				CompletedTurns: aliveCells.CurrentTurn,
				CellsCount:     aliveCells.AliveCellsCount,
			}
		case key := <-c.keyPresses:
			switch key {
			case 'p':
				fmt.Printf("pausing, current turn: %d\n", setProcessingPaused(c.distributor, true))
				for {
					if <-c.keyPresses == 'p' {
						fmt.Printf("resuming, current turn: %d\n", setProcessingPaused(c.distributor, false))
						break
					}
				}
			//case 'q':
			//	writeStateToImage(p, currentState, c, turn)
			//	os.Exit(0)
			case 's':
				fmt.Printf("fetching state\n")
				state := runInstruction(c.distributor, stubs.GetCurrentTurn|stubs.GetWholeState)
				filename := fmt.Sprintf("%dx%dx%d", state.State.Width, state.State.Height, state.CurrentTurn)
				fmt.Printf("got state at turn %d, saving to %s\n", state.CurrentTurn, filename)
				c.io.writeStateToImage(state.State, filename)
			case 'k':
				fmt.Printf("Shutting down system\n")
				runInstruction(c.distributor, stubs.GetCurrentTurn|stubs.Shutdown)
			}
		}
		//var result [][]byte
		//distributor.Call(stubs.GetState, stubs.GetWholeState, &result)
		//fmt.Println("Got", result)
	}
}

func RunController(thisAddr string, distributorAddr string, filename string, keyPresses chan rune, events chan Event) {
	fmt.Println("starting controller")

	thisController := Controller{
		thisAddr:    thisAddr,
		distributor: stubs.Remote{Addr: distributorAddr},
		filename:    filename,
		io:          StartIo(),
		keyPresses:  keyPresses,
		events:      events,
	}

	thisController.run()
}

func RunControllerWithArgs(args []string) {
	//controllerCommand := flag.NewFlagSet("controller", flag.ExitOnError)
	//
	////thisAddr := controllerCommand.String("ip", "127.0.0.1:8020", "IP and port to listen on")
	//distributorAddr := controllerCommand.String("distributor", "127.0.0.1:8030", "Address of distributor instance")
	//width := controllerCommand.Int("width", -1, "Width of game board")
	//height := controllerCommand.Int("height", -1, "Height of game board")
	//start := controllerCommand.Bool("start", false, "Whether to start game (false to connect to existing game)")
	//util.Check(controllerCommand.Parse(args))
	//
	//keyPresses := make(chan rune, 10)
	//events := make(chan Event, 1000)
	//
	//go RunController(*distributorAddr, *width, *height, *start, keyPresses, events)
}

//
//func controller(p Params, c distributorChannels) {
//	currentState := readImageToSlice(p, c)
//
//	workers := startWorkers(p, currentState)
//	//ticker := time.NewTicker(2 * time.Second)
//
//	for turn := 0; turn < p.Turns; {
//		select {
//		//case <-ticker.C:
//		//	c.events <- AliveCellsCount{
//		//		CompletedTurns: turn,
//		//		CellsCount:     countAliveCells(p, currentState),
//		//	}
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
//
