package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

// volumeCalcQueue is the list holding pending volume recalculation jobs.
const volumeCalcQueue = "gym:volume:recalculate"

// VolumeCalcJob identifies a finished workout whose volume needs
// recalculation.
type VolumeCalcJob struct {
	WorkoutID uuid.UUID `json:"workout_id"`
	UserID    uuid.UUID `json:"user_id"`
}

// Queue enqueues background jobs into Redis.
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
