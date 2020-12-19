package gol

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"uk.ac.bris.cs/gameoflife/util"
)

// ioState is the internal state of the io goroutine.
type ioState struct {
	command chan ioCommand
	idle    chan bool

	filename chan string
	output   chan Grid
	input    chan Grid
}

// IoCommand allows requesting behaviour from the io (pgm) goroutine.
type ioCommand uint8

// This is a way of creating enums in Go.
// It will evaluate to:
//		ioOutput 	= 0
//		ioInput 	= 1
//		ioCheckIdle = 2
const (
	ioOutput ioCommand = iota
	ioInput
	ioCheckIdle
)

// writePgmImage receives an array of bytes and writes it to a pgm file.
func (io *ioState) writePgmImage() {
	_ = os.Mkdir("out", os.ModePerm)

	filename := <-io.filename
	file, ioError := os.Create("out/" + filename + ".pgm")
	util.Check(ioError)
	defer file.Close()

	world := <-io.output

	_, _ = file.WriteString("P5\n")
	//_, _ = file.WriteString("# PGM file writer by pnmmodules (https://github.com/owainkenwayucl/pnmmodules).\n")
	_, _ = file.WriteString(strconv.Itoa(world.Width))
	_, _ = file.WriteString(" ")
	_, _ = file.WriteString(strconv.Itoa(world.Height))
	_, _ = file.WriteString("\n")
	_, _ = file.WriteString(strconv.Itoa(255))
	_, _ = file.WriteString("\n")


	for y := 0; y < world.Height; y++ {
		for x := 0; x < world.Width; x++ {
			_, ioError = file.Write([]byte{world.Cells[y][x] * 255})
			util.Check(ioError)
		}
	}

	ioError = file.Sync()
	util.Check(ioError)

	fmt.Println("File", filename, "output done!")
}

// readPgmImage opens a pgm file and sends its data as an array of bytes.
func (io *ioState) readPgmImage() {
	filename := <-io.filename
	data, ioError := ioutil.ReadFile("images/" + filename + ".pgm")
	util.Check(ioError)

	fields := strings.Fields(string(data))

	if fields[0] != "P5" {
		panic("Not a pgm file")
	}

	width, _ := strconv.Atoi(fields[1])
	height, _ := strconv.Atoi(fields[2])
	maxval, _ := strconv.Atoi(fields[3])
	if maxval != 255 {
		panic("Incorrect maxval/bit depth")
	}

	world := make([][]byte, width)
	raw := []byte(fields[4])

	for y := 0; y < height; y++ {
		world[y] = make([]byte, width)
		for x := 0; x < width; x++ {
			world[y][x] = raw[y * width + x] >> 7
		}
	}
	io.input <- Grid{
		Width:  width,
		Height: height,
		Cells:  world,
	}
}

func (io *ioState) ioLoop() {
	for {
		select {
		case command := <-io.command:
			switch command {
			case ioInput:
				io.readPgmImage()
			case ioOutput:
				io.writePgmImage()
			case ioCheckIdle:
				io.idle <- true
			}
		}
	}
}

// startIo should be the entrypoint of the io goroutine.
func StartIo() ioState {
	io := ioState{
		command:  make(chan ioCommand),
		idle:     make(chan bool),
		filename: make(chan string),
		output:   make(chan Grid),
		input:    make(chan Grid),
	}

	go io.ioLoop()

	return io
}

// synchronously write a grid to a PGM file
func (io *ioState) writeStateToImage(state Grid, filename string) {
	if os.Getenv("NO_FILE_WRITE") != "" {
		fmt.Println("skipping file write for benchmark")
		return
	}
	io.command <- ioOutput
	io.filename <- filename
	io.output <- state

	io.waitUntilFinished()
}

// synchronously read a grid from a PGM file
func (io *ioState) readImageToSlice(filename string) Grid {
	fmt.Println("reading", filename)
	io.command <- ioInput // send ioInput command to io goroutine
	io.filename <- filename

	return <-io.input
}

func (io *ioState) waitUntilFinished() {
	io.command <- ioCheckIdle
	<-io.idle
}