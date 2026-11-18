package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prabej7/goqueue/internal/config"
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
		failErr error,
		retryAt time.Time,
	) (*model.Job, error)
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
	RenewLease(
		ctx context.Context,
		jobID uuid.UUID,
		claimToken string,
		leaseExpiration time.Time,
	) error
}

type Worker struct {
	id       uuid.UUID
	repo     JobRepository
	registry *Registry
	cfg      *config.Config
}

var (
	ErrNoHandlerFound = errors.New("no handler registered for the job type")
)

func NewWorker(
	id uuid.UUID,
	repo JobRepository,
	registry *Registry,
	cfg *config.Config,
) *Worker {
	return &Worker{
		id:       id,
		repo:     repo,
		registry: registry,
		cfg:      cfg,
	}
}

func (w *Worker) processNextJob(ctx context.Context) error {
	claimToken := uuid.New()
	nextJob, err := w.repo.ClaimNextJob(ctx, repository.ClaimNextJobPayload{
		ClaimToken:      claimToken,
		WorkerID:        w.id,
		LeaseExpiration: time.Now().Add(w.cfg.LeaseDuration),
	})

	if err != nil {
		return err
	}

	if nextJob == nil {
		return errors.New("repository returned nil job without error")
	}

	jobCtx, cancel := context.WithTimeout(
		/*
			Cancellation of application context shouldn't cancel
			thhe processing worker job
		*/
		context.WithoutCancel(ctx),
		w.cfg.JobTimeout,
	)
	defer cancel()

	handlerCtx, cancelHandler := context.WithCancel(jobCtx)
	defer cancelHandler()

	handler, found := w.registry.Get(nextJob.Type)
	if !found {
		backoff := computeBackoff(
			nextJob.Attempts,
			time.Second*5,
			time.Minute*5,
		)
		_, err := w.repo.FailJob(
			ctx,
			nextJob.ID,
			claimToken.String(),
			ErrNoHandlerFound,
			time.Now().Add(backoff),
		)

		if err != nil {
			return err
		}
		return nil
	}

	heartbeatCtx, stopHeatbeat := context.WithCancel(jobCtx)
	defer stopHeatbeat()

	heartbeatErr := make(chan error, 1)

	go func() {
		heartbeatErr <- w.heartbeat(
			heartbeatCtx,
			nextJob.ID,
			claimToken.String(),
			cancelHandler,
		)
	}()

	err = handler(handlerCtx, nextJob.Payload)
	stopHeatbeat()

	hbErr := <-heartbeatErr

	if hbErr != nil {
		return hbErr
	}

	if err != nil {
		backoff := computeBackoff(
			nextJob.Attempts,
			time.Second*5,
			time.Minute*5,
		)
		if _, err := w.repo.FailJob(
			ctx,
			nextJob.ID,
			claimToken.String(),
			err,
			time.Now().Add(backoff),
		); err != nil {
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

func (w *Worker) heartbeat(
	ctx context.Context,
	jobID uuid.UUID,
	claimToken string,
	cancelJobContext context.CancelFunc,
) error {
	ticker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(
				ctx,
				w.cfg.RenewTimeout,
			)

			expiration := time.Now().Add(w.cfg.LeaseDuration)

			err := w.repo.RenewLease(
				renewCtx,
				jobID,
				claimToken,
				expiration,
			)

			cancel()

			if err != nil {
				cancelJobContext()
				return fmt.Errorf("heartbeat failed: %w", err)
			}

		}
	}
}

func computeBackoff(
	attempts int,
	baseDelay time.Duration,
	maxDelay time.Duration,
) time.Duration {
	if attempts <= 0 || baseDelay <= 0 {
		return 0
	}

	delay := baseDelay

	for i := 1; i < attempts; i++ {
		if delay >= maxDelay/2 {
			return maxDelay
		}

		delay *= 2
	}

	if delay > maxDelay {
		return maxDelay
	}

	return delay
}
