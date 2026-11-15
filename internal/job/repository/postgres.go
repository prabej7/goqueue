package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/utils/db"
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

func (r *JobRepository) GetByID(ctx context.Context, ID string) (*model.Job, error) {
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

func (r *JobRepository) ClaimNextJob(
	ctx context.Context,
	payload ClaimNextJobPayload,
) (*model.Job, error) {
	return db.WithTx(r.db, ctx, func(tx *sql.Tx) (*model.Job, error) {
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
			WHERE status = 'pending'
				AND scheduled_at <= NOW()
			ORDER BY priority DESC, scheduled_at ASC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED;
		`
		var job model.Job
		err := tx.QueryRowContext(ctx, query).Scan(
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
		)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrNoJobsAvailable
			}
			return nil, err
		}

		query = `
			UPDATE jobs
			SET 
				status = 'processing',
				locked_by = $2,
				locked_at = NOW(),
				lease_expires_at = $3,
				claim_token = $4
			WHERE id = $1
			RETURNING status, locked_by, locked_at, lease_expires_at, claim_token;
		`

		err = tx.QueryRowContext(
			ctx,
			query,
			job.ID,
			payload.WorkerID,
			payload.LeaseExpiration,
			payload.ClaimToken,
		).Scan(
			&job.Status,
			&job.LockedBy,
			&job.LockedAt,
			&job.LeaseExpiresAt,
			&job.ClaimToken,
		)

		if err != nil {
			return nil, err
		}

		return &job, nil
	})
}
