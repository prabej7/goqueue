package store

import (
	"database/sql"

	"github.com/prabej7/goqueue/internal/job/repository"
	"github.com/prabej7/goqueue/internal/job/service"
)

type Store struct {
	JobService    *service.Service
	JobRepository *repository.Repository
}

func new(
	jobRepository repository.Repository) *Store {
	return &Store{
		JobService:    service.NewService(jobRepository),
		JobRepository: &jobRepository,
	}
}

func NewPostgresStore(db *sql.DB) *Store {
	return new(
		repository.NewJobRepository(db),
	)
}

func NewMockStore() *Store {
	return new(
		repository.NewMockRepository(),
	)
}
