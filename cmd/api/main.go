package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/prabej7/goqueue/internal/config"
	"github.com/prabej7/goqueue/internal/database"
	"github.com/prabej7/goqueue/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("config loaded successfully")

	ctx := context.Background()

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

	handler := server.New(
		server.Dependencies{
			DB: db,
		},
	)

	log.Printf("GoQueue API starting on port %s", cfg.Port)

	err = http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler)

	if err != nil {
		log.Fatal(err)
	}
}
