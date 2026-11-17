package handler

import (
	"context"
	"encoding/json"
	"log"
)

func SendEmail(ctx context.Context, payload json.RawMessage) error {
	log.Printf("processing email job: %s", payload)

	/// TODO: implement email sending

	return nil
}
