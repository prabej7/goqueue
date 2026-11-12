package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresDB(
	ctx context.Context,
	addr string,
	maxOpenConnection int,
	maxIdleConnections int,
	maxConnectionIdleTime time.Duration,
) (*sql.DB, error) {
	db, err := sql.Open("pgx", addr)

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConnection)
	db.SetMaxIdleConns(maxIdleConnections)

	db.SetConnMaxIdleTime(maxConnectionIdleTime)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
