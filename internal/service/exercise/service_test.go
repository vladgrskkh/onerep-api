package exercise_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	exercisemocks "github.com/vladgrskkh/onerep-api/internal/service/exercise/mocks"
)

type ServiceTestSuite struct {
	suite.Suite

	svc             *serviceexercise.ExerciseService
	exerciseRepo    *exercisemocks.MockExerciseRepository
	muscleGroupRepo *exercisemocks.MockMuscleGroupRepository
	trManager       *exercisemocks.MockTransactionManager
}

func (s *ServiceTestSuite) SetupTest() {
	s.exerciseRepo = exercisemocks.NewMockExerciseRepository(s.T())
	s.muscleGroupRepo = exercisemocks.NewMockMuscleGroupRepository(s.T())
	s.trManager = exercisemocks.NewMockTransactionManager(s.T())
	s.svc = serviceexercise.NewExerciseService(s.exerciseRepo, s.muscleGroupRepo, s.trManager)
}

// expectTx runs the transaction closure inline.
func (s *ServiceTestSuite) expectTx() {
	s.trManager.EXPECT().
		Do(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
}

func (s *ServiceTestSuite) TestList_PassesFilters() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	filter := domainexercise.ExerciseFilter{
		Search:      "squat",
		MuscleGroup: "legs",
		Since:       &since,
	}
	expected := []domainexercise.Exercise{{ID: uuid.Must(uuid.NewV7()), Name: "Squat"}}
	s.exerciseRepo.EXPECT().List(mock.Anything, filter).Return(expected, nil)

	got, err := s.svc.List(context.Background(), serviceexercise.ListExercisesCommand{
		Search:      "squat",
		MuscleGroup: "legs",
		Since:       &since,
	})
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], *got[0])
}

func (s *ServiceTestSuite) TestGet_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	expected := domainexercise.Exercise{ID: exerciseID, Name: "Bench Press"}
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).Return(expected, nil)

	got, err := s.svc.Get(context.Background(), exerciseID)
	s.Require().NoError(err)
	s.Equal(expected, *got)
}

func (s *ServiceTestSuite) TestGet_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound)

	_, err := s.svc.Get(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
}

func (s *ServiceTestSuite) TestCreate_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	groups := []domainexercise.ExerciseMuscleGroup{{MuscleGroupID: 1}, {MuscleGroupID: 2}}

	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]domainexercise.MuscleGroup{{ID: 1, Name: "Chest"}, {ID: 2, Name: "Back"}}, nil)
	s.expectTx()
	s.exerciseRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(ex domainexercise.Exercise) bool {
			return ex.Name == "Bench Press" && ex.CreatedByUserID != nil && *ex.CreatedByUserID == userID
		})).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Bench Press"}, nil)
	s.exerciseRepo.EXPECT().BatchInsertMuscleGroups(mock.Anything, exerciseID, groups).Return(nil)

	created, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		Description:    "Chest press",
		Notes:          "Medium grip",
		UserID:         userID,
		MuscleGroupIDs: []int{1, 2},
	})
	s.Require().NoError(err)
	s.Equal(exerciseID, created.ID)
	s.Equal(groups, created.MuscleGroups)
	s.exerciseRepo.AssertCalled(s.T(), "BatchInsertMuscleGroups", mock.Anything, exerciseID, groups)
}

func (s *ServiceTestSuite) TestCreate_InvalidName() {
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:   "   ",
		UserID: uuid.Must(uuid.NewV7()),
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidName)
	s.exerciseRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_InvalidUserID() {
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{Name: "Squat"})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidUserID)
	s.exerciseRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_InvalidMuscleGroupID() {
	userID := uuid.Must(uuid.NewV7())
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		UserID:         userID,
		MuscleGroupIDs: []int{0},
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidID)
	s.muscleGroupRepo.AssertNotCalled(s.T(), "List", mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_MuscleGroupNotFound() {
	userID := uuid.Must(uuid.NewV7())
	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]domainexercise.MuscleGroup{{ID: 1, Name: "Chest"}}, nil)

	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		UserID:         userID,
		MuscleGroupIDs: []int{1, 99},
	})
	s.Require().ErrorIs(err, domainexercise.ErrMuscleGroupMissing)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	newName := "Incline Bench Press"
	newGroups := []domainexercise.ExerciseMuscleGroup{{MuscleGroupID: 3}}

	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{
			ID:        exerciseID,
			Name:      "Bench Press",
			IsBuiltIn: false,
			Version:   3,
		}, nil)
	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]domainexercise.MuscleGroup{{ID: 3, Name: "Shoulders"}}, nil)
	s.expectTx()
	s.exerciseRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(ex domainexercise.Exercise) bool {
			return ex.ID == exerciseID && ex.Name == newName
		})).
		Return(domainexercise.Exercise{ID: exerciseID, Name: newName, Version: 4}, nil)
	s.exerciseRepo.EXPECT().ReplaceMuscleGroups(mock.Anything, exerciseID, newGroups).Return(nil)

	updated, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:             exerciseID,
		Name:           &newName,
		MuscleGroupIDs: &[]int{3},
	})
	s.Require().NoError(err)
	s.Equal(newName, updated.Name)
	s.Equal(newGroups, updated.MuscleGroups)
}

func (s *ServiceTestSuite) TestUpdate_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{ID: exerciseID})
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Squat", IsBuiltIn: true}, nil)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: new("Front Squat"),
	})
	s.Require().ErrorIs(err, domainexercise.ErrCannotEditBuiltIn)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_InvalidName() {
	exerciseID := uuid.Must(uuid.NewV7())
	empty := "   "
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: &empty,
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidName)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestSoftDelete_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)
	s.exerciseRepo.EXPECT().SoftDelete(mock.Anything, exerciseID).Return(nil)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSoftDelete_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
	s.exerciseRepo.AssertNotCalled(s.T(), "SoftDelete", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestSoftDelete_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Squat", IsBuiltIn: true}, nil)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrCannotEditBuiltIn)
	s.exerciseRepo.AssertNotCalled(s.T(), "SoftDelete", mock.Anything, mock.Anything)
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
