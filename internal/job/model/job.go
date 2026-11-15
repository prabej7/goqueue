package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusDead       JobStatus = "dead"
	JobStatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID             uuid.UUID       `json:"id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Status         JobStatus       `json:"status"`
	Priority       int             `json:"priority"`
	Attempts       int             `json:"atempts"`
	MaxAttempts    int             `json:"max_attempts"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	LockedBy       string          `json:"locked_by,omitempty"`
	LockedAt       time.Time       `json:"locked_at,omitempty"`
	LeaseExpiresAt time.Time       `json:"lease_expires_at,omitempty"`
	ClaimToken     string          `json:"claim_token,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	StartedAt      time.Time       `json:"started_at,omitempty"`
	CompletedAt    time.Time       `json:"completed_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
