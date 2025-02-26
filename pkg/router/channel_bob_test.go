package router

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestCancellation demonstrates how channel-based cancellation works
func TestCancellation(t *testing.T) {
	// Create a cancellation channel
	cancelCh := make(chan struct{})
	var wg sync.WaitGroup

	// Start several workers that will check for cancellation
	startTime := time.Now()
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			fmt.Printf("[%v] Worker %d: Started\n", time.Since(startTime).Truncate(time.Millisecond), id)

			// Simulate some work
			for j := 1; j <= 5; j++ {
				// Check for cancellation at each iteration
				select {
				case <-cancelCh:
					fmt.Printf("[%v] Worker %d: Cancelled during iteration %d\n",
						time.Since(startTime).Truncate(time.Millisecond), id, j)
					return
				default:
					// Continue working
				}

				// Do some work
				fmt.Printf("[%v] Worker %d: Working on iteration %d\n",
					time.Since(startTime).Truncate(time.Millisecond), id, j)
				time.Sleep(200 * time.Millisecond)
			}

			fmt.Printf("[%v] Worker %d: Completed all work\n",
				time.Since(startTime).Truncate(time.Millisecond), id)
		}(i)
	}

	// Wait a bit and then trigger cancellation
	go func() {
		time.Sleep(550 * time.Millisecond)
		fmt.Printf("[%v] Main: Triggering cancellation\n",
			time.Since(startTime).Truncate(time.Millisecond))
		close(cancelCh)
	}()

	// Wait for all workers to finish
	wg.Wait()
	fmt.Printf("[%v] Main: All workers have exited\n",
		time.Since(startTime).Truncate(time.Millisecond))
}

// TestBlockingSelect shows how select can be used to wait for events
func TestBlockingSelect(t *testing.T) {
	cancelCh := make(chan struct{})
	workDoneCh := make(chan struct{})
	startTime := time.Now()

	fmt.Println("\n=== Blocking Select Test ===")

	// Test 1: Work completes before cancellation
	go func() {
		fmt.Printf("[%v] Starting work that will complete\n",
			time.Since(startTime).Truncate(time.Millisecond))
		time.Sleep(300 * time.Millisecond)
		close(workDoneCh)
		fmt.Printf("[%v] Work completed\n",
			time.Since(startTime).Truncate(time.Millisecond))
	}()

	// Wait for either completion or cancellation
	select {
	case <-workDoneCh:
		fmt.Printf("[%v] Main: Work completed successfully\n",
			time.Since(startTime).Truncate(time.Millisecond))
	case <-cancelCh:
		fmt.Printf("[%v] Main: Work was cancelled\n",
			time.Since(startTime).Truncate(time.Millisecond))
	}

	// Reset channels
	cancelCh = make(chan struct{})
	workDoneCh = make(chan struct{})
	startTime = time.Now()

	fmt.Println("\n=== Test with cancellation happening first ===")

	// Test 2: Cancellation happens before work completes
	go func() {
		fmt.Printf("[%v] Starting work that will be cancelled\n",
			time.Since(startTime).Truncate(time.Millisecond))
		time.Sleep(500 * time.Millisecond)
		close(workDoneCh)
		fmt.Printf("[%v] Work completed (but we already cancelled)\n",
			time.Since(startTime).Truncate(time.Millisecond))
	}()

	go func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Printf("[%v] Triggering cancellation\n",
			time.Since(startTime).Truncate(time.Millisecond))
		close(cancelCh)
	}()

	// Wait for either completion or cancellation
	select {
	case <-workDoneCh:
		fmt.Printf("[%v] Main: Work completed successfully\n",
			time.Since(startTime).Truncate(time.Millisecond))
	case <-cancelCh:
		fmt.Printf("[%v] Main: Work was cancelled\n",
			time.Since(startTime).Truncate(time.Millisecond))
	}

	// Need to wait a bit to see the "work completed" message after cancellation
	time.Sleep(600 * time.Millisecond)
}

// Run the tests (no need to use the testing framework for this demo)
func TestMain(m *testing.M) {
	fmt.Println("=== Running Channel Cancellation Tests ===")

	fmt.Println("\n=== Test 1: Multiple workers with cancellation ===")
	TestCancellation(&testing.T{})

	fmt.Println("\n=== Test 2: Blocking select patterns ===")
	TestBlockingSelect(&testing.T{})
}
