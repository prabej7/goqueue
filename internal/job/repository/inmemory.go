package repository

import (
	"context"
	"time"

	"github.com/prabej7/goqueue/internal/job/model"
)

type MockJobRepository struct {
	Jobs      []*model.Job
	CreateErr error
}

var _ Repository = (*MockJobRepository)(nil)

func NewMockRepository() *MockJobRepository {
	return &MockJobRepository{
		Jobs: []*model.Job{},
	}
}

func (m *MockJobRepository) Create(ctx context.Context, job *model.Job) (*model.Job, error) {

	if m.CreateErr != nil {
		return nil, m.CreateErr
	}

	m.Jobs = append(m.Jobs, job)

	return job, nil
}

func (m *MockJobRepository) GetByID(ctx context.Context, ID string) (*model.Job, error) {
	for _, job := range m.Jobs {
		if job.ID.String() == ID {
			return job, nil
		}
	}

	return nil, ErrJobNotFound
}

func (m *MockJobRepository) ClaimNextJob(
	ctx context.Context,
	payload ClaimNextJobPayload,
) (*model.Job, error) {

	now := time.Now()

	var selected *model.Job

	for _, job := range m.Jobs {
		if job.Status != model.JobStatusPending {
			continue
		}

		if job.ScheduledAt.After(now) {
			continue
		}

		if selected == nil ||
			job.Priority > selected.Priority ||
			(job.Priority == selected.Priority &&
				job.ScheduledAt.Before(selected.ScheduledAt)) ||
			(job.Priority == selected.Priority &&
				job.ScheduledAt.Equal(selected.ScheduledAt) &&
				job.CreatedAt.Before(selected.CreatedAt)) {

			selected = job
		}
	}

	if selected == nil {
		return nil, ErrNoJobsAvailable
	}

	selected.Status = model.JobStatusProcessing
	selected.LockedBy = payload.WorkerID.String()

	lockedAt := now
	selected.LockedAt = lockedAt

	selected.LeaseExpiresAt = payload.LeaseExpiration
	selected.ClaimToken = payload.ClaimToken.String()

	selected.UpdatedAt = now

	return selected, nil
}
