package main

import (
	"context"
	"sync"

	//"sync/atomic"
	"testing"
	// use it as "hotstuff", e.g., hotstuff.Node{}
)


func TestFull(t *testing.T) {
	// create 4 nodes, where 4 = 4f+1 for f=1
	numNodes := NUMBER_OF_NODES


	ctx := context.Background()
	//Derive a context with cancel
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)

	var wg sync.WaitGroup

	totalToAchieve := 10

	confirmed := make([]int, numNodes)
	confirmedChannel := make(chan int, 1000)

	// select {case totalConfirmed}
	// if num confirmed > 10; exit
	// pass context for graceful exit

	for idx := 0; idx < numNodes; idx++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entryPoint(ctxWithCancel, idx, confirmedChannel)
		}(idx)
	}

	for {
		select{
		case signal := <-confirmedChannel:
			confirmed[signal] += 1
		}
		allSet := true
		for i := 0; i < numNodes; i++ {
			if confirmed[i] < totalToAchieve {
				allSet = false
			}
		}
		if allSet {
			break
		}
	}
	cancelFunction()
	wg.Wait()
}
