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
		port := workerCommand.Int("port", 8000, "port to serve on")
		util.Check(workerCommand.Parse(os.Args[2:]))
		gol.RunWorker(*port)
	case "distributor":
		distributorCommand := flag.NewFlagSet("distributor", flag.ExitOnError)
		thisAddr := distributorCommand.String("ip", "", "IP of this distributor")
		workerAddrs := distributorCommand.String("workers", "127.0.0.1:8030", "comma separated worker addresses")
		util.Check(distributorCommand.Parse(os.Args[2:]))
		gol.RunDistributor(*thisAddr, strings.Split(*workerAddrs, ","))
	case "controller":
		controllerCommand := flag.NewFlagSet("controller", flag.ExitOnError)
		//thisAddr := controllerCommand.String("ip", "", "IP of this controller")
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
		go gol.RunController(*distributorAddr, newJob, keyPresses, events)
		sdl.Start(gol.Params{0, 0, 1, 1}, events, keyPresses)
	default:
		panic("instance type not recognised")
	}
}
