package main

import (
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

// Benchmark applies the filter to the ship.png b.N times.
// The time taken is carefully measured by go.
// The b.N  repetition is needed because benchmark results are not always constant.
func Benchmark(b *testing.B) {
	os.Stdout = nil
	events := make(chan int, 0)
	tests := []struct {
		name string
		fun func(...<-chan int) <-chan int
	} {
		{"TestGol", TestGol},
		{"TestAlive", TestAlive},
		{"TestPgm", TestPgm},
	}
	for _, test := range tests {
		// b.Run(test.TestGol, func(b *testing.B)) {
		// 	golChan := test.TestGol()
		// }
		// b.Run(test.TestAlive, func(b *testing.B)) {
		// 	aliveChan := test.TestAlive()
		// }
		// b.Run(test.TestPgm, func(b *testing.B)) {
		// 	pgmChan := test.TestPgm()
		// }
		b.Run(test.TestGol)
		gol.Run(p Params, tests, keyPresses)
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

