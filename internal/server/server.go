package server

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	jobDtos "github.com/prabej7/goqueue/internal/job/dtos"
	jobHandler "github.com/prabej7/goqueue/internal/job/handler"
	"github.com/prabej7/goqueue/internal/middleware"
	"github.com/prabej7/goqueue/internal/store"
)

type Dependencies struct {
	DB    *sql.DB
	Store *store.Store
}

func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GoQueue is running"))
	})

	r.Route("/api/v1", func(r chi.Router) {

		jobHandler := jobHandler.NewHandler(deps.Store.JobService)

		r.Route("/jobs", func(r chi.Router) {
			r.Post("/", middleware.ValidatePayload[jobDtos.QueueJobPayload](jobHandler.QueueJob))

			r.Get("/{id}", middleware.ValidateUUIDParam("id")(jobHandler.GetJobByID))

			r.Get("/",
				middleware.ValidateQuery[jobDtos.GetAllQuery](jobHandler.GetAll),
			)
		})
	})

	return r
}
