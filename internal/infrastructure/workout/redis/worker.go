package redis

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// popErrorBackoff pauses the loop after a failed pop so a Redis outage does
// not busy-spin.
const popErrorBackoff = time.Second

// VolumeJobSource pops the next pending volume recalculation job, blocking
// until a job arrives or the wait times out.
type VolumeJobSource interface {
	PopVolumeCalcJob(ctx context.Context) (VolumeCalcJob, error)
}

// VolumeCalculator recomputes and stores the progress volume of a workout.
type VolumeCalculator interface {
	Recalculate(ctx context.Context, workoutID, userID uuid.UUID) error
}

// Worker consumes volume recalculation jobs from the queue and runs them
// through the calculator.
type Worker struct {
	source     VolumeJobSource
	calculator VolumeCalculator
	logger     *slog.Logger
}

// NewWorker creates a worker that consumes the volume job source and hands
// each job to the calculator.
func NewWorker(source VolumeJobSource, calculator VolumeCalculator, logger *slog.Logger) *Worker {
	return &Worker{source: source, calculator: calculator, logger: logger}
}

// Run blocks processing jobs until ctx is cancelled. A failed job is logged
// and the worker moves on to the next one.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("starting volume recalculation worker")
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.source.PopVolumeCalcJob(ctx)
		if err != nil {
			if errors.Is(err, ErrNoVolumeJob) {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			w.logger.Error("pop volume calc job failed", "error", err)
			time.Sleep(popErrorBackoff)
			continue
		}
		if err := w.calculator.Recalculate(ctx, job.WorkoutID, job.UserID); err != nil {
			w.logger.Error(
				"volume recalculation failed",
				"error", err,
				"workout_id", job.WorkoutID,
				"user_id", job.UserID,
			)
		}
	}
}
