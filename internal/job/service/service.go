package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/job/repository"
)

type Service struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *Service {
	return &Service{
		repo,
	}
}

type CreateJobInput struct {
	JobType  string
	Payload  json.RawMessage
	Priority int
}

func (s *Service) CreateJob(ctx context.Context, job *CreateJobInput) (*model.Job, error) {
	//generaing a new uuid
	id := uuid.New()
	createJob, err := s.repo.Create(ctx, &model.Job{
		ID:          id,
		Type:        job.JobType,
		Payload:     job.Payload,
		Status:      model.JobStatusPending,
		Priority:    job.Priority,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
	})

	if err != nil {
		return nil, err
	}

	return createJob, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*model.Job, error) {
	return s.repo.GetByID(ctx, id)
}
