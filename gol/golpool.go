package gol

import (
	"fmt"
	"math"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Strip struct {
	Top    int
	Height int
}

type GolProcessor struct {
	p            Params
	currentState *Grid
	nextState    *Grid
	pauseChan    chan bool
	wg           sync.WaitGroup
	currentTurn  int
}

func makeStrips(totalHeight, numStrips int) []Strip {
	result := make([]Strip, 0, numStrips)
	thread := 0
	for i := 0; i < totalHeight; thread++ {
		// calculate (p.ImageHeight - i) / n and round up
		size := int(math.Ceil(float64(totalHeight-i) / float64(numStrips-thread)))
		result = append(result, Strip{Top: i, Height: size})
		i += size
	}
	return result
}

func (g *GolProcessor) doTurn(strip Strip) *[]util.Cell {
	cellFlips := make([]util.Cell, 0)
	for y := strip.Top; y < strip.Top+strip.Height; y++ {
		for x := 0; x < g.p.ImageWidth; x++ {
			changed := g.nextState.set(x, y, g.currentState.nextState(x, y))
			if changed {
				cellFlips = append(cellFlips, util.Cell{
					X: x,
					Y: y,
				})
			}
		}
	}
	return &cellFlips
}

func (g *GolProcessor) worker(strip Strip, events chan<- Event, ready chan bool, done chan bool) {
	//t := time.Now().UnixNano()
	for i := 0; i <= g.p.Turns; i++ {
		//n := time.Now().UnixNano()
		//fmt.Printf("dt %d %d %v\n", strip.Top, g.currentTurn, (n-t)/1000000)
		//t = n
		for _, flip := range *(g.doTurn(strip)) {
			events <- CellFlipped{
				CompletedTurns: g.currentTurn + 1,
				Cell:           flip,
			}
		}
		done <- true
		<-ready
		//fmt.Println("r3")
	}
}

func (g *GolProcessor) pause() {
	g.pauseChan <- true
}

func (g *GolProcessor) start(events chan<- Event, done chan<- bool) {
	g.nextState = &Grid{
		Width:  g.currentState.Width,
		Height: g.currentState.Height,
	}
	g.nextState.make()
	g.pauseChan = make(chan bool, 1)
	readyChan := make(chan bool, g.p.Threads)
	doneChan := make(chan bool, g.p.Threads)
	strips := makeStrips(g.p.ImageHeight, g.p.Threads)

	for i := 0; i < g.p.Threads; i++ {
		go g.worker(strips[i], events, readyChan, doneChan)
	}

	for g.currentTurn = 0; g.currentTurn <= g.p.Turns; {
		for i := 0; i < g.p.Threads; i++ {
			<-doneChan
		}
		// safe time
		g.currentTurn++

		fmt.Println(g.currentState)
		select {
		case <-g.pauseChan:
			fmt.Println("paused")
			<-g.pauseChan
		default:
		}
		events <- TurnComplete{
			CompletedTurns: g.currentTurn,
		}
		if g.currentTurn == g.p.Turns {
			break
		}
		inter := g.currentState
		g.currentState = g.nextState
		g.nextState = inter
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < g.p.Threads; i++ {
			readyChan <- true
		}
	}
	done <- true
}
