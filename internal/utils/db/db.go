package db

import (
	"context"
	"database/sql"
)

func WithTx[T any](
	db *sql.DB,
	ctx context.Context,
	fn func(*sql.Tx) (T, error),
) (T, error) {
	var zero T

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}

	result, err := fn(tx)
	if err != nil {
		tx.Rollback()
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, err
	}

	return result, nil
}
