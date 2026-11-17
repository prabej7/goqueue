package bootstrap

import (
	"github.com/prabej7/goqueue/internal/job/handler"
	"github.com/prabej7/goqueue/internal/worker"
)

func NewWorkerRegistry() *worker.Registry {
	registry := &worker.Registry{}

	registry.Register("email.send", handler.SendEmail)

	return registry
}
