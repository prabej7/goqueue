package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/config"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/job/repository"
)

func loadConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		// t.Fatalf("error loading config: %+v", err)
	}

	return cfg
}

func TestProcessNextJob_Success(t *testing.T) {
	job := &model.Job{
		ID:      uuid.New(),
		Type:    "email.send",
		Payload: json.RawMessage(`{"message":"hello"}`),
		Status:  model.JobStatusPending,
	}

	repo := &repository.MockJobRepository{
		Jobs: []*model.Job{job},
	}

	registry := Registry{}

	registry.Register("email.send", func(ctx context.Context, payload json.RawMessage) error {
		return nil
	})

	cfg := loadConfig(t)

	w := NewWorker(
		uuid.New(),
		repo,
		&registry,
		cfg,
	)

	ctx := t.Context()
	err := w.processNextJob(ctx)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	processedJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Errorf("error on GetByID(): %+v", err)
	}

	if processedJob.Status != model.JobStatusCompleted {
		t.Fatal("expected job to be completed")
	}

	if processedJob.Status == model.JobStatusFailed {
		t.Fatal("expected job not to be failed")
	}
}

func TestProcessNextJob_HandlerFailure(t *testing.T) {
	job := &model.Job{
		ID:      uuid.New(),
		Type:    "email.send",
		Payload: json.RawMessage(`{"message":"hello"}`),
		Status:  model.JobStatusPending,
	}

	repo := &repository.MockJobRepository{
		Jobs: []*model.Job{job},
	}

	registry := Registry{}
	cfg := loadConfig(t)

	handlerMockErr := errors.New("something went wrong")
	registry.Register(
		"email.send",
		func(
			ctx context.Context,
			payload json.RawMessage,
		) error {
			return handlerMockErr
		})

	w := NewWorker(
		uuid.New(),
		repo,
		&registry,
		cfg,
	)

	ctx := t.Context()
	err := w.processNextJob(ctx)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	processedJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Errorf("error on GetByID(): %+v", err)
	}

	if processedJob.Status != model.JobStatusDead {
		t.Errorf("expected status %v, got %v", model.JobStatusDead, processedJob.Status)
	}

	if processedJob.LastError == nil {
		t.Fatalf("expetect last error, got: %+v", processedJob.LastError)
	} else if *processedJob.LastError != "something went wrong" {
		t.Fatalf("expected %s, got %s", handlerMockErr, *processedJob.LastError)
	}

}

func TestProcessNextJob_NoHandler(t *testing.T) {
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "email.send",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		MaxAttempts: 3,
	}

	repo := &repository.MockJobRepository{
		Jobs: []*model.Job{job},
	}

	registry := Registry{}
	cfg := loadConfig(t)

	w := NewWorker(
		uuid.New(),
		repo,
		&registry,
		cfg,
	)

	ctx := t.Context()
	err := w.processNextJob(ctx)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	processedJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Errorf("error on GetByID(): %+v", err)
	}

	if processedJob.Status != model.JobStatusFailed {
		t.Fatalf("expected job to be failed, got %v", processedJob.Status)
	}

	if processedJob.LastError == nil {
		t.Fatalf("expetect last error, got: %+v", processedJob.LastError)
	} else if *processedJob.LastError != ErrNoHandlerFound.Error() {
		t.Fatalf("expected %s, got %s", ErrNoHandlerFound, *processedJob.LastError)
	}
}

type claimLosingRepository struct {
	*repository.MockJobRepository
}

func (r *claimLosingRepository) CompleteJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
) error {
	job, err := r.GetByID(ctx, ID)
	if err != nil {
		return err
	}

	// Simulate another worker taking ownership.
	newToken := uuid.New().String()
	job.ClaimToken = &newToken

	// The original worker's token should now be rejected.
	return r.MockJobRepository.CompleteJob(ctx, ID, claimToken)
}

func TestProcessNextJob_ClaimLost(t *testing.T) {
	job := &model.Job{
		ID:      uuid.New(),
		Type:    "email.send",
		Payload: json.RawMessage(`{"message":"hello"}`),
		Status:  model.JobStatusPending,
	}

	repo := &claimLosingRepository{
		MockJobRepository: &repository.MockJobRepository{
			Jobs: []*model.Job{job},
		},
	}

	registry := Registry{}

	registry.Register("email.send", func(
		ctx context.Context,
		payload json.RawMessage,
	) error {
		return nil
	})

	cfg := loadConfig(t)
	w := NewWorker(uuid.New(), repo, &registry, cfg)

	err := w.processNextJob(t.Context())

	if !errors.Is(err, repository.ErrJobClaimLost) {
		t.Fatalf(
			"expected ErrJobClaimLost, got: %v",
			err,
		)
	}

	processedJob, err := repo.GetByID(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("error on GetByID(): %v", err)
	}

	if processedJob.Status != model.JobStatusProcessing {
		t.Fatalf(
			"expected processing status, got: %s",
			processedJob.Status,
		)
	}
}

type mockJobRepository struct {
	claimErr error
}

var _ JobRepository = (*mockJobRepository)(nil)

func (m *mockJobRepository) ClaimNextJob(
	ctx context.Context,
	payload repository.ClaimNextJobPayload,
) (*model.Job, error) {
	return nil, m.claimErr
}

func (m *mockJobRepository) CompleteJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
) error {
	return nil
}

func (m *mockJobRepository) FailJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	failErr error,
	retryAt time.Time,
) (*model.Job, error) {
	return nil, nil
}

func (m *mockJobRepository) UpdateLastError(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	err string,
) error {
	return nil
}

func (m *mockJobRepository) RenewLease(
	ctx context.Context,
	jobID uuid.UUID,
	claimToken string,
	leaseExpiration time.Time,
) error {
	return nil
}
func TestProcessNextJob_ClaimError(t *testing.T) {
	mockErr := errors.New("database connection failed")

	repo := &mockJobRepository{
		claimErr: mockErr,
	}

	registry := Registry{}
	cfg := loadConfig(t)

	w := NewWorker(uuid.New(), repo, &registry, cfg)

	err := w.processNextJob(t.Context())

	if !errors.Is(err, mockErr) {
		t.Fatalf("expected %v, got %v", mockErr, err)
	}
}

func TestProcessNextJob_NoJobsAvailable(t *testing.T) {
	repo := &repository.MockJobRepository{}

	registry := Registry{}
	cfg := loadConfig(t)

	w := NewWorker(uuid.New(), repo, &registry, cfg)

	err := w.processNextJob(t.Context())

	if !errors.Is(err, repository.ErrNoJobsAvailable) {
		t.Fatalf(
			"expected ErrNoJobsAvailable, got: %v",
			err,
		)
	}
}

func TestProcessNextJob_Heartbeat(t *testing.T) {
	job := &model.Job{
		ID:      uuid.New(),
		Type:    "email.send",
		Payload: json.RawMessage(`{"message":"hello"}`),
		Status:  model.JobStatusPending,
	}
	repo := &repository.MockJobRepository{
		Jobs: []*model.Job{job},
	}

	registry := Registry{}

	cfg := loadConfig(t)
	cfg.HeartbeatInterval = 20 * time.Millisecond
	cfg.LeaseDuration = time.Minute
	cfg.JobTimeout = time.Second

	registry.Register(
		"email.send",
		func(ctx context.Context, payload json.RawMessage) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)

	w := NewWorker(uuid.New(), repo, &registry, cfg)

	err := w.processNextJob(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if repo.RenewLeaseCalls == 0 {
		t.Fatal("expected heartbeat to renew the lease at least once")
	}
}

func TestComputeBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempts int
		base     time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{
			name:     "first attempt",
			attempts: 1,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 5 * time.Second,
		},
		{
			name:     "second attempt",
			attempts: 2,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 10 * time.Second,
		},
		{
			name:     "third attempt",
			attempts: 3,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 20 * time.Second,
		},
		{
			name:     "fourth attempt",
			attempts: 4,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 40 * time.Second,
		},
		{
			name:     "fifth attempt",
			attempts: 5,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 80 * time.Second,
		},
		{
			name:     "sixth attempt",
			attempts: 6,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 160 * time.Second,
		},
		{
			name:     "maximum backoff reached",
			attempts: 7,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 5 * time.Minute,
		},
		{
			name:     "zero attempts",
			attempts: 0,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 0,
		},
		{
			name:     "negative attempts",
			attempts: -1,
			base:     5 * time.Second,
			max:      5 * time.Minute,
			expected: 0,
		},
		{
			name:     "zero base delay",
			attempts: 3,
			base:     0,
			max:      5 * time.Minute,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeBackoff(
				tt.attempts,
				tt.base,
				tt.max,
			)

			if got != tt.expected {
				t.Fatalf(
					"expected %v, got %v",
					tt.expected,
					got,
				)
			}
		})
	}
}

type backoffTestRepository struct {
	job        *model.Job
	retryAt    time.Time
	failErr    error
	failCalled bool
}

var _ JobRepository = (*backoffTestRepository)(nil)

func (r *backoffTestRepository) ClaimNextJob(
	ctx context.Context,
	payload repository.ClaimNextJobPayload,
) (*model.Job, error) {
	return r.job, nil
}

func (r *backoffTestRepository) FailJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	failErr error,
	retryAt time.Time,
) (*model.Job, error) {
	r.failCalled = true
	r.retryAt = retryAt
	r.failErr = failErr

	return nil, nil
}

func (r *backoffTestRepository) CompleteJob(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
) error {
	return nil
}

func (r *backoffTestRepository) UpdateLastError(
	ctx context.Context,
	ID uuid.UUID,
	claimToken string,
	err string,
) error {
	return nil
}

func (r *backoffTestRepository) RenewLease(
	ctx context.Context,
	jobID uuid.UUID,
	claimToken string,
	leaseExpiration time.Time,
) error {
	return nil
}

func TestProcessNextJob_BackoffRetry(t *testing.T) {
	tests := []struct {
		name     string
		attempts int
		expected time.Duration
	}{
		{
			name:     "first retry",
			attempts: 1,
			expected: 5 * time.Second,
		},
		{
			name:     "second retry",
			attempts: 2,
			expected: 10 * time.Second,
		},
		{
			name:     "third retry",
			attempts: 3,
			expected: 20 * time.Second,
		},
		{
			name:     "maximum backoff",
			attempts: 7,
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &model.Job{
				ID:          uuid.New(),
				Type:        "email.send",
				Payload:     json.RawMessage(`{"message":"hello"}`),
				Status:      model.JobStatusPending,
				Attempts:    tt.attempts,
				MaxAttempts: 10,
			}

			repo := &backoffTestRepository{
				job: job,
			}

			registry := Registry{}

			handlerErr := errors.New("email delivery failed")

			registry.Register(
				"email.send",
				func(
					ctx context.Context,
					payload json.RawMessage,
				) error {
					return handlerErr
				},
			)

			cfg := loadConfig(t)
			cfg.LeaseDuration = time.Minute
			cfg.JobTimeout = time.Second
			cfg.HeartbeatInterval = time.Hour
			cfg.RenewTimeout = time.Second

			w := NewWorker(
				uuid.New(),
				repo,
				&registry,
				cfg,
			)

			before := time.Now()

			err := w.processNextJob(context.Background())

			after := time.Now()

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if !repo.failCalled {
				t.Fatal("expected FailJob to be called")
			}

			if !errors.Is(repo.failErr, handlerErr) {
				t.Fatalf(
					"expected handler error %v, got %v",
					handlerErr,
					repo.failErr,
				)
			}

			expectedEarliest := before.Add(tt.expected)
			expectedLatest := after.Add(tt.expected)

			if repo.retryAt.Before(expectedEarliest) ||
				repo.retryAt.After(expectedLatest) {
				t.Fatalf(
					"expected retryAt around %v, got %v",
					expectedEarliest,
					repo.retryAt,
				)
			}
		})
	}
}
