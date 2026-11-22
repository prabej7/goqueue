package handler

import (
	"context"
	"encoding/json"
	"time"
)

func SendEmail(ctx context.Context, payload json.RawMessage) error {

	// Simulate email sending delay
	time.Sleep(5 * time.Second)

	return nil
}
