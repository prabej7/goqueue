package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
)

type ClaimNextJobPayload struct {
	WorkerID        uuid.UUID
	LeaseExpiration time.Time
	ClaimToken      uuid.UUID
}

type Repository interface {
	Create(
		ctx context.Context,
		job *model.Job,
	) (*model.Job, error)
	GetByID(ctx context.Context,
		ID uuid.UUID,
	) (*model.Job, error)
	CompleteJob(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
	) error
	FailJob(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
		failErr error,
		retryAt time.Time,
	) (*model.Job, error)
	ClaimNextJob(
		ctx context.Context,
		payload ClaimNextJobPayload,
	) (*model.Job, error)
	UpdateLastError(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
		lastErr string,
	) error
	RenewLease(
		ctx context.Context,
		jobID uuid.UUID,
		claimToken string,
		leaseExpiration time.Time,
	) error
}
