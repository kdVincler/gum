package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kdVincler/gum/internal/checker"
	"github.com/kdVincler/gum/internal/domain"
	"github.com/kdVincler/gum/internal/scheduler"
	"github.com/kdVincler/gum/internal/server"
	"github.com/kdVincler/gum/internal/storage"
)

func main() {
	fmt.Println("gum - uptime monitor starting...")

	// Hardcode the endpoints to monitor
	http_monitors := []domain.Monitor{
		{ID: "1", Name: "HTTPBin OK", URL: "https://httpbin.org/status/200"},
		{ID: "2", Name: "HTTPBin Fail", URL: "https://httpbin.org/status/500"},
		{ID: "3", Name: "Go Website", URL: "https://go.dev"},
	}
	tcp_monitors := []domain.Monitor{
		{ID: "4", Name: "Googe TLS", URL: "google.com:443"},
	}
	all_monitors := append(http_monitors, tcp_monitors...) // combined monitors list for the server

	// Set up storage
	str, err := storage.New("gum.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer str.Close()

	// Create both checkers with a 5 second timeout per req
	http_chkr := checker.NewHTTPChecker(5 * time.Second)
	tcp_chkr := checker.NewTCPChecker(5 * time.Second)

	// Create both schedulers with their respective monitor slices and checkers
	http_schd := scheduler.New(http_monitors, http_chkr, str, 3) // 3 monitors -> 3 workers
	tcp_schd := scheduler.New(tcp_monitors, tcp_chkr, str, 1)    // 1 monitor -> 1 worker

	// Create a server instance for ALL monitors that serves the status page and API
	srv := server.New(str, all_monitors, ":8080")

	// Create a cancelable context to allow for graceful shutdown of workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server in its own goroutine via an anonymous function
	go func() {
		fmt.Println("status page: http://localhost:8080")
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			// log server errors (apart from ErrServerClosed; that's expected during the shutdown period)
			log.Printf("server error %v", err)
		}
	}()

	// Start both schedulers in their own goroutines
	go http_schd.Start(ctx)
	go tcp_schd.Start(ctx)

	// Server and Schedulers are running in their own goroutintes,
	// now we need to block main process until the user presses Ctrl+C to shut down
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM) // put Ctrl+C and kill signals into the channel
	<-sigCh                                               // block until a signal is received

	fmt.Println("\nReceived shutdown signal, exiting...")
	cancel()                           // explicitly cancel the context
	srv.Shutdown(context.Background()) // shut down HTTP server
	fmt.Println("goodbye!")
}
