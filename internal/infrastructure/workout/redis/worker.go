package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// volumeBlockTimeout is how long a BRPop waits for a new job before
// re-checking for shutdown.
const volumeBlockTimeout = 5 * time.Second

// brpopErrorBackoff pauses the loop after a failed BRPop so a Redis outage
// does not busy-spin.
const brpopErrorBackoff = time.Second

// VolumeCalculator recomputes and stores the progress volume of a workout.
type VolumeCalculator interface {
	Recalculate(ctx context.Context, workoutID, userID uuid.UUID) error
}

// Worker consumes volume recalculation jobs from Redis and runs them
// through the calculator.
type Worker struct {
	client     *redis.Client
	calculator VolumeCalculator
	logger     *slog.Logger
}

// NewWorker creates a worker that consumes the volume queue and hands each
// job to the calculator.
func NewWorker(client *redis.Client, calculator VolumeCalculator, logger *slog.Logger) *Worker {
	return &Worker{client: client, calculator: calculator, logger: logger}
}

// Run blocks processing jobs until ctx is cancelled. A failed job is logged
// and the worker moves on to the next one.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("starting volume recalculation worker")
	for {
		if ctx.Err() != nil {
			return
		}
		result, err := w.client.BRPop(ctx, volumeBlockTimeout, volumeCalcQueue).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			w.logger.Error("brpop failed", "error", err)
			time.Sleep(brpopErrorBackoff)
			continue
		}
		if err := w.handleJob(ctx, result[1]); err != nil {
			w.logger.Error("volume recalculation failed", "error", err, "payload", result[1])
		}
	}
}

// handleJob decodes a job payload and runs the volume recalculation for it.
func (w *Worker) handleJob(ctx context.Context, payload string) error {
	var job VolumeCalcJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return err
	}
	return w.calculator.Recalculate(ctx, job.WorkoutID, job.UserID)
}
