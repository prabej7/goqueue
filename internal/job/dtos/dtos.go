package dtos

import (
	"encoding/json"

	"github.com/google/uuid"
)

type QueueJobPayload struct {
	JobType  string          `json:"job_type" validate:"required"`
	Payload  json.RawMessage `json:"payload" validate:"required"`
	Priority int             `json:"priority" validate:"required"`
}

type GetJobParams struct {
	ID uuid.UUID `json:"id"`
}
