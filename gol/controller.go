package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Controller struct {
	thisAddr    string
	distributor stubs.Remote
	filename    string
	maxTurns    int
	io          ioState
	keyPresses  chan rune
	events      chan Event
	gameEndChan chan stubs.InstructionResult
}

func (c *Controller) GameFinished(req stubs.InstructionResult, res *bool) (err error) {
	c.gameEndChan <- req
	return
}

func (c *Controller) startGame(grid stubs.Grid) {
	err := c.distributor.Call(stubs.SetInitialState, stubs.DistributorInitialState{
		Grid:           grid,
		MaxTurns:       c.maxTurns,
		ControllerAddr: c.thisAddr,
	}, nil)
	if err != nil {
		panic(err)
	}
}

func (c *Controller) runInstruction(instruction stubs.Instruction) (result stubs.InstructionResult, err error) {
	//err := c.distributor.Call(stubs.GetState, instruction, &result)
	//if err != nil && err.Error() == errors.GameAlreadyFinished.Error() {
	//	return result
	//}
	//util.Check(err)
	return result, c.distributor.Call(stubs.GetState, instruction, &result)
}


//func (c *Controller) setProcessingPaused(paused bool) int {
//	p := stubs.GetCurrentTurn
//	if paused {
//		p |= stubs.Pause
//	} else {
//		p |= stubs.Resume
//	}
//	return c.runInstruction(p).CurrentTurn
//}

func (c *Controller) saveState(state stubs.InstructionResult) {
	filename := fmt.Sprintf("%dx%dx%d", state.State.Width, state.State.Height, state.CurrentTurn)
	fmt.Printf("got state at turn %d, saving to %s\n", state.CurrentTurn, filename)
	c.io.writeStateToImage(state.State, filename)
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
			aliveCells, err := c.runInstruction(stubs.GetAliveCellsCount | stubs.GetCurrentTurn)
			if err != nil {
				continue
			}
			fmt.Printf("turn %d, %d alive cells\n", aliveCells.CurrentTurn, aliveCells.AliveCellsCount)
			c.events <- AliveCellsCount{
				CompletedTurns: aliveCells.CurrentTurn,
				CellsCount:     aliveCells.AliveCellsCount,
			}
		case key := <-c.keyPresses:
			switch key {
			case 'p':
				state, err :=  c.runInstruction(stubs.GetCurrentTurn | stubs.Pause)
				if err != nil {
					fmt.Printf("unable to pause state: %v", err)
					continue
				}
				fmt.Printf("paused, current turn: %d\n", state.CurrentTurn)
				for {
					if <-c.keyPresses == 'p' {
						state, err :=  c.runInstruction(stubs.GetCurrentTurn | stubs.Resume)
						util.Check(err)
						fmt.Printf("resumed, current turn: %d\n", state.CurrentTurn)
						break
					}
				}
			case 'q':
				fmt.Printf("fetching state and quitting\n")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.GetWholeState)
				if err == nil {
					c.saveState(state)
				} else {
					fmt.Printf("unable to fetch state: %v", err)
				}
				c.saveState(state)
				os.Exit(0)
			case 's':
				fmt.Printf("fetching state\n")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.GetWholeState)
				if err == nil {
					c.saveState(state)
				} else {
					fmt.Printf("unable to fetch state: %v", err)
				}
			case 'k':
				fmt.Printf("shutting down system\n")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.Shutdown)
				if err == nil {
					fmt.Printf("system shut down at turn %d\n", state.CurrentTurn)
				} else {
					fmt.Printf("unable to shut down system state: %v", err)
				}
				os.Exit(0)
			}
		case gameEnd := <-c.gameEndChan:
			fmt.Printf("game ended\n")
			c.saveState(gameEnd)
			os.Exit(0)
		}
	}
}

func RunController(thisAddr string, distributorAddr string, filename string, maxTurns int, keyPresses chan rune, events chan Event) {
	fmt.Println("starting controller")

	thisController := Controller{
		thisAddr:    thisAddr,
		distributor: stubs.Remote{Addr: distributorAddr},
		filename:    filename,
		maxTurns:    maxTurns,
		io:          StartIo(),
		keyPresses:  keyPresses,
		events:      events,
		gameEndChan: make(chan stubs.InstructionResult, 1),
	}

	util.Check(rpc.Register(&thisController))
	listener, _ := net.Listen("tcp", thisAddr)
	defer listener.Close()
	go rpc.Accept(listener)

	thisController.run()
}
