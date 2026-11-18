package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prabej7/goqueue/internal/job/dtos"
	"github.com/prabej7/goqueue/internal/job/repository"
	"github.com/prabej7/goqueue/internal/job/service"
	"github.com/prabej7/goqueue/internal/middleware"
	"github.com/prabej7/goqueue/internal/utils/tests"
)

func newHandler() *Handler {
	jobRespository := repository.NewMockRepository()
	jobService := service.NewService(jobRespository)
	return &Handler{
		service: jobService,
	}
}

func TestQueueJob(t *testing.T) {
	requestBody, err := tests.GetJsonPayload(dtos.QueueJobPayload{
		JobType:  "testing",
		Payload:  json.RawMessage(`{"message":"Testing"}`),
		Priority: 1,
	})

	if err != nil {
		t.Errorf("error on GetJsonPayload(): %+v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(requestBody),
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	jobHandler := newHandler()
	handler := middleware.ValidatePayload[dtos.QueueJobPayload](jobHandler.QueueJob)

	handler.ServeHTTP(rec, req)

	tests.AssertHttpResponse(t, rec, http.StatusCreated)
}

func TestQueueHob_InvalidPayload(t *testing.T) {
	requestBody, err := tests.GetJsonPayload(dtos.QueueJobPayload{
		JobType: "testing",
		Payload: json.RawMessage(`{"message":"Testing"}`),
		// Priority: 1,
	})

	if err != nil {
		t.Errorf("error on GetJsonPayload(): %+v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(requestBody),
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	jobHandler := newHandler()
	handler := middleware.ValidatePayload[dtos.QueueJobPayload](jobHandler.QueueJob)

	handler.ServeHTTP(rec, req)

	tests.AssertHttpResponse(t, rec, http.StatusBadRequest)
}

func TestGetJob(t *testing.T) {
	jobHandler := newHandler()

	createdJob, err := jobHandler.service.CreateJob(
		t.Context(),
		&service.CreateJobInput{
			JobType:  "test",
			Payload:  json.RawMessage(`{"message":"hello"}`),
			Priority: 1,
		},
	)

	if err != nil {
		t.Fatalf("error on CreateJob(): %+v", err)
	}

	router := chi.NewRouter()

	router.Get("/api/v1/jobs/{id}", jobHandler.GetJobByID)

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/jobs/%s", createdJob.ID),
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	tests.AssertHttpResponse(t, rec, http.StatusOK)
}

func TestGetAll(t *testing.T) {

	req, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/jobs?page=1&limit=1",
		nil,
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	jobHandler := newHandler()
	handler := middleware.ValidateQuery[dtos.GetAllQuery](jobHandler.GetAll)

	handler.ServeHTTP(rec, req)

	tests.AssertHttpResponse(t, rec, http.StatusOK)
}
