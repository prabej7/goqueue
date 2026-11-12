package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prabej7/goqueue/internal/config"
	"github.com/prabej7/goqueue/internal/database"
	"github.com/prabej7/goqueue/internal/server"
)

func main() {
	cfg := config.Load()

	_, err := database.NewPostgresDB(
		cfg.DatabaseConfig.DatabaseURL,
		cfg.DatabaseConfig.MaxOpenConnection,
		cfg.DatabaseConfig.MaxIdleConnections,
		cfg.DatabaseConfig.MaxConnectionIdleTime,
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Print("db initialized successfully")

	handler := server.New()

	log.Printf("GoQueue API starting on port %s", cfg.Port)

	err = http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler)

	if err != nil {
		log.Fatal(err)
	}
}
