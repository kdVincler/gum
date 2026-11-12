package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kdVincler/gum/internal/domain"
	"github.com/kdVincler/gum/internal/storage"
)

// Interface for the http checker so that the scheduler can be decoupled from the checker implementation.
type Checker interface {
	Check(ctx context.Context, monitor domain.Monitor) domain.CheckResult
}

/*
Orchestrator for concurrent monitoring

Specifies the following:
  - monitors (URLs) to check,
  - the specifick checker implementation to use,
  - the storage to save the check results to,
  - the number of workers, a.k.a. how many goroutines to run concurrently.
*/
type Scheduler struct {
	monitors []domain.Monitor
	checker  Checker
	store    *storage.Store
	workers  int
}

// Constructor for the Scheduler
func New(monitors []domain.Monitor, checker Checker, store *storage.Store, workers int) *Scheduler {
	return &Scheduler{
		monitors: monitors,
		checker:  checker,
		store:    store,
		workers:  workers,
	}
}

// Run once immediately and then repeat every 30 seconds with a ticker until context is canceled.
func (s *Scheduler) Start(ctx context.Context) {
	// Run once immediately so the user has immediate output
	s.run(ctx)

	// Create ticker to run the scheduler every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop() // clean up ticker when context is canceled

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.run(ctx)
		}
	}
}

// Dispatch all monitors to worker goroutines
func (s *Scheduler) run(ctx context.Context) {
	// Buffered channel to hold all jobs without blocking until a worker picks it up
	jobs := make(chan domain.Monitor, len(s.monitors))
	var wg sync.WaitGroup

	// Spawn anonymous worker goroutines that pull monitors to check from the jobs channel
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for monitor := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					// Do nothing and just move on to the core code below until the context is canceled
				}
				result := s.checker.Check(ctx, monitor)
				state := "DOWN"
				if result.IsUp {
					state = "UP"
				}
				// Print the result to stdout
				fmt.Printf("[%s] %s %s %d %s\n", state, monitor.Name, monitor.URL, result.Status, result.Latency.Round(time.Millisecond))
				if err := s.store.SaveResult(result); err != nil {
					fmt.Printf("Error saving result: %v\n", err)
				}
			}
		}()
	}

	// Send all monitors into the jobs channel for processing
	for _, m := range s.monitors {
		jobs <- m
	}
	close(jobs)
	wg.Wait()

	fmt.Println("--- Run complete ---")
}
