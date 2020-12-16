package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

//func readImageToSlice(p Params, c ioChannels) [][]byte {
//	c.command <- ioInput // send ioInput command to io goroutine
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

//func doIO(p Params) ioChannels {
//		ioCommand := make(chan ioCommand)
//		ioIdle := make(chan bool)
//		ioFilename := make(chan string)
//		ioInput := make(chan uint8)
//		ioOutput := make(chan uint8)
//
//		ioChannels := ioChannels{
//			ioCommand,
//			ioIdle,
//			ioFilename,
//			ioOutput,
//			ioInput,
//		}
//
//		go startIo(p, ioChannels)
//		return ioChannels
//}

func main() {
	thisAddr := flag.String("ip", "127.0.0.1:8020", "IP and port to listen on")
	distributorAddr := flag.String("distributor", "127.0.0.1:8030", "Address of distributor instance")
	flag.Parse()
	//go serve(Worker{}, *thisAddr)
	//workerServer()
	client, _ := rpc.Dial("tcp", *distributorAddr)

	state := make([][]byte, 16)
	for y := 0; y < 16; y++ {
		state[y] = make([]byte, 16)
		for x := 0; x < 16; x++ {
			state[y][x] = byte(rand.Intn(2))
		}
	}

	client.Call(stubs.SetInitialState, stubs.DistributorInitialState{
		Width:          16,
		Height:         16,
		Cells:          state,
		ControllerAddr: *thisAddr,
	}, nil)
	for {
		time.Sleep(5 * time.Second)
		fmt.Println("Getting state")
		var result [][]byte
		client.Call(stubs.GetState, false, &result)
		fmt.Println("Got", result)
	}
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
