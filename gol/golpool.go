package gol

import (
	"math"
	"sync"
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
	currentTurn  int
	wg           sync.WaitGroup
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
			changed := g.nextState.set(x, y, shouldSurvive(x, y, *g.currentState))
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

func (g *GolProcessor) worker(strip Strip, cellFlips chan<- *[]util.Cell) {
	for i := 0; i < g.p.Turns; i++ {
		flips := g.doTurn(strip)
		g.wg.Done()
		cellFlips <- flips
	}
}

func (g *GolProcessor) start(events chan<- Event, stop <-chan bool) {
	g.nextState = &Grid{
		Width:  g.currentState.Width,
		Height: g.currentState.Height,
	}
	g.nextState.make()

	workers := make([]<-chan *[]util.Cell, 8)
	strips := makeStrips(g.p.ImageHeight, g.p.Threads)
	g.wg.Add(g.p.Threads)
	for i := 0; i < g.p.Threads; i++ {
		workerChan := make(chan *[]util.Cell)
		go g.worker(strips[i], workerChan)
		workers = append(workers, workerChan)
	}
	for ; g.currentTurn < g.p.Turns; g.currentTurn++ {
		g.wg.Wait()
		select {
		case <-stop:
			<-stop
		default:
		}
		// flip states
		inter := g.currentState
		g.currentState = g.nextState
		g.nextState = inter
		for i := 0; i < g.p.Threads; i++ {
			for _, flip := range *(<-workers[i]) {
				events <- CellFlipped{
					CompletedTurns: g.currentTurn,
					Cell:           flip,
				}
			}
		}
		g.wg.Add(g.p.Threads)
	}
}
