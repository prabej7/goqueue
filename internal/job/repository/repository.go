package repository

import (
	"context"

	"github.com/prabej7/goqueue/internal/job/model"
)

type Repository interface {
	Create(ctx context.Context, job *model.Job) (*model.Job, error)
	GetByID(ctx context.Context, ID string) (*model.Job, error)
}
