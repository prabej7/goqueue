package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("goqueue_test"),
		tcpostgres.WithUsername("goqueue"),
		tcpostgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	migrationPath, err := filepath.Abs("../../../migrations")
	if err != nil {
		log.Fatalf("failed to resolve migrations path: %v", err)
	}
	migrationURL := "file://" + filepath.ToSlash(migrationPath)

	m2, err := migrate.New(migrationURL, connStr)
	if err != nil {
		log.Fatalf("failed to initialize migrations: %v", err)
	}
	if err := m2.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to run migrations: %v", err)
	}
	if sErr, dErr := m2.Close(); sErr != nil || dErr != nil {
		log.Fatalf("failed to close migration resources: source=%v database=%v", sErr, dErr)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	testDB = db

	code := m.Run()

	_ = db.Close()
	terminateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := container.Terminate(terminateCtx); err != nil {
		log.Printf("failed to terminate postgres container: %v", err)
	}

	os.Exit(code)
}

// setupTestDB now just returns the shared DB and resets rows for isolation.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if _, err := testDB.Exec(`TRUNCATE TABLE jobs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("failed to truncate jobs: %v", err)
	}
	return testDB
}

func TestJobRepository_Create(t *testing.T) {
	db := setupTestDB(t)

	repo := NewJobRepository(db)
	ctx := context.Background()

	id := uuid.New()
	job := model.Job{
		ID:          id,
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
	}

	createdJob, err := repo.Create(ctx, &job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if createdJob == nil {
		t.Fatal("createdJob must not be nil")
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if got.ID != job.ID {
		t.Errorf("ID mismatch: got %v, expected %v", got.ID, job.ID)
	}

	if got.Type != job.Type {
		t.Errorf("Type mismatch: got %q, expected %q", got.Type, job.Type)
	}

	var expectedPayload any
	var actualPayload any

	if err := json.Unmarshal(job.Payload, &expectedPayload); err != nil {
		t.Fatalf("failed to decode expected payload: %v", err)
	}

	if err := json.Unmarshal(got.Payload, &actualPayload); err != nil {
		t.Fatalf("failed to decode actual payload: %v", err)
	}

	if !reflect.DeepEqual(actualPayload, expectedPayload) {
		t.Errorf("Payload mismatch: got %s, expected %s", got.Payload, job.Payload)
	}

	if got.Status != job.Status {
		t.Errorf("Status mismatch: got %v, expected %v", got.Status, job.Status)
	}

	if got.Priority != job.Priority {
		t.Errorf("Priority mismatch: got %v, expected %v", got.Priority, job.Priority)
	}

	if got.MaxAttempts != job.MaxAttempts {
		t.Errorf("MaxAttempts mismatch: got %v, expected %v", got.MaxAttempts, job.MaxAttempts)
	}

	if got.Attempts != job.Attempts {
		t.Errorf("Attempts mismatch: got %v, expected %v", got.Attempts, job.Attempts)
	}

	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestJobRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)

	ctx := context.Background()

	id := uuid.New()

	job, err := repo.GetByID(ctx, id)
	if !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}

	if job != nil {
		t.Fatalf("job should be nil but got: %+v", job)
	}
}

func TestJobRepository_Create_DuplicateID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	id := uuid.New()
	job := model.Job{
		ID:          id,
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
	}

	createdJob, err := repo.Create(ctx, &job)
	if err != nil {
		t.Fatalf("error in Create() %+v", err)
	}

	if createdJob == nil {
		t.Fatalf("createdJob must not be nil")
	}

	job2 := model.Job{
		ID:          id,
		Type:        "test1",
		Payload:     json.RawMessage(`{"message":"hello1"}`),
		Status:      model.JobStatusPending,
		Priority:    2,
		Attempts:    0,
		MaxAttempts: 5,
		ScheduledAt: time.Now(),
	}

	createdJob2, err := repo.Create(ctx, &job2)
	if err == nil {
		t.Fatalf("must throw error on duplicate uuid")
	}

	if createdJob2 != nil {
		t.Fatalf("createdJob2 must be nill as that would duplicate")
	}
}

func TestJobRepository_ClaimJob(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	id := uuid.New()
	job := model.Job{
		ID:          id,
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdJob, err := repo.Create(ctx, &job)

	if err != nil {
		t.Fatalf("error in Create() %+v", err)
	}

	if createdJob == nil {
		t.Fatalf("createdJob must not be nil")
	}

	createdJob, err = repo.GetByID(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("error on GetByID(): %+v", err)
	}

	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}
	claimedJob, err := repo.ClaimNextJob(ctx, claimJobPayload)

	if err != nil {
		t.Fatalf("should return a job: %+v", err)
	}

	lockedBy := workerId.String()
	assertJobsEqual(t, claimedJob, &model.Job{
		ID:             job.ID,
		Type:           job.Type,
		Payload:        job.Payload,
		Status:         model.JobStatusProcessing,
		Priority:       job.Priority,
		Attempts:       job.Attempts,
		MaxAttempts:    job.MaxAttempts,
		ScheduledAt:    job.ScheduledAt,
		LockedBy:       &lockedBy,
		LockedAt:       claimedJob.LockedAt,
		LeaseExpiresAt: claimedJob.LeaseExpiresAt,
		ClaimToken:     claimedJob.ClaimToken,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	})
}

func TestJobRepository_ClaimJob_NoJobsAvailable(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	_, err := repo.ClaimNextJob(ctx, claimJobPayload)

	if !errors.Is(err, ErrNoJobsAvailable) {
		t.Error("should return ErrNoJobsAvailable")
	}

}

func TestJobRepository_ConcurrentClaims(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	const workers = 2

	start := make(chan struct{})
	results := make(chan error, workers)

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			payload := ClaimNextJobPayload{
				WorkerID:        uuid.New(),
				LeaseExpiration: time.Now().Add(5 * time.Minute),
				ClaimToken:      uuid.New(),
			}

			claimedJob, err := repo.ClaimNextJob(ctx, payload)

			if err != nil {
				results <- err
				return
			}

			if claimedJob.ID != job.ID {
				results <- fmt.Errorf("unexpected job claimed: %s", claimedJob.ID)
				return
			}

			results <- nil
		}()
	}

	// Release both workers simultaneously.
	close(start)

	wg.Wait()
	close(results)

	successfulClaims := 0
	unavailableClaims := 0

	for err := range results {
		if err == nil {
			successfulClaims++
			continue
		}

		if errors.Is(err, ErrNoJobsAvailable) {
			unavailableClaims++
			continue
		}

		t.Errorf("unexpected claim error: %v", err)
	}

	if successfulClaims != 1 {
		t.Errorf("successful claims = %d, want 1", successfulClaims)
	}

	if unavailableClaims != 1 {
		t.Errorf("unavailable claims = %d, want 1", unavailableClaims)
	}
}

func TestJobRepository_CompleteJob(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	//claimJob
	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	_, err = repo.ClaimNextJob(ctx, claimJobPayload)
	if err != nil {
		t.Errorf("error on ClaimJob(): %+v", err)
	}

	if err := repo.CompleteJob(ctx, job.ID, claimToken.String()); err != nil {
		t.Errorf("error on CompleteJob(): %+v", err)
	}

	createdJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to GetByID job: %v", err)
	}

	t.Log("claimToken:", createdJob.ClaimToken)

	if createdJob.Status != model.JobStatusCompleted {
		t.Errorf("expected 'completed', got: %s", createdJob.Status)
	}
}

func TestJobRepository_ClaimJob_StaleClaimToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	_, err = repo.ClaimNextJob(ctx, claimJobPayload)
	if err != nil {
		t.Errorf("error on ClaimJob(): %+v", err)
	}

	claimToken = uuid.New()
	if err := repo.CompleteJob(ctx, job.ID, claimToken.String()); !errors.Is(err, ErrJobClaimLost) {
		t.Errorf("expected ErrJobClaimLost, got: %+v", err)
	}
}

func TestJobRepository_FailJob(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	//claimJob
	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	claimedJob, err := repo.ClaimNextJob(ctx, claimJobPayload)
	if err != nil {
		t.Errorf("error on ClaimJob(): %+v", err)
	}

	if err := repo.FailJob(ctx, job.ID, *claimedJob.ClaimToken); err != nil {
		t.Errorf("error on CompleteJob(): %+v", err)
	}

	createdJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to GetByID job: %v", err)
	}

	if createdJob.Status != model.JobStatusFailed {
		t.Errorf("expected 'completed', got: %s", createdJob.Status)
	}
}

func assertJobsEqual(t *testing.T, got, want *model.Job) {
	t.Helper()

	if got == nil {
		t.Fatal("got job must not be nil")
	}
	if want == nil {
		t.Fatal("want job must not be nil")
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

	// *string
	if (got.LockedBy == nil) != (want.LockedBy == nil) {
		t.Fatalf("job.LockedBy nil-ness mismatch: got %v, want %v", got.LockedBy, want.LockedBy)
	}
	if want.LockedBy != nil && *got.LockedBy != *want.LockedBy {
		t.Fatalf("job.LockedBy = %q, want %q", *got.LockedBy, *want.LockedBy)
	}

	// *time.Time
	assertTimePtrEqual(t, "LockedAt", got.LockedAt, want.LockedAt)
	assertTimePtrEqual(t, "LeaseExpiresAt", got.LeaseExpiresAt, want.LeaseExpiresAt)
	assertTimePtrEqual(t, "StartedAt", got.StartedAt, want.StartedAt)
	assertTimePtrEqual(t, "CompletedAt", got.CompletedAt, want.CompletedAt)

	// ClaimToken: adjust the deref to match its concrete type.
	// If ClaimToken is *uuid.UUID:
	if (got.ClaimToken == nil) != (want.ClaimToken == nil) {
		t.Fatalf("job.ClaimToken nil-ness mismatch: got %v, want %v", got.ClaimToken, want.ClaimToken)
	}
	if want.ClaimToken != nil && *got.ClaimToken != *want.ClaimToken {
		t.Fatalf("job.ClaimToken = %v, want %v", *got.ClaimToken, *want.ClaimToken)
	}

	if !reflect.DeepEqual(got.Result, want.Result) {
		t.Fatalf("job.Result = %s, want %s", got.Result, want.Result)
	}

	if !got.ScheduledAt.Truncate(time.Microsecond).Equal(want.ScheduledAt.Truncate(time.Microsecond)) {
		t.Fatalf("job.ScheduledAt = %v, want %v", got.ScheduledAt, want.ScheduledAt)
	}
	if !got.CreatedAt.Truncate(time.Microsecond).Equal(want.CreatedAt.Truncate(time.Microsecond)) {
		t.Fatalf("job.CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Truncate(time.Microsecond).Equal(want.UpdatedAt.Truncate(time.Microsecond)) {
		t.Fatalf("job.UpdatedAt = %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertTimePtrEqual(t *testing.T, field string, got, want *time.Time) {
	t.Helper()

	if (got == nil) != (want == nil) {
		t.Fatalf("job.%s nil-ness mismatch: got %v, want %v", field, got, want)
	}
	if want == nil {
		return // both nil
	}
	if !got.Equal(*want) {
		t.Fatalf("job.%s = %v, want %v", field, *got, *want)
	}
}

func TestUpdateLastError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	claimedJob, err := repo.ClaimNextJob(ctx, claimJobPayload)
	if err != nil {
		t.Errorf("error on ClaimJob(): %+v", err)
	}

	lastError := errors.New("test error")

	if err := repo.UpdateLastError(
		ctx,
		claimedJob.ID,
		claimToken.String(),
		lastError.Error(),
	); err != nil {
		t.Errorf("should updated last error but: %+v", err)
	}

	updatedJob, err := repo.GetByID(ctx, claimedJob.ID)
	if err != nil {
		t.Errorf("error on GetByID(): %+v", err)
	}

	lastErrorString := lastError.Error()
	if *updatedJob.LastError != lastErrorString {
		t.Errorf("expected %v, got %v", lastErrorString, *updatedJob.LastError)
	}
}

func TestRenewLease(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	ctx := context.Background()

	// Create one pending job.
	job := &model.Job{
		ID:          uuid.New(),
		Type:        "test",
		Payload:     json.RawMessage(`{"message":"hello"}`),
		Status:      model.JobStatusPending,
		Priority:    1,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	_, err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	workerId := uuid.New()
	claimToken := uuid.New()
	claimJobPayload := ClaimNextJobPayload{
		WorkerID:        workerId,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
		ClaimToken:      claimToken,
	}

	claimedJob, err := repo.ClaimNextJob(ctx, claimJobPayload)
	if err != nil {
		t.Errorf("error on ClaimJob(): %+v", err)
	}

	newLeaseExpiration := time.Now().
		Add(time.Minute * 5).
		Truncate(time.Microsecond)
	if err := repo.RenewLease(
		ctx,
		claimedJob.ID,
		claimToken.String(),
		newLeaseExpiration,
	); err != nil {
		t.Fatalf("expected no error on RenewLease(), got:%+v", err)
	}

	updatedJob, err := repo.GetByID(ctx, claimedJob.ID)
	if err != nil {
		t.Fatalf("expected no error on GetByID(): %+v", err)
	}

	assertTimePtrEqual(
		t,
		"lease expiration",
		updatedJob.LeaseExpiresAt,
		&newLeaseExpiration,
	)
}
