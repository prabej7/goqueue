package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
)

type Repository interface {
	Create(ctx context.Context, job *model.Job) (*model.Job, error)
	GetByID(ctx context.Context, ID uuid.UUID) (*model.Job, error)
}
