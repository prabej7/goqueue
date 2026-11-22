package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prabej7/goqueue/internal/config"
	"github.com/prabej7/goqueue/internal/database"
	"github.com/prabej7/goqueue/internal/job/bootstrap"
	"github.com/prabej7/goqueue/internal/server"
	"github.com/prabej7/goqueue/internal/store"
	"github.com/prabej7/goqueue/internal/worker"
)

func main() {
	// Application context
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("config loaded successfully")

	// Initialize database
	db, err := database.NewPostgresDB(
		ctx,
		cfg.DatabaseConfig.DatabaseURL,
		cfg.DatabaseConfig.MaxOpenConnections,
		cfg.DatabaseConfig.MaxIdleConnections,
		cfg.DatabaseConfig.MaxConnectionIdleTime,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Print("db initialized successfully")

	// store := store.NewMockStore()
	store := store.NewPostgresStore(db)
	// Initialize HTTP server
	handler := server.New(server.Dependencies{
		DB:    db,
		Store: store,
	})

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	// Register job handlers
	registry := bootstrap.NewWorkerRegistry()

	// Start HTTP server
	serverErr := make(chan error, 1)

	go func() {
		log.Printf("GoQueue API starting on port %s", cfg.Port)
		serverErr <- httpServer.ListenAndServe()
	}()

	// Start worker
	const numWorkers = 2
	workerDone := worker.InitializeWorkers(
		ctx,
		*store.JobRepository,
		registry,
		cfg,
		numWorkers,
	)

	remainingWorkers := numWorkers

	// Wait for shutdown event
	select {
	case <-ctx.Done():
		log.Print("shutdown signal received")

	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
		stop()

	case err := <-workerDone:
		remainingWorkers--
		if err != nil {
			log.Printf("worker exited unexpectedly: %v", err)
		} else {
			log.Print("worker exited")
		}
		stop()
	}

	// Create shutdown deadline
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	// Gracefully shut down HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)

		// Force close remaining connections if graceful shutdown fails.
		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("HTTP force close error: %v", closeErr)
		}
	}

	// Wait for worker to finish
	for remainingWorkers > 0 {
		select {
		case err := <-workerDone:
			remainingWorkers--
			if err != nil {
				log.Printf("worker exited with error: %v", err)
			} else {
				log.Print("worker exited cleanly")
			}
		case <-shutdownCtx.Done():
			log.Printf("worker shutdown timed out with %d worker(s) still running", remainingWorkers)
			remainingWorkers = 0 // break out
		}
	}

	log.Print("GoQueue shutting down")
}
