package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
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

func (r *JobRepository) GetByID(
	ctx context.Context,
	ID uuid.UUID,
) (*model.Job, error) {
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
			locked_by,
			locked_at,
			lease_expires_at,
			claim_token,
			result,
			last_error,
			started_at,
			completed_at,
			created_at,
			updated_at
		FROM jobs
		WHERE id = $1
	`

	job := &model.Job{}

	err := r.db.QueryRowContext(
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
		&job.LockedBy,
		&job.LockedAt,
		&job.LeaseExpiresAt,
		&job.ClaimToken,
		&job.Result,
		&job.LastError,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
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
				locked_by,
				locked_at, 
				lease_expires_at,
				claim_token,
				result,
				last_error,
				started_at,
				completed_at,
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
			&job.LockedBy,
			&job.LockedAt,
			&job.LeaseExpiresAt,
			&job.ClaimToken,
			&job.Result,
			&job.LastError,
			&job.StartedAt,
			&job.CompletedAt,
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
				claim_token = $4,
				attempts = attempts + 1
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

func (r *JobRepository) CompleteJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
) error {
	return r.updateClaimedJobStatus(
		ctx,
		ID,
		model.JobStatusCompleted,
		claimToken,
	)
}

func (r *JobRepository) FailJob(ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	failErr error,
	retryAt time.Time,
) (*model.Job, error) {
	query := `
				UPDATE jobs
				SET
					status = CASE
						WHEN attempts >= max_attempts THEN 'dead'
						ELSE 'pending'
					END,
					last_error = $3,
					scheduled_at = CASE
						WHEN attempts >= max_attempts THEN scheduled_at
						ELSE $4
						END,
								updated_at = NOW(),
					locked_by = NULL,
					locked_at = NULL,
					lease_expires_at = NULL,
					claim_token = NULL
				WHERE
					id = $1
					AND claim_token = $2
					AND status = 'processing'
				RETURNING 
					id,
					job_type,
					payload,
					status,
					priority,
					attempts,
					max_attempts,
					scheduled_at,
					locked_by,
					locked_at, 
					lease_expires_at,
					claim_token,
					result,
					last_error,
					started_at,
					completed_at,
					created_at,
					updated_at;
			`
	job := model.Job{}
	err := r.db.QueryRowContext(
		ctx,
		query,
		ID,
		claimToken,
		failErr.Error(),
		retryAt,
	).Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Status,
		&job.Priority,
		&job.Attempts,
		&job.MaxAttempts,
		&job.ScheduledAt,
		&job.LockedBy,
		&job.LockedAt,
		&job.LeaseExpiresAt,
		&job.ClaimToken,
		&job.Result,
		&job.LastError,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobClaimLost
		}
		return nil, err
	}

	return &job, nil

}

func (r *JobRepository) updateClaimedJobStatus(
	ctx context.Context,
	ID uuid.UUID,
	status model.JobStatus,
	claimToken string,
) error {
	query := `
		UPDATE jobs
		SET 
			status = $2,
			updated_at = NOW(),
			locked_by = NULL,
			locked_at = NULL,
			lease_expires_at = NULL,
			claim_token = NULL
		WHERE 
			id = $1
			AND status = 'processing'
			AND claim_token = $3;
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		ID,
		status,
		claimToken,
	)

	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return ErrJobClaimLost
	}

	return nil

}

func (r *JobRepository) UpdateLastError(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	lastErr string,
) error {
	query := `
		UPDATE jobs
		SET last_error = $3
		WHERE
			id = $1
			AND claim_token = $2;
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		ID,
		claimToken,
		lastErr,
	)

	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return ErrJobClaimLost
	}

	return nil
}

func (r *JobRepository) RenewLease(
	ctx context.Context,
	jobID uuid.UUID,
	claimToken string,
	leaseExpiration time.Time,
) error {
	query := `
		UPDATE jobs
		SET 
			lease_expires_at = $2
		WHERE 
			id = $1
			AND claim_token = $3
			AND status = 'processing';
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		jobID,
		leaseExpiration,
		claimToken,
	)

	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return ErrJobClaimLost
	}

	return nil
}
