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

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/config"
	"github.com/prabej7/goqueue/internal/database"
	"github.com/prabej7/goqueue/internal/job/bootstrap"
	"github.com/prabej7/goqueue/internal/job/repository"
	"github.com/prabej7/goqueue/internal/server"
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

	// Initialize HTTP server
	handler := server.New(server.Dependencies{
		DB: db,
	})

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	// Register job handlers
	registry := bootstrap.NewWorkerRegistry()

	// Initialize worker
	jobWorker := worker.NewWorker(
		uuid.New(),
		repository.NewJobRepository(db),
		registry,
	)

	// Start HTTP server
	serverErr := make(chan error, 1)

	go func() {
		log.Printf("GoQueue API starting on port %s", cfg.Port)
		serverErr <- httpServer.ListenAndServe()
	}()

	// Start worker
	workerDone := make(chan error, 1)

	go func() {
		workerDone <- jobWorker.Start(ctx)
	}()

	workerExited := false

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
		if err != nil {
			log.Printf("worker exited unexpectedly: %v", err)
		}
		workerExited = true
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
	if !workerExited {
		select {
		case err := <-workerDone:
			if err != nil {
				log.Printf("worker exited with error: %v", err)
			}
			log.Print("worker shutdown complete")

		case <-shutdownCtx.Done():
			log.Print("worker shutdown timed out")
		}
	}

	log.Print("GoQueue shutting down")
}
