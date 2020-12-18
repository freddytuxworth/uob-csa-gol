package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()

	fmt.Println(os.Args)
	switch os.Args[1] {
	case "worker":
		workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)
		thisAddr := workerCommand.String("ip", "", "IP and port to listen on")
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
		//filename := controllerCommand.String("filename", "", "filename to start game (empty to join existing game)")
		//maxTurns := controllerCommand.Int("turns", 99999, "number of turns to run (ignored if not starting new game")
		job := controllerCommand.String("job", "", "job details, format 'job_name,filename,turns'. empty to join existing game.")
		util.Check(controllerCommand.Parse(os.Args[2:]))

		jobParts := strings.Split(*job, ",")
		var newJob stubs.GolJob
		if len(jobParts) == 3 {
			turns, err := strconv.Atoi(jobParts[2])
			util.Check(err)
			newJob = stubs.GolJob{
				Name:     jobParts[0],
				Filename: jobParts[1],
				Turns:    turns,
			}
		}
		keyPresses := make(chan rune, 10)
		events := make(chan gol.Event, 1000)
		go gol.RunController(*thisAddr, *distributorAddr, newJob, keyPresses, events)
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
