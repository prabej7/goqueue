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
	"github.com/prabej7/goqueue/internal/job/model"
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

	repo := repository.NewMockRepository()
	s := service.NewService(repo)

	jbs := []*service.CreateJobInput{
		{
			JobType:  "test",
			Payload:  json.RawMessage(`{"message":"Hello"}`),
			Priority: 1,
		},

		{
			JobType:  "test",
			Payload:  json.RawMessage(`{"message":"World"}`),
			Priority: 2,
		},
	}

	for _, job := range jbs {
		_, err := s.CreateJob(req.Context(), job)
		if err != nil {
			t.Fatalf("error on CreateJob(): %+v", err)
		}
	}

	jobHandler := NewHandler(s)
	handler := middleware.ValidateQuery[dtos.GetAllQuery](jobHandler.GetAll)

	handler.ServeHTTP(rec, req)

	type response struct {
		Message string      `json:"message"`
		Data    []model.Job `json:"data"`
	}

	var resp response

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error on Unmarshal(): %+v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(resp.Data))
	}

	tests.AssertHttpResponse(t, rec, http.StatusOK)
}

func TestGetAll_InvalidQuery(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/jobs?page=0&limit=1",
		nil,
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	jobHandler := newHandler()
	handler := middleware.ValidateQuery[dtos.GetAllQuery](jobHandler.GetAll)

	handler.ServeHTTP(rec, req)

	tests.AssertHttpResponse(t, rec, http.StatusBadRequest)
}

func TestGetAll_JobTypeFilter(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/jobs?page=1&limit=10&job_type=test1",
		nil,
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	repo := repository.NewMockRepository()
	s := service.NewService(repo)

	jbs := []*service.CreateJobInput{
		{
			JobType:  "test1",
			Payload:  json.RawMessage(`{"message":"Hello"}`),
			Priority: 1,
		},

		{
			JobType:  "test2",
			Payload:  json.RawMessage(`{"message":"World"}`),
			Priority: 2,
		},
	}

	for _, job := range jbs {
		_, err := s.CreateJob(req.Context(), job)
		if err != nil {
			t.Fatalf("error on CreateJob(): %+v", err)
		}
	}

	jobHandler := NewHandler(s)
	handler := middleware.ValidateQuery[dtos.GetAllQuery](jobHandler.GetAll)

	handler.ServeHTTP(rec, req)

	type response struct {
		Message string      `json:"message"`
		Data    []model.Job `json:"data"`
	}

	var resp response

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error on Unmarshal(): %+v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 job, got %d", len(resp.Data))
	}

	tests.AssertHttpResponse(t, rec, http.StatusOK)
}

func TestGetAll_StatusFilter(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/jobs?page=1&limit=10&status=pending",
		nil,
	)

	if err != nil {
		t.Errorf("error on NewRequest(): %+v", err)
	}

	rec := httptest.NewRecorder()

	repo := repository.NewMockRepository()
	s := service.NewService(repo)

	jbs := []*service.CreateJobInput{
		{
			JobType:  "test1",
			Payload:  json.RawMessage(`{"message":"Hello"}`),
			Priority: 1,
		},

		{
			JobType:  "test2",
			Payload:  json.RawMessage(`{"message":"World"}`),
			Priority: 2,
		},
	}

	for _, job := range jbs {
		_, err := s.CreateJob(req.Context(), job)
		if err != nil {
			t.Fatalf("error on CreateJob(): %+v", err)
		}
	}

	jobHandler := NewHandler(s)
	handler := middleware.ValidateQuery[dtos.GetAllQuery](jobHandler.GetAll)

	handler.ServeHTTP(rec, req)

	type response struct {
		Message string      `json:"message"`
		Data    []model.Job `json:"data"`
	}

	var resp response

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error on Unmarshal(): %+v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(resp.Data))
	}

	tests.AssertHttpResponse(t, rec, http.StatusOK)
}