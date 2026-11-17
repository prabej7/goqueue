package worker

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/job/model"
	"github.com/prabej7/goqueue/internal/job/repository"
)

type JobRepository interface {
	CompleteJob(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
	) error
	FailJob(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
	) error
	ClaimNextJob(
		ctx context.Context,
		payload repository.ClaimNextJobPayload,
	) (*model.Job, error)
	UpdateLastError(
		ctx context.Context,
		ID uuid.UUID,
		claimToken string,
		err string,
	) error
}

type Worker struct {
	id       uuid.UUID
	repo     JobRepository
	registry *Registry
}

var (
	ErrNoHandlerFound = errors.New("no handler registered for the job type")
)

func NewWorker(id uuid.UUID, repo JobRepository, registry *Registry) *Worker {
	return &Worker{
		id:       id,
		repo:     repo,
		registry: registry,
	}
}

func (w *Worker) processNextJob(ctx context.Context) error {
	claimToken := uuid.New()
	nextJob, err := w.repo.ClaimNextJob(ctx, repository.ClaimNextJobPayload{
		ClaimToken:      claimToken,
		WorkerID:        w.id,
		LeaseExpiration: time.Now().Add(time.Minute * 5),
	})

	if err != nil {
		return err
	}

	if nextJob == nil {
		return errors.New("repository returned nil job without error")
	}

	jobCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		2*time.Minute,
	)
	
	defer cancel()

	handler, found := w.registry.Get(nextJob.Type)
	if !found {
		if err := w.repo.UpdateLastError(
			ctx,
			nextJob.ID,
			claimToken.String(),
			ErrNoHandlerFound.Error(),
		); err != nil {
			return err
		}

		if err := w.repo.FailJob(ctx, nextJob.ID, claimToken.String()); err != nil {
			return err
		}

		return nil
	}

	err = handler(jobCtx, nextJob.Payload)
	if err != nil {
		if err := w.repo.UpdateLastError(
			ctx,
			nextJob.ID,
			claimToken.String(),
			err.Error(),
		); err != nil {
			return err
		}

		if err := w.repo.FailJob(ctx, nextJob.ID, claimToken.String()); err != nil {
			return err
		}

		return nil
	}

	if err := w.repo.CompleteJob(
		ctx,
		nextJob.ID,
		claimToken.String(),
	); err != nil {
		return err
	}

	return nil
}

func (w *Worker) Start(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		err := w.processNextJob(ctx)

		if err != nil {
			if errors.Is(err, repository.ErrNoJobsAvailable) {
				timer := time.NewTimer(time.Second)

				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case <-timer.C:
				}

				continue
			}

			return err
		}
	}
}
