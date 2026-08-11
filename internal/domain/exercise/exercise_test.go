package exercise_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

type ExerciseTestSuite struct {
	suite.Suite
}

func (s *ExerciseTestSuite) TestNewExercise() {
	createdBy := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	e, err := exercise.NewExercise(
		"  Bench Press  ",
		"Chest press with a barbell",
		"Keep shoulders back",
		createdBy,
	)

	s.Require().NoError(err)
	s.NotEqual(uuid.Nil, e.ID)
	s.Equal(uuid.Version(7), e.ID.Version())
	s.Equal("Bench Press", e.Name)
	s.Equal("Chest press with a barbell", e.Description)
	s.Equal("Keep shoulders back", e.Notes)
	s.False(e.IsBuiltIn)
	s.Equal(createdBy, e.CreatedByUserID)
	s.False(e.CreatedAt.IsZero())
	s.False(e.UpdatedAt.IsZero())
	s.WithinDuration(time.Now(), e.CreatedAt, time.Minute)
	s.True(e.DeletedAt.IsZero())
	s.Equal(1, e.Version)
	s.Empty(e.Media)
	s.Empty(e.MuscleGroups)
}

func (s *ExerciseTestSuite) TestNewExercise_BuiltIn() {
	e, err := exercise.NewExercise("Squat", "desc", "notes", uuid.Nil)

	s.Require().NoError(err)
	s.Equal(uuid.Nil, e.CreatedByUserID)
}

func (s *ExerciseTestSuite) TestNewExercise_Validation() {
	createdBy := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	tests := []struct {
		name            string
		exerciseName    string
		createdByUserID uuid.UUID
		wantErr         error
	}{
		{name: "valid", exerciseName: "Bench Press", createdByUserID: createdBy},
		{name: "blank name", exerciseName: "   ", createdByUserID: createdBy, wantErr: exercise.ErrInvalidName},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			e, err := exercise.NewExercise(tt.exerciseName, "desc", "notes", tt.createdByUserID)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, e.ID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.exerciseName, e.Name)
		})
	}
}

func (s *ExerciseTestSuite) TestNewMuscleGroup() {
	mg, err := exercise.NewMuscleGroup(1, " chest ")

	s.Require().NoError(err)
	s.Equal(1, mg.ID)
	s.Equal("chest", mg.Name)
}

func (s *ExerciseTestSuite) TestNewMuscleGroup_Validation() {
	tests := []struct {
		name    string
		id      int
		mgName  string
		wantErr error
	}{
		{name: "valid", id: 1, mgName: "chest"},
		{name: "zero id", id: 0, mgName: "chest", wantErr: exercise.ErrInvalidID},
		{name: "blank name", id: 1, mgName: " ", wantErr: exercise.ErrInvalidName},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := exercise.NewMuscleGroup(tt.id, tt.mgName)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func (s *ExerciseTestSuite) TestMediaTypeConstants() {
	s.Equal(exercise.MediaTypePhoto, exercise.MediaType("photo"))
	s.Equal(exercise.MediaTypeVideo, exercise.MediaType("video"))
}

func (s *ExerciseTestSuite) TestErrors() {
	s.Require().ErrorContains(exercise.ErrExerciseNotFound, "exercise not found")
	s.Require().ErrorContains(exercise.ErrCannotEditBuiltIn, "built-in")
}

func TestExerciseSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ExerciseTestSuite))
}
