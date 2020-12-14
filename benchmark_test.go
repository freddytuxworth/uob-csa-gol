package main

import (
	"fmt"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func runGolOnce(p gol.Params) {
	events := make(chan gol.Event, 1000)
	gol.Run(gol.Params{
		Turns:       p.Turns,
		Threads:     p.Threads,
		ImageWidth:  p.ImageWidth,
		ImageHeight: p.ImageHeight,
	}, events, nil)
	//gol.Run(gol.Params{
	//	Turns:       100,
	//	Threads:     workers,
	//	ImageWidth:  size,
	//	ImageHeight: size,
	//}, events, nil)
	for {
		e := <-events
		switch e.(type) {
		case gol.FinalTurnComplete:
			return
		}
	}
}

// Benchmark runs the game of life 100 times on each of 5 different image sizes with 1-16 workers.
// The time taken is carefully measured by go.
// The b.N  repetition is needed because benchmark results are not always constant.
func BenchmarkGol(b *testing.B) {
	//os.Stdout = nil
	sizes := []int{16, 64, 128, 256, 512}
	turns := []int{1, 100, 1000}
	//distributorChannels := gol.SetupIO(events, nil)
	for _, size := range sizes {
		for workers := 1; workers <= 16; workers *= 2 {
			for _, nTurns := range turns {
				b.Run(fmt.Sprintf("%dx%d, %d workers, %d turns", size, size, workers, nTurns), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						runGolOnce(gol.Params{
							Turns:       nTurns,
							Threads:     workers,
							ImageWidth:  size,
							ImageHeight: size,
						})
					}
				})
			}
		}
	}

	// os.Stdout = nil // Disable all program output apart from benchmark results
	// events := make(chan int, 0)
	// b.Run("Game of life benchmark", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		gol.Run(p Params, AliveCellsCount, keyPresses)
	// 	}
	// })
}

// TestGol, TestAlive, TestPgmpackage main
