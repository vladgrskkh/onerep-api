package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

// volumeCalcQueue is the list holding pending volume recalculation jobs.
const volumeCalcQueue = "gym:volume:recalculate"

// volumeBlockTimeout is how long a pop waits for a new job before reporting
// an empty queue, letting the worker re-check for shutdown.
const volumeBlockTimeout = 5 * time.Second

// ErrNoVolumeJob reports an empty queue at the end of a pop wait.
var ErrNoVolumeJob = errors.New("no volume job available")

// VolumeCalcJob identifies a finished workout whose volume needs
// recalculation.
type VolumeCalcJob struct {
	WorkoutID uuid.UUID `json:"workout_id"`
	UserID    uuid.UUID `json:"user_id"`
}

// Queue enqueues background jobs into Redis and pops them for processing.
type Queue struct {
	client *redis.Client
}

// NewQueue creates a queue backed by the given Redis client.
func NewQueue(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// EnqueueVolumeCalc pushes a volume recalculation job for the workout onto
// the queue.
func (q *Queue) EnqueueVolumeCalc(ctx context.Context, workout domainworkout.Workout) error {
	job := VolumeCalcJob{WorkoutID: workout.ID, UserID: workout.UserID}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal volume calc job: %w", err)
	}
	return q.client.LPush(ctx, volumeCalcQueue, data).Err()
}

// PopVolumeCalcJob blocks until a volume recalculation job arrives, returning
// ErrNoVolumeJob when the queue stays empty for the block timeout.
func (q *Queue) PopVolumeCalcJob(ctx context.Context) (VolumeCalcJob, error) {
	result, err := q.client.BRPop(ctx, volumeBlockTimeout, volumeCalcQueue).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return VolumeCalcJob{}, ErrNoVolumeJob
		}
		return VolumeCalcJob{}, err
	}

	var job VolumeCalcJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return VolumeCalcJob{}, fmt.Errorf("unmarshal volume calc job: %w", err)
	}
	return job, nil
}
