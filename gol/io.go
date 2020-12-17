package gol

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"uk.ac.bris.cs/gameoflife/util"
)

type IoChannels struct {
	Command chan IoCommand
	Idle    chan bool

	Filename chan string
	Output   chan uint8
	Input    chan uint8
}

// ioState is the internal ioState of the io goroutine.
type ioState struct {
	params   Params
	channels IoChannels
}

// IoCommand allows requesting behaviour from the io (pgm) goroutine.
type IoCommand uint8

// This is a way of creating enums in Go.
// It will evaluate to:
//		ioOutput 	= 0
//		ioInput 	= 1
//		ioCheckIdle = 2
const (
	ioOutput IoCommand = iota
	ioInput
	ioCheckIdle
)

// writePgmImage receives an array of bytes and writes it to a pgm file.
func (io *ioState) writePgmImage() {
	_ = os.Mkdir("out", os.ModePerm)

	filename := <-io.channels.Filename
	file, ioError := os.Create("out/" + filename + ".pgm")
	util.Check(ioError)
	defer file.Close()

	_, _ = file.WriteString("P5\n")
	//_, _ = file.WriteString("# PGM file writer by pnmmodules (https://github.com/owainkenwayucl/pnmmodules).\n")
	_, _ = file.WriteString(strconv.Itoa(io.params.ImageWidth))
	_, _ = file.WriteString(" ")
	_, _ = file.WriteString(strconv.Itoa(io.params.ImageHeight))
	_, _ = file.WriteString("\n")
	_, _ = file.WriteString(strconv.Itoa(255))
	_, _ = file.WriteString("\n")

	world := make([][]byte, io.params.ImageHeight)
	for i := range world {
		world[i] = make([]byte, io.params.ImageWidth)
	}

	for y := 0; y < io.params.ImageHeight; y++ {
		for x := 0; x < io.params.ImageWidth; x++ {
			val := <-io.channels.Output
			//if val != 0 {
			//	fmt.Println(x, y)
			//}
			world[y][x] = val
		}
	}

	for y := 0; y < io.params.ImageHeight; y++ {
		for x := 0; x < io.params.ImageWidth; x++ {
			_, ioError = file.Write([]byte{world[y][x]})
			util.Check(ioError)
		}
	}

	ioError = file.Sync()
	util.Check(ioError)

	fmt.Println("File", filename, "output done!")
}

// readPgmImage opens a pgm file and sends its data as an array of bytes.
func (io *ioState) readPgmImage() {
	fmt.Println("Reading image!")
	filename := <-io.channels.Filename
	data, ioError := ioutil.ReadFile("images/" + filename + ".pgm")
	util.Check(ioError)

	fields := strings.Fields(string(data))

	if fields[0] != "P5" {
		panic("Not a pgm file")
	}

	width, _ := strconv.Atoi(fields[1])
	if width != io.params.ImageWidth {
		panic("Incorrect width")
	}

	height, _ := strconv.Atoi(fields[2])
	if height != io.params.ImageHeight {
		panic("Incorrect height")
	}

	maxval, _ := strconv.Atoi(fields[3])
	if maxval != 255 {
		panic("Incorrect maxval/bit depth")
	}

	image := []byte(fields[4])

	for _, b := range image {
		io.channels.Input <- b
	}

	fmt.Println("File", filename, "input done!")
}

// startIo should be the entrypoint of the io goroutine.
func StartIo(p Params, c IoChannels) {
	io := ioState{
		params:   p,
		channels: c,
	}

	for {
		select {
		case command := <-io.channels.Command:
			fmt.Println("Received IO Command", command)
			switch command {
			case ioInput:
				io.readPgmImage()
			case ioOutput:
				io.writePgmImage()
			case ioCheckIdle:
				io.channels.Idle <- true
			}
		}
	}
}

func ReadImageToSlice(p Params, c IoChannels) [][]byte {
	c.Command <- ioInput // send ioInput command to io goroutine
	c.Filename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	loadedCells := make([][]byte, p.ImageHeight)
	for y := range loadedCells {
		loadedCells[y] = make([]byte, p.ImageWidth)
		for x := range loadedCells[y] {
			if <-c.Input > 0 {
				loadedCells[y][x] = 1
			}
		}
	}
	return loadedCells
}
