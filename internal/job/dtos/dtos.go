package dtos

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
)

type QueueJobPayload struct {
	JobType  string          `json:"job_type" validate:"required"`
	Payload  json.RawMessage `json:"payload" validate:"required"`
	Priority int             `json:"priority" validate:"required"`
}

type GetJobParams struct {
	ID uuid.UUID `json:"id"`
}

type GetAllQuery struct {
	Page    int              `schema:"page" validate:"min=1"`
	Limit   int              `schema:"limit" validate:"min=1,max=100"`
	Status  *model.JobStatus `schema:"status"`
	JobType *string          `schema:"job_type"`
}
