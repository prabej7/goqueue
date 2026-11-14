package handler

import (
	"net/http"

	"github.com/prabej7/goqueue/internal/job/dtos"
	"github.com/prabej7/goqueue/internal/job/repository"
	"github.com/prabej7/goqueue/internal/job/service"
	"github.com/prabej7/goqueue/internal/middleware"
	apperrors "github.com/prabej7/goqueue/internal/utils/errors"
	"github.com/prabej7/goqueue/internal/utils/response"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) QueueJob(w http.ResponseWriter, r *http.Request) {
	payload := middleware.GetPayload[dtos.QueueJobPayload](r)
	ctx := r.Context()

	createdJob, err := h.service.CreateJob(ctx, &service.CreateJobInput{
		JobType:  payload.JobType,
		Payload:  payload.Payload,
		Priority: payload.Priority,
	})

	if err != nil {
		apperrors.InternalServerError(w, err)
		return
	}

	response.SendResponse(w, http.StatusCreated, &response.Response{
		Message: "job queued successfully",
		Data:    createdJob,
	})
}

func (h *Handler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetParams[dtos.GetJobParams](r).ID
	ctx := r.Context()

	job, err := h.service.GetByID(ctx, id)
	if err != nil {
		switch err {
		case repository.ErrJobNotFound:
			apperrors.NotFoundError(w, "job not found")
			return
		default:
			apperrors.InternalServerError(w, err)
			return
		}
	}

	response.SendResponse(w, http.StatusOK, &response.Response{
		Message: "job retrived successfully",
		Data:    job,
	})
}
