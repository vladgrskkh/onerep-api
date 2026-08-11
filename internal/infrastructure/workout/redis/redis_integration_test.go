//go:build integration

package redis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	workoutredis "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/redis"
)

// integrationBlockTimeout bounds the BRPop in the integration test.
const integrationBlockTimeout = 2 * time.Second

type QueueIntegrationSuite struct {
	suite.Suite

	client *redis.Client
	queue  *workoutredis.Queue
	ctx    context.Context
}

func (s *QueueIntegrationSuite) SetupTest() {
	opts, err := redis.ParseURL("redis://localhost:6379/0")
	s.Require().NoError(err)
	s.client = redis.NewClient(opts)
	s.queue = workoutredis.NewQueue(s.client)
	s.ctx = context.Background()
	s.Require().NoError(s.client.Del(s.ctx, volumeCalcQueueKey).Err())
}

func (s *QueueIntegrationSuite) TearDownTest() {
	if s.client != nil {
		_ = s.client.Del(s.ctx, volumeCalcQueueKey).Err()
		s.Require().NoError(s.client.Close())
	}
}

func (s *QueueIntegrationSuite) TestEnqueueVolumeCalc_RoundTrip() {
	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

	s.Require().NoError(s.queue.EnqueueVolumeCalc(s.ctx, workout))

	result, err := s.client.BRPop(s.ctx, integrationBlockTimeout, volumeCalcQueueKey).Result()
	s.Require().NoError(err)
	s.Equal(volumeCalcQueueKey, result[0])
	var job workoutredis.VolumeCalcJob
	s.Require().NoError(json.Unmarshal([]byte(result[1]), &job))
	s.Equal(workout.ID, job.WorkoutID)
	s.Equal(workout.UserID, job.UserID)
}

func TestQueueIntegrationSuite(t *testing.T) {
	suite.Run(t, new(QueueIntegrationSuite))
}
