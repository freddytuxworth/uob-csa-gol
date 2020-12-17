package main

import (
	"flag"
	"os"
	"runtime"
	"strings"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
	"uk.ac.bris.cs/gameoflife/util"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()

	switch os.Args[1] {
	case "worker":
		workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)
		thisAddr := workerCommand.String("ip", "127.0.0.1:8020", "IP and port to listen on")
		util.Check(workerCommand.Parse(os.Args[2:]))
		gol.RunWorker(*thisAddr)
	case "distributor":
		distributorCommand := flag.NewFlagSet("distributor", flag.ExitOnError)
		thisAddr := distributorCommand.String("ip", "127.0.0.1:8020", "IP and port to listen on")
		workerAddrs := distributorCommand.String("workers", "127.0.0.1:8030", "comma separated worker addresses")
		util.Check(distributorCommand.Parse(os.Args[2:]))
		gol.RunDistributor(*thisAddr, strings.Split(*workerAddrs, ","))
	case "controller":
		controllerCommand := flag.NewFlagSet("controller", flag.ExitOnError)
		thisAddr := controllerCommand.String("ip", "127.0.0.1:8020", "IP and port to listen on")
		distributorAddr := controllerCommand.String("distributor", "127.0.0.1:8030", "address of distributor instance")
		filename := controllerCommand.String("filename", "", "filename to start game (empty to join existing game)")
		//width := controllerCommand.Int("width", -1, "Width of game board")
		//height := controllerCommand.Int("height", -1, "Height of game board")
		//start := controllerCommand.Bool("start", false, "Whether to start game (false to connect to existing game)")
		util.Check(controllerCommand.Parse(os.Args[2:]))
		keyPresses := make(chan rune, 10)
		events := make(chan gol.Event, 1000)
		go gol.RunController(*thisAddr, *distributorAddr, *filename, keyPresses, events)
		sdl.Start(gol.Params{0, 0, 1, 1}, events, keyPresses)
	default:
		panic("instance type not recognised")
	}
	//var params gol.Params
	//
	//flag.IntVar(
	//	&params.Threads,
	//	"t",
	//	8,
	//	"Specify the number of worker threads to use. Defaults to 8.")
	//
	//flag.IntVar(
	//	&params.ImageWidth,
	//	"w",
	//	512,
	//	"Specify the width of the image. Defaults to 512.")
	//
	//flag.IntVar(
	//	&params.ImageHeight,
	//	"h",
	//	512,
	//	"Specify the height of the image. Defaults to 512.")
	//
	//flag.IntVar(
	//	&params.Turns,
	//	"turns",
	//	10000000000,
	//	"Specify the number of turns to process. Defaults to 10000000000.")
	//
	//flag.Parse()
	//
	//fmt.Println("Threads:", params.Threads)
	//fmt.Println("Width:", params.ImageWidth)
	//fmt.Println("Height:", params.ImageHeight)
	//
	//keyPresses := make(chan rune, 10)
	//events := make(chan gol.Event, 1000)
	//
	//gol.Run(params, events, keyPresses)
	//sdl.Start(params, events, keyPresses)
}
