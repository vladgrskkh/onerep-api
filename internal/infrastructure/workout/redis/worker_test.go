package redis_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	workoutredis "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/redis"
	redismocks "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/redis/mocks"
)

// workerStopTimeout bounds how long a test waits for the worker to stop
// after its context is cancelled. miniredis never aborts a blocked BRPop on
// disconnect, so the worker only notices the cancellation after the 5s
// block timeout elapses.
const workerStopTimeout = 7 * time.Second

// outageRecoveryWait keeps the server failing long enough for the worker to
// hit the BRPop error path and start backing off.
const outageRecoveryWait = 2 * time.Second

type WorkerTestSuite struct {
	suite.Suite

	server     *miniredis.Miniredis
	client     *redis.Client
	queue      *workoutredis.Queue
	calculator *redismocks.MockVolumeCalculator
	worker     *workoutredis.Worker
}

func (s *WorkerTestSuite) SetupTest() {
	server, err := miniredis.Run()
	s.Require().NoError(err)
	s.server = server
	opts, err := redis.ParseURL("redis://" + server.Addr() + "/0")
	s.Require().NoError(err)
	s.client = redis.NewClient(opts)
	s.queue = workoutredis.NewQueue(s.client)
	s.calculator = redismocks.NewMockVolumeCalculator(s.T())
	s.worker = workoutredis.NewWorker(
		s.client,
		s.calculator,
		slog.New(slog.DiscardHandler),
	)
}

func (s *WorkerTestSuite) TearDownTest() {
	if s.client != nil {
		s.Require().NoError(s.client.Close())
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *WorkerTestSuite) TestRun_ProcessesJob() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, workout))
	recalculated := s.expectRecalculate(workout, nil)

	done := s.runWorker(ctx)
	s.awaitCall(recalculated)
	s.awaitStop(cancel, done)
}

func (s *WorkerTestSuite) TestRun_SkipsMalformedJobAndContinues() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Require().NoError(s.client.LPush(ctx, volumeCalcQueueKey, "not-json").Err())
	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, workout))
	recalculated := s.expectRecalculate(workout, nil)

	done := s.runWorker(ctx)
	s.awaitCall(recalculated)
	s.awaitStop(cancel, done)
}

func (s *WorkerTestSuite) TestRun_KeepsGoingAfterCalculatorError() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	second := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, first))
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, second))
	firstRecalculated := s.expectRecalculate(first, errBoom)
	secondRecalculated := s.expectRecalculate(second, nil)

	done := s.runWorker(ctx)
	s.awaitCall(firstRecalculated)
	s.awaitCall(secondRecalculated)
	s.awaitStop(cancel, done)
}

func (s *WorkerTestSuite) TestRun_StopsImmediatelyWhenCancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := s.runWorker(ctx)
	s.awaitStop(func() {}, done)
}

func (s *WorkerTestSuite) TestRun_SurvivesRedisOutage() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.server.SetError("server down")
	done := s.runWorker(ctx)
	time.Sleep(outageRecoveryWait)
	s.server.SetError("")

	workout := domainworkout.Workout{ID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
	s.Require().NoError(s.queue.EnqueueVolumeCalc(ctx, workout))
	recalculated := s.expectRecalculate(workout, nil)

	s.awaitCall(recalculated)
	s.awaitStop(cancel, done)
}

// expectRecalculate wires the calculator to signal on the returned channel
// when the given workout is processed.
func (s *WorkerTestSuite) expectRecalculate(workout domainworkout.Workout, recalcErr error) <-chan struct{} {
	called := make(chan struct{})
	s.calculator.EXPECT().
		Recalculate(mock.Anything, workout.ID, workout.UserID).
		Run(func(context.Context, uuid.UUID, uuid.UUID) { close(called) }).
		Return(recalcErr).
		Once()
	return called
}

// runWorker runs the worker until the context is cancelled.
func (s *WorkerTestSuite) runWorker(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		s.worker.Run(ctx)
		close(done)
	}()
	return done
}

// awaitCall fails the test if the calculator is not invoked in time.
func (s *WorkerTestSuite) awaitCall(called <-chan struct{}) {
	select {
	case <-called:
	case <-time.After(workerStopTimeout):
		s.Fail("calculator was not invoked in time")
	}
}

// awaitStop cancels the worker and fails the test if it does not stop in
// time.
func (s *WorkerTestSuite) awaitStop(cancel context.CancelFunc, done <-chan struct{}) {
	cancel()
	select {
	case <-done:
	case <-time.After(workerStopTimeout):
		s.Fail("worker did not stop after context cancellation")
	}
}

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }

func TestWorkerSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}
