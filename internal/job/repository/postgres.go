package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{
		db: db,
	}
}

var _ Repository = (*JobRepository)(nil)

func (r *JobRepository) Create(
	ctx context.Context,
	job *model.Job,
) (*model.Job, error) {
	query := `
		INSERT INTO jobs
		(id, job_type, payload, status, priority, attempts, max_attempts, scheduled_at)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	if err := r.db.QueryRowContext(
		ctx,
		query,
		job.ID,
		job.Type,
		job.Payload,
		job.Status,
		job.Priority,
		job.Attempts,
		job.MaxAttempts,
		job.ScheduledAt,
	).Scan(
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *JobRepository) GetByID(ctx context.Context, ID uuid.UUID) (*model.Job, error) {
	query := `
		SELECT
			id,
			job_type,
			payload,
			status,
			priority,
			attempts,
			max_attempts,
			scheduled_at,
			created_at,
			updated_at
		FROM jobs
		WHERE id = $1
	`
	job := &model.Job{}
	if err := r.db.QueryRowContext(
		ctx,
		query,
		ID,
	).Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Status,
		&job.Priority,
		&job.Attempts,
		&job.MaxAttempts,
		&job.ScheduledAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}

		return nil, err
	}

	return job, nil
}
