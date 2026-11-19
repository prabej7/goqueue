package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/job/repository"
	"github.com/prabej7/goqueue/internal/job/service"
)

func newTestService(t *testing.T) (context.Context, *service.Service) {
	return t.Context(), service.NewService(repository.NewMockRepository())
}

func assertJobsEqual(t *testing.T, got, want *model.Job) {
	t.Helper()

	if got == nil {
		t.Fatal("got job must not be nil")
	}
	if want == nil {
		t.Fatal("want job must not be nil")
	}
	assertTimesEqual := func(name string, got, want *time.Time) {
		t.Helper()
		if (got == nil) != (want == nil) {
			t.Fatalf("job.%s nilness differs: got %v, want %v", name, got == nil, want == nil)
		}
		if got != nil && !got.Equal(*want) {
			t.Fatalf("job.%s = %v, want %v", name, got, want)
		}
	}
	if got.ID != want.ID {
		t.Fatalf("job.ID = %q, want %q", got.ID, want.ID)
	}
	if got.Type != want.Type {
		t.Fatalf("job.Type = %q, want %q", got.Type, want.Type)
	}
	var gotPayload, wantPayload any

	if err := json.Unmarshal(got.Payload, &gotPayload); err != nil {
		t.Fatalf("failed to unmarshal got payload: %v", err)
	}

	if err := json.Unmarshal(want.Payload, &wantPayload); err != nil {
		t.Fatalf("failed to unmarshal want payload: %v", err)
	}

	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Fatalf("job.Payload = %s, want %s", got.Payload, want.Payload)
	}
	if got.Priority != want.Priority {
		t.Fatalf("job.Priority = %d, want %d", got.Priority, want.Priority)
	}
	if got.Attempts != want.Attempts {
		t.Fatalf("job.Attempts = %d, want %d", got.Attempts, want.Attempts)
	}
	if got.MaxAttempts != want.MaxAttempts {
		t.Fatalf("job.MaxAttempts = %d, want %d", got.MaxAttempts, want.MaxAttempts)
	}
	if got.Status != want.Status {
		t.Fatalf("job.Status = %q, want %q", got.Status, want.Status)
	}
	if got.LockedBy != want.LockedBy {
		t.Fatalf("job.LockedBy = %v, want %v", got.LockedBy, want.LockedBy)
	}
	assertTimesEqual("LockedAt", got.LockedAt, want.LockedAt)
	assertTimesEqual("LeaseExpiresAt", got.LeaseExpiresAt, want.LeaseExpiresAt)
	if got.ClaimToken != want.ClaimToken {
		t.Fatalf("job.ClaimToken = %v, want %v", got.ClaimToken, want.ClaimToken)
	}
	if !reflect.DeepEqual(got.Result, want.Result) {
		t.Fatalf("job.Result = %s, want %s", got.Result, want.Result)
	}
	assertTimesEqual("StartedAt", got.StartedAt, want.StartedAt)
	assertTimesEqual("CompletedAt", got.CompletedAt, want.CompletedAt)
	if !got.ScheduledAt.Equal(want.ScheduledAt) {
		t.Fatalf("job.ScheduledAt = %v, want %v", got.ScheduledAt, want.ScheduledAt)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("job.CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("job.UpdatedAt = %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func TestService_CreateJob(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload := &service.CreateJobInput{
		JobType:  "test",
		Payload:  json.RawMessage(`{"message":"Hello"}`),
		Priority: 1,
	}

	createdJob, err := jobService.CreateJob(ctx, jobPayload)

	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	if createdJob == nil {
		t.Fatalf("createdJob must not be nil")
	}

	if createdJob.ID == uuid.Nil {
		t.Fatalf("createdJob.ID must not be nil")
	}

	if createdJob.Type != jobPayload.JobType {
		t.Fatalf("createdJob.Type = %q, want %q", createdJob.Type, jobPayload.JobType)
	}

	if !json.Valid(createdJob.Payload) {
		t.Fatalf("createdJob.Payload must contain valid JSON: %s", createdJob.Payload)
	}

	if string(createdJob.Payload) != string(jobPayload.Payload) {
		t.Fatalf("createdJob.Payload = %s, want %s", createdJob.Payload, jobPayload.Payload)
	}

	if createdJob.Priority != jobPayload.Priority {
		t.Fatalf("createdJob.Priority = %d, want %d", createdJob.Priority, jobPayload.Priority)
	}

	if createdJob.Attempts != 0 {
		t.Fatalf("createdJob.Attempts = %d, want 0", createdJob.Attempts)
	}

	if createdJob.MaxAttempts != 3 {
		t.Fatalf("createdJob.MaxAttempts = %d, want 3", createdJob.MaxAttempts)
	}

	if createdJob.Status != model.JobStatusPending {
		t.Fatalf("createdJob.Status = %q, want %q", createdJob.Status, model.JobStatusPending)
	}

	if createdJob.ScheduledAt.IsZero() {
		t.Fatalf("createdJob.ScheduledAt must not be zero")
	}
}

func TestService_CreateJob_RepositoryError(t *testing.T) {
	ctx, jobService := newTestService(t)

	expectedErr := errors.New("database connection failed")

	// Configure the mock repository to return an error.
	mockRepo := &repository.MockJobRepository{
		CreateErr: expectedErr,
	}

	jobService = service.NewService(mockRepo)

	jobPayload := &service.CreateJobInput{
		JobType:  "test",
		Payload:  json.RawMessage(`{"message":"Hello"}`),
		Priority: 1,
	}

	createdJob, err := jobService.CreateJob(ctx, jobPayload)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if createdJob != nil {
		t.Fatalf("expected nil job, got %+v", createdJob)
	}
}

func TestService_GetByID(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload := &service.CreateJobInput{
		JobType:  "test",
		Payload:  json.RawMessage(`{"message":"Hello"}`),
		Priority: 1,
	}

	createdJob, err := jobService.CreateJob(ctx, jobPayload)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	retrievedJob, err := jobService.GetByID(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("error on GetByID(): %+v", err)
	}

	assertJobsEqual(t, retrievedJob, createdJob)
}

func TestService_GetByID_NotFound(t *testing.T) {
	ctx, jobService := newTestService(t)

	nonExistentID := uuid.New()
	retrievedJob, err := jobService.GetByID(ctx, nonExistentID)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !errors.Is(err, repository.ErrJobNotFound) {
		t.Fatalf("expected error %v, got %v", repository.ErrJobNotFound, err)
	}

	if retrievedJob != nil {
		t.Fatalf("expected nil job, got %+v", retrievedJob)
	}

}

func TestService_ClaimJob(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload := &service.CreateJobInput{
		JobType:  "test",
		Payload:  json.RawMessage(`{"message":"Hello"}`),
		Priority: 1,
	}

	createdJob, err := jobService.CreateJob(ctx, jobPayload)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	if createdJob == nil {
		t.Fatal("should return a  job")
	}

	createdJob, err = jobService.GetByID(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("error on GetByID(): %+v", err)
	}

	claimedJob, err := jobService.ClaimJob(ctx)
	if err != nil {
		t.Fatalf("error on ClaimJob(): %+v", err)
	}

	if claimedJob == nil {
		t.Fatal("claimedJob must not be nil")
	}

	assertJobsEqual(t, claimedJob, createdJob)
}

func TestGetAllJobs(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload1 := &service.CreateJobInput{
		JobType:  "test1",
		Payload:  json.RawMessage(`{"message":"Hello1"}`),
		Priority: 1,
	}

	jobPayload2 := &service.CreateJobInput{
		JobType:  "test2",
		Payload:  json.RawMessage(`{"message":"Hello2"}`),
		Priority: 2,
	}

	_, err := jobService.CreateJob(ctx, jobPayload1)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	_, err = jobService.CreateJob(ctx, jobPayload2)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	jobs, err := jobService.GetAll(ctx, repository.GetAllFilters{}, repository.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("error on GetAll(): %+v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

}

func TestGetAllJobs_NotFound(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobType := "nonexistent"
	jobs, err := jobService.GetAll(
		ctx,
		repository.GetAllFilters{JobType: &jobType},
		repository.PaginationParams{Page: 1, Limit: 10},
	)
	if err != nil {
		t.Fatalf("error on GetAll(): %+v", err)
	}

	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestGetAllJobs_JobTypeFilter(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload1 := &service.CreateJobInput{
		JobType:  "test1",
		Payload:  json.RawMessage(`{"message":"Hello1"}`),
		Priority: 1,
	}

	_, err := jobService.CreateJob(ctx, jobPayload1)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	jobType := "test1"
	jobs, err := jobService.GetAll(
		ctx,
		repository.GetAllFilters{JobType: &jobType},
		repository.PaginationParams{Page: 1, Limit: 10},
	)
	if err != nil {
		t.Fatalf("error on GetAll(): %+v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}

func TestGetAllJobs_StatusFilter(t *testing.T) {
	ctx, jobService := newTestService(t)

	jobPayload1 := &service.CreateJobInput{
		JobType:  "test1",
		Payload:  json.RawMessage(`{"message":"Hello1"}`),
		Priority: 1,
	}

	_, err := jobService.CreateJob(ctx, jobPayload1)
	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	status := model.JobStatusPending
	jobs, err := jobService.GetAll(
		ctx,
		repository.GetAllFilters{Status: &status},
		repository.PaginationParams{Page: 1, Limit: 10},
	)

	if err != nil {
		t.Fatalf("error on GetAll(): %+v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

}
