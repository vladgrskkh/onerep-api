package redis_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	workoutredis "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/redis"
)

// volumeCalcQueueKey mirrors the queue key the queue implementation uses.
const volumeCalcQueueKey = "gym:volume:recalculate"

type QueueTestSuite struct {
	suite.Suite

	server *miniredis.Miniredis
	client *redis.Client
	queue  *workoutredis.Queue
}

func (s *QueueTestSuite) SetupTest() {
	server, err := miniredis.Run()
	s.Require().NoError(err)
	s.server = server
	opts, err := redis.ParseURL("redis://" + server.Addr() + "/0")
	s.Require().NoError(err)
	s.client = redis.NewClient(opts)
	s.queue = workoutredis.NewQueue(s.client)
}

func (s *QueueTestSuite) TearDownTest() {
	if s.client != nil {
		s.Require().NoError(s.client.Close())
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *QueueTestSuite) TestEnqueueVolumeCalc_StoresJSONJob() {
	ctx := context.Background()
	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, workout))

	payload, err := s.client.LPop(ctx, volumeCalcQueueKey).Result()
	s.Require().NoError(err)
	var job workoutredis.VolumeCalcJob
	s.Require().NoError(json.Unmarshal([]byte(payload), &job))
	s.Equal(workout.ID, job.WorkoutID)
	s.Equal(workout.UserID, job.UserID)
}

func (s *QueueTestSuite) TestEnqueueVolumeCalc_RedisUnavailable() {
	ctx := context.Background()
	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.server.Close()

	err := s.queue.EnqueueVolumeCalc(ctx, workout)
	s.Error(err)
}

func (s *QueueTestSuite) TestPopVolumeCalcJob_RoundTrip() {
	ctx := context.Background()
	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, workout))

	job, err := s.queue.PopVolumeCalcJob(ctx)
	s.Require().NoError(err)
	s.Equal(workout.ID, job.WorkoutID)
	s.Equal(workout.UserID, job.UserID)
}

func (s *QueueTestSuite) TestPopVolumeCalcJob_MalformedPayload() {
	ctx := context.Background()
	s.Require().NoError(s.client.LPush(ctx, volumeCalcQueueKey, "not-json").Err())

	_, err := s.queue.PopVolumeCalcJob(ctx)
	s.Require().Error(err)
	s.Require().NotErrorIs(err, workoutredis.ErrNoVolumeJob)
}

func TestQueueSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}
