package handler

import (
	"net/http"

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
	payload := middleware.GetPayload[QueueJobPayload](r)
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
