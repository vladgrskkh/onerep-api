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

	e := exercise.NewExercise("Bench Press", "Chest press with a barbell", "Keep shoulders back", createdBy)

	s.NotEqual(uuid.Nil, e.ID)
	s.Equal(uuid.Version(7), e.ID.Version())
	s.Equal("Bench Press", e.Name)
	s.Equal("Chest press with a barbell", e.Description)
	s.Equal("Keep shoulders back", e.Notes)
	s.False(e.IsBuiltIn)
	s.NotNil(e.CreatedByUserID)
	s.Equal(createdBy, *e.CreatedByUserID)
	s.False(e.CreatedAt.IsZero())
	s.False(e.UpdatedAt.IsZero())
	s.WithinDuration(time.Now(), e.CreatedAt, time.Minute)
	s.Nil(e.DeletedAt)
	s.Equal(1, e.Version)
	s.Empty(e.Media)
	s.Empty(e.MuscleGroups)
}

func (s *ExerciseTestSuite) TestNewMuscleGroup() {
	mg := exercise.NewMuscleGroup(1, "chest")

	s.Equal(1, mg.ID)
	s.Equal("chest", mg.Name)
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
