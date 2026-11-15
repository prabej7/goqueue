package repository

import (
	"context"

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
