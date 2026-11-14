package dtos

import "encoding/json"

type QueueJobPayload struct {
	JobType  string          `json:"job_type" validate:"required"`
	Payload  json.RawMessage `json:"payload" validate:"required"`
	Priority int             `json:"priority" validate:"required"`
}

type GetJobParams struct {
	ID string `json:"id"`
}
