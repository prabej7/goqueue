package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
)

type MockJobRepository struct {
	Jobs            []*model.Job
	CreateErr       error
	RenewLeaseCalls int
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

func (m *MockJobRepository) GetByID(ctx context.Context, ID uuid.UUID) (*model.Job, error) {
	for _, job := range m.Jobs {
		if job.ID == ID {
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
	lockedBy := payload.WorkerID.String()
	selected.LockedBy = &lockedBy

	lockedAt := now
	selected.LockedAt = &lockedAt

	selected.LeaseExpiresAt = &payload.LeaseExpiration
	claimToken := payload.ClaimToken.String()
	selected.ClaimToken = &claimToken

	selected.UpdatedAt = now

	return selected, nil
}

func (m *MockJobRepository) CompleteJob(ctx context.Context, ID uuid.UUID, claimToken string) error {
	return m.updateClaimedJobStatus(
		ctx,
		ID,
		model.JobStatusCompleted,
		claimToken,
	)
}

func (m *MockJobRepository) FailJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	failErr error,
	retryAt time.Time,
) (*model.Job, error) {
	job, err := m.GetByID(
		ctx,
		ID,
	)

	if err != nil {
		return nil, err
	}

	if *job.ClaimToken != claimToken {
		return nil, ErrJobClaimLost
	}

	errString := failErr.Error()
	job.LastError = &errString

	if job.Attempts == job.MaxAttempts {
		job.Status = model.JobStatusDead
	} else {
		job.Status = model.JobStatusFailed
	}

	return job, nil
}

func (m *MockJobRepository) updateClaimedJobStatus(
	ctx context.Context,
	ID uuid.UUID,
	status model.JobStatus,
	claimToken string,
) error {
	job, err := m.GetByID(ctx, ID)
	if err != nil {
		return err
	}

	if job.ClaimToken != nil && *job.ClaimToken == claimToken {
		job.Status = status
	} else {
		return ErrJobClaimLost
	}

	return nil
}

func (m *MockJobRepository) UpdateLastError(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	lastError string,
) error {
	job, err := m.GetByID(ctx, ID)
	if err != nil {
		return err
	}

	if job.ClaimToken != nil && *job.ClaimToken == claimToken {
		job.LastError = &lastError
	} else {
		return ErrJobClaimLost
	}

	job.LastError = &lastError

	return nil
}

func (m *MockJobRepository) RenewLease(
	ctx context.Context,
	jobID uuid.UUID,
	claimToken string,
	leaseExpiration time.Time,
) error {
	m.RenewLeaseCalls++
	job, err := m.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if job == nil {
		return ErrJobNotFound
	}

	if *job.ClaimToken != claimToken {
		return ErrJobClaimLost
	}

	if job.Status != model.JobStatusProcessing {
		return ErrJobClaimLost
	}

	job.LeaseExpiresAt = &leaseExpiration

	return nil
}

func (m *MockJobRepository) GetAll(
	ctx context.Context,
	pagination PaginationParams,
) ([]*model.Job, error) {
	return m.Jobs, nil
}
