package main

import (
	"context"
	"fmt"
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
	totalConfirmed := 0

	confirmedChannel := make(chan int, 10)

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

	for totalConfirmed < totalToAchieve * 4 {
		select{
		case signal := <-confirmedChannel:
			if signal == 1 {
				totalConfirmed ++
				fmt.Println("----------------------------------------")
				fmt.Println("Total Confirmed: ", totalConfirmed)
				fmt.Println("----------------------------------------")
			}
		}
	}
	cancelFunction()
	wg.Wait()
}
