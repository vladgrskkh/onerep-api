package workout_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

type WorkoutTestSuite struct {
	suite.Suite
}

func (s *WorkoutTestSuite) TestNewWorkout() {
	userID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	templateID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	w := workout.NewWorkout(userID, &templateID)

	s.NotEqual(uuid.Nil, w.ID)
	s.Equal(uuid.Version(7), w.ID.Version())
	s.Equal(userID, w.UserID)
	s.NotNil(w.TemplateID)
	s.Equal(templateID, *w.TemplateID)
	s.False(w.StartedAt.IsZero())
	s.WithinDuration(time.Now(), w.StartedAt, time.Minute)
	s.Nil(w.FinishedAt)
	s.Empty(w.Notes)
	s.False(w.CreatedAt.IsZero())
	s.False(w.UpdatedAt.IsZero())
	s.Nil(w.DeletedAt)
	s.Equal(1, w.Version)
	s.Empty(w.Exercises)
}

func (s *WorkoutTestSuite) TestNewWorkout_NoTemplate() {
	userID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	w := workout.NewWorkout(userID, nil)

	s.Nil(w.TemplateID)
	s.Nil(w.FinishedAt)
	s.Equal(userID, w.UserID)
}

func (s *WorkoutTestSuite) TestNewWorkoutExercise() {
	workoutID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	exerciseID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	we := workout.NewWorkoutExercise(workoutID, exerciseID)

	s.NotEqual(uuid.Nil, we.ID)
	s.Equal(uuid.Version(7), we.ID.Version())
	s.Equal(workoutID, we.WorkoutID)
	s.Equal(exerciseID, we.ExerciseID)
	s.Equal(0, we.SortOrder)
	s.Empty(we.Notes)
}

func (s *WorkoutTestSuite) TestNewWorkoutSet() {
	workoutExerciseID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	rpe := 8
	restSeconds := 90

	ws := workout.NewWorkoutSet(workoutExerciseID, 82.5, 5, &rpe, &restSeconds, false)

	s.NotEqual(uuid.Nil, ws.ID)
	s.Equal(uuid.Version(7), ws.ID.Version())
	s.Equal(workoutExerciseID, ws.WorkoutExerciseID)
	s.InEpsilon(82.5, ws.WeightKg, 1e-6)
	s.Equal(5, ws.Reps)
	s.NotNil(ws.RPE)
	s.Equal(8, *ws.RPE)
	s.NotNil(ws.RestSeconds)
	s.Equal(90, *ws.RestSeconds)
	s.False(ws.IsWarmup)
}

func (s *WorkoutTestSuite) TestNewWorkoutSet_OptionalNil() {
	workoutExerciseID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	ws := workout.NewWorkoutSet(workoutExerciseID, 20, 12, nil, nil, true)

	s.Nil(ws.RPE)
	s.Nil(ws.RestSeconds)
	s.True(ws.IsWarmup)
}

func (s *WorkoutTestSuite) TestErrors() {
	s.Require().ErrorContains(workout.ErrWorkoutNotFound, "workout not found")
	s.Require().ErrorContains(workout.ErrActiveWorkout, "active workout")
}

func TestWorkoutSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(WorkoutTestSuite))
}
