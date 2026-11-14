package server

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	jobDtos "github.com/prabej7/goqueue/internal/job/dtos"
	jobHandler "github.com/prabej7/goqueue/internal/job/handler"
	jobRepository "github.com/prabej7/goqueue/internal/job/repository"
	jobService "github.com/prabej7/goqueue/internal/job/service"
	"github.com/prabej7/goqueue/internal/middleware"
)

type Dependencies struct {
	DB *sql.DB
}

func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GoQueue is running"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		jobRepository := jobRepository.NewJobRepository(deps.DB)
		jobService := jobService.NewService(jobRepository)
		jobHandler := jobHandler.NewHandler(jobService)
		r.Route("/jobs", func(r chi.Router) {
			r.Post("/", middleware.ValidatePayload[jobDtos.QueueJobPayload](jobHandler.QueueJob))
		})
	})

	return r
}
