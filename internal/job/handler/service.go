package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/job/service"
)

type JobService interface {
	CreateJob(ctx context.Context, job *service.CreateJobInput) (*model.Job, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error)
}
