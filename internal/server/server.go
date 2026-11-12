package server

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	return r
}
