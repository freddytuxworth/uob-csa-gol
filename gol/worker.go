package gol

import (
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct{}

var stateChan chan WorkerInitialState
var rowAboveChan chan RowAboveUpdate
var rowBelowChan chan []byte
var stateRequestChan chan bool

func (w *Worker) SetState(req WorkerInitialState, res bool) (err error) {
	stateChan <- req
	return
}

func (w *Worker) SetRowAbove(req RowAboveUpdate, res bool) (err error) {
	rowAboveChan <- req
	return
}

func (w *Worker) SetRowBelow(req []byte, res bool) (err error) {
	rowBelowChan <- req
	return
}

func (W *Worker) GetState(req bool, res bool) (err error) {
	stateRequestChan <- true
	return
}

func sendEdges(workerId int, wrappedGrid Grid, workerAbove, workerBelow *rpc.Client, makeStateRequest bool,) {

}

func workerServer(thisAddr string) {
	//distributor, _ := rpc.Dial("tcp", distributorAddr)

	var workerId int
	var wrappedGrid Grid

	var distributor *rpc.Client
	var workerAbove *rpc.Client
	var workerBelow *rpc.Client

	gotRowAbove := false
	gotRowBelow := false
	gotStateRequest := false
	stateRequestWorker := -1

	currentTurn := 0

	for {
		select {
		case initialState := <-stateChan:
			workerId = initialState.workerId
			wrappedGrid := Grid{
				width:  initialState.width,
				height: initialState.height + 2,
				cells:  make([][]byte, initialState.height+2),
			}
			currentTurn = 0

			for y := 0; y < wrappedGrid.height; y++ {
				wrappedGrid.cells[y] = make([]byte, initialState.width)
				if y > 0 && y < wrappedGrid.height-1 {
					copy(wrappedGrid.cells[y], initialState.cells[y-1])
				}
			}
			workerAbove, _ = rpc.Dial("tcp", initialState.workerAboveAddr)
			workerBelow, _ = rpc.Dial("tcp", initialState.workerBelowAddr)
			distributor, _ = rpc.Dial("tcp", initialState.distributorAddr)
		case rowAbove := <-rowAboveChan:
			wrappedGrid.cells[0] = rowAbove.rowAbove
			stateRequestWorker = rowAbove.stateRequestWorker
			gotRowAbove = true
		case rowBelow := <-rowBelowChan:
			wrappedGrid.cells[wrappedGrid.height-1] = rowBelow
			gotRowBelow = true
		case <-stateRequestChan:
			gotStateRequest = true
		}
		if gotRowAbove && gotRowBelow {
			//cellFlips := make([]util.Cell, 0)
			for y := 1; y < wrappedGrid.height-1; y++ {
				for x := 0; x < wrappedGrid.width; x++ {
					wrappedGrid.cells[y][x] = shouldSurvive(x, y, wrappedGrid)
					//if nextCellState != wrappedGrid.cells[y][x] {
					//	cellFlips = append(cellFlips, util.Cell{
					//		X: x,
					//		Y: y - 1,
					//	})
					//}
				}
			}
			//for _, cellFlip := range cellFlips {
			//	wrappedGrid.cells[cellFlip.Y+1][cellFlip.X] = 1 - wrappedGrid.cells[cellFlip.Y+1][cellFlip.X]
			//}

			nextWorkerStateRequestWorker := -1
			if gotStateRequest {
				nextWorkerStateRequestWorker = workerId
			} else if stateRequestWorker > -1 {
				nextWorkerStateRequestWorker = stateRequestWorker
			}
			workerAbove.Go(SetRowBelow, wrappedGrid.cells[1], nil, nil)
			workerBelow.Go(SetRowAbove, wrappedGrid.cells[wrappedGrid.height - 2], nil, nil)
			//distributor.Go(ChangeCells, cellFlips, nil, nil)
				//c.results <- cellFlips
			gotRowAbove = false
			gotRowBelow = false
			gotStateRequest = false
			currentTurn++
		}
	}

}
