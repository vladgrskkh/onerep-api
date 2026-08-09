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

	w, err := workout.NewWorkout(userID, &templateID)

	s.Require().NoError(err)
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

	w, err := workout.NewWorkout(userID, nil)

	s.Require().NoError(err)
	s.Nil(w.TemplateID)
	s.Nil(w.FinishedAt)
	s.Equal(userID, w.UserID)
}

func (s *WorkoutTestSuite) TestNewWorkout_Validation() {
	userID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	tests := []struct {
		name    string
		userID  uuid.UUID
		wantErr error
	}{
		{name: "valid", userID: userID},
		{name: "nil user id", userID: uuid.Nil, wantErr: workout.ErrInvalidUserID},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			w, err := workout.NewWorkout(tt.userID, nil)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, w.ID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.userID, w.UserID)
		})
	}
}

func (s *WorkoutTestSuite) TestNewWorkoutExercise() {
	workoutID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	exerciseID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	we, err := workout.NewWorkoutExercise(workoutID, exerciseID)

	s.Require().NoError(err)
	s.NotEqual(uuid.Nil, we.ID)
	s.Equal(uuid.Version(7), we.ID.Version())
	s.Equal(workoutID, we.WorkoutID)
	s.Equal(exerciseID, we.ExerciseID)
	s.Equal(0, we.SortOrder)
	s.Empty(we.Notes)
}

func (s *WorkoutTestSuite) TestNewWorkoutExercise_Validation() {
	workoutID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	exerciseID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	tests := []struct {
		name       string
		workoutID  uuid.UUID
		exerciseID uuid.UUID
		wantErr    error
	}{
		{name: "valid", workoutID: workoutID, exerciseID: exerciseID},
		{name: "nil workout id", workoutID: uuid.Nil, exerciseID: exerciseID, wantErr: workout.ErrInvalidWorkoutID},
		{name: "nil exercise id", workoutID: workoutID, exerciseID: uuid.Nil, wantErr: workout.ErrInvalidExerciseID},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			we, err := workout.NewWorkoutExercise(tt.workoutID, tt.exerciseID)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, we.ID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.workoutID, we.WorkoutID)
			s.Equal(tt.exerciseID, we.ExerciseID)
		})
	}
}

func (s *WorkoutTestSuite) TestNewWorkoutSet() {
	workoutExerciseID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	rpe := 8
	restSeconds := 90

	ws, err := workout.NewWorkoutSet(workoutExerciseID, 82.5, 5, &rpe, &restSeconds, false)

	s.Require().NoError(err)
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

	ws, err := workout.NewWorkoutSet(workoutExerciseID, 20, 12, nil, nil, true)

	s.Require().NoError(err)
	s.Nil(ws.RPE)
	s.Nil(ws.RestSeconds)
	s.True(ws.IsWarmup)
}

func (s *WorkoutTestSuite) TestNewWorkoutSet_Validation() {
	weID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	rpe := 8
	rpeBelowRange := 0
	rpeAboveRange := 11
	restSeconds := 90
	negativeRestSeconds := -1

	tests := []struct {
		name              string
		workoutExerciseID uuid.UUID
		weightKg          float64
		reps              int
		rpe               *int
		restSeconds       *int
		wantErr           error
	}{
		{name: "valid", workoutExerciseID: weID, weightKg: 82.5, reps: 5, rpe: &rpe, restSeconds: &restSeconds},
		{
			name:              "nil workout exercise id",
			workoutExerciseID: uuid.Nil,
			weightKg:          82.5,
			reps:              5,
			wantErr:           workout.ErrInvalidWorkoutExerciseID,
		},
		{name: "zero weight", workoutExerciseID: weID, weightKg: 0, reps: 5, wantErr: workout.ErrInvalidWeight},
		{name: "zero reps", workoutExerciseID: weID, weightKg: 82.5, reps: 0, wantErr: workout.ErrInvalidReps},
		{
			name:              "rpe below range",
			workoutExerciseID: weID,
			weightKg:          82.5,
			reps:              5,
			rpe:               &rpeBelowRange,
			wantErr:           workout.ErrInvalidRPE,
		},
		{
			name:              "rpe above range",
			workoutExerciseID: weID,
			weightKg:          82.5,
			reps:              5,
			rpe:               &rpeAboveRange,
			wantErr:           workout.ErrInvalidRPE,
		},
		{
			name:              "negative rest seconds",
			workoutExerciseID: weID,
			weightKg:          82.5,
			reps:              5,
			restSeconds:       &negativeRestSeconds,
			wantErr:           workout.ErrInvalidRestSeconds,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			ws, err := workout.NewWorkoutSet(
				tt.workoutExerciseID,
				tt.weightKg,
				tt.reps,
				tt.rpe,
				tt.restSeconds,
				false,
			)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, ws.ID)
				return
			}
			s.Require().NoError(err)
			s.InEpsilon(tt.weightKg, ws.WeightKg, 1e-6)
			s.Equal(tt.reps, ws.Reps)
		})
	}
}

func (s *WorkoutTestSuite) TestErrors() {
	s.Require().ErrorContains(workout.ErrWorkoutNotFound, "workout not found")
	s.Require().ErrorContains(workout.ErrActiveWorkout, "active workout")
}

func TestWorkoutSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(WorkoutTestSuite))
}
