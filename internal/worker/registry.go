package worker

import (
	"context"
	"encoding/json"
)

type JobHandler func(ctx context.Context, payload json.RawMessage) error

type Registry map[string]JobHandler

func (r Registry) Register(jobType string, handler JobHandler) {
	r[jobType] = handler
}

func (r Registry) Get(jobType string) (JobHandler, bool) {
	handler, ok := r[jobType]
	return handler, ok
}
