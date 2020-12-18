package gol

import (
	"fmt"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func (c *Controller) logf(format string, obj ...interface{}) {
	fmt.Printf("%s	%s\n",
		bold("[%s] controller (%s):", c.job.Name, c.thisAddr),
		fmt.Sprintf(format, obj...))
}

type Controller struct {
	thisAddr    string
	distributor stubs.Remote
	job         stubs.GolJob
	io          ioState
	keyPresses  chan rune
	events      chan Event
	gameEndChan chan *rpc.Call
}

//func (c *Controller) GameFinished(req stubs.InstructionResult, res *bool) (err error) {
//	c.gameEndChan <- req
//	return
//}

func (c *Controller) startGame(grid stubs.Grid) {
	util.Check(c.distributor.Call(stubs.SetInitialState, stubs.DistributorInitialState{
		JobName: c.job.Name,
		Grid:    grid,
		Turns:   c.job.Turns,
		//ControllerAddr: c.thisAddr,
	}, nil))
	c.logf("set distributor initial state")
}

func (c *Controller) runInstruction(instruction stubs.Instruction) (result stubs.InstructionResult, err error) {
	return result, c.distributor.Call(stubs.GetState, instruction, &result)
}

func (c *Controller) saveState(state stubs.InstructionResult) {
	fmt.Println(state)
	filename := fmt.Sprintf("%dx%dx%d", state.State.Width, state.State.Height, state.CurrentTurn)
	c.logf("saving state at turn %d, to %s\n", state.CurrentTurn, filename)
	c.io.writeStateToImage(state.State, filename)
}

func (c *Controller) stop() {
	c.logf("stopping gracefully")
	c.io.waitUntilFinished()
	close(c.events)
}

func (c *Controller) run() {
	c.logf("starting controller")

	if c.job.Filename != "" {
		state := c.io.readImageToSlice(c.job.Filename)
		c.logf("read state from %s", c.job.Filename)
		if c.job.Turns < 1 {
			c.saveState(stubs.InstructionResult{
				CurrentTurn:     0,
				State:           state,
			})
			c.events <- FinalTurnComplete{
				CompletedTurns: 0,
				Alive:          getAliveCells(state.Width, state.Height, state.Cells),
			}
			return
		}

		c.distributor.Connect()
		c.logf("connected to distributor")

		c.startGame(state)
	}

	var endState stubs.InstructionResult
	c.distributor.Go("Distributor.GetEndState", false, &endState, c.gameEndChan)

	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ticker.C:
			aliveCells, err := c.runInstruction(stubs.GetAliveCellsCount | stubs.GetCurrentTurn)
			if err != nil {
				continue
			}
			c.events <- AliveCellsCount{
				CompletedTurns: aliveCells.CurrentTurn,
				CellsCount:     aliveCells.AliveCellsCount,
			}
		case key := <-c.keyPresses:
			switch key {
			case 'p':
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.Pause)
				if err != nil {
					c.logf("unable to pause state: %v", err)
					continue
				}
				c.logf("paused, current turn: %d", state.CurrentTurn)
				for <-c.keyPresses != 'p' {
				}
				state, err = c.runInstruction(stubs.GetCurrentTurn | stubs.Resume)
				util.Check(err)
				c.logf("resumed, current turn: %d", state.CurrentTurn)
			case 'q':
				c.logf("fetching state, saving to image and quitting")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.GetWholeState)
				if err == nil {
					c.saveState(state)
				} else {
					c.logf("unable to fetch state: %v", err)
				}
				c.saveState(state)
				return
			case 's':
				c.logf("fetching state and saving to image")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.GetWholeState)
				if err == nil {
					c.saveState(state)
				} else {
					c.logf("unable to fetch state: %v", err)
				}
			case 'k':
				c.logf("shutting down system")
				state, err := c.runInstruction(stubs.GetCurrentTurn | stubs.Shutdown)
				if err == nil {
					c.logf("system shut down at turn %d", state.CurrentTurn)
				} else {
					c.logf("unable to shut down system state: %v", err)
				}
				return
			}
		case <-c.gameEndChan:
			c.logf("game finished")
			c.events <- FinalTurnComplete{
				CompletedTurns: endState.CurrentTurn,
				Alive:          getAliveCells(endState.State.Width, endState.State.Height, endState.State.Cells),
			}
			c.saveState(endState)
			return
		}
	}
}

func RunController(distributorAddr string, job stubs.GolJob, keyPresses chan rune, events chan Event) {
	thisController := Controller{
		//thisAddr:    thisAddr,
		distributor: stubs.Remote{Addr: distributorAddr},
		job:         job,
		io:          StartIo(),
		keyPresses:  keyPresses,
		events:      events,
		gameEndChan: make(chan *rpc.Call, 1),
	}

	//server := rpc.NewServer()
	//server.HandleHTTP()
	//util.Check(rpc.Register(&thisController))
	//stubs.ServeHTTP(thisAddr)
	//rpc.HandleHTTP()
	//l, e := net.Listen("tcp", thisAddr)
	//if e != nil {
	//	log.Fatal("listen error:", e)
	//}
	//go http.Serve(l, nil)

	thisController.run()
	thisController.stop()
	//l.Close()
	//defer listener.Close()
}
