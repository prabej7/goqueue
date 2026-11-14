package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
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

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

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
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	migrationPath, err := filepath.Abs("../../../migrations")
	if err != nil {
		t.Fatalf("failed to resolve migrations path: %v", err)
	}

	migrationPath = filepath.ToSlash(migrationPath)
	migrationURL := "file://" + migrationPath

	m, err := migrate.New(migrationURL, connStr)
	if err != nil {
		t.Fatalf("failed to initialize migrations: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("failed to run migrations: %v", err)
	}

	sourceErr, databaseErr := m.Close()
	if sourceErr != nil || databaseErr != nil {
		t.Fatalf("failed to close migration resources: source=%v database=%v", sourceErr, databaseErr)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()

		terminateCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := container.Terminate(terminateCtx); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	return db
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

	if err := repo.Create(ctx, &job); err != nil {
		t.Fatalf("failed to create job: %v", err)
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

	_, err := repo.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrJobNotFound) {
			t.Errorf("expected ErrJobNotFound, got %v", err)
			return
		}
	}
}
