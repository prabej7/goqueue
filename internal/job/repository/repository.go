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
	Create(ctx context.Context, job *model.Job) (*model.Job, error)
	GetByID(ctx context.Context, ID string) (*model.Job, error)
	ClaimNextJob(ctx context.Context, payload ClaimNextJobPayload) (*model.Job, error)
}
