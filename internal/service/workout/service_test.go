package workout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	serviceworkout "github.com/vladgrskkh/onerep-api/internal/service/workout"
	workoutmocks "github.com/vladgrskkh/onerep-api/internal/service/workout/mocks"
)

type ServiceTestSuite struct {
	suite.Suite

	svc         *serviceworkout.WorkoutService
	workoutRepo *workoutmocks.MockWorkoutRepository
	tmplRepo    *workoutmocks.MockTemplateRepository
	progress    *workoutmocks.MockProgressRepository
	queue       *workoutmocks.MockVolumeQueue
	trManager   *workoutmocks.MockTransactionManager
}

func (s *ServiceTestSuite) SetupTest() {
	s.workoutRepo = workoutmocks.NewMockWorkoutRepository(s.T())
	s.tmplRepo = workoutmocks.NewMockTemplateRepository(s.T())
	s.progress = workoutmocks.NewMockProgressRepository(s.T())
	s.queue = workoutmocks.NewMockVolumeQueue(s.T())
	s.trManager = workoutmocks.NewMockTransactionManager(s.T())
	s.svc = serviceworkout.NewWorkoutService(s.workoutRepo, s.tmplRepo, s.progress, s.queue, s.trManager)
}

// expectTx runs the transaction closure inline.
func (s *ServiceTestSuite) expectTx() {
	s.trManager.EXPECT().
		Do(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
}

func (s *ServiceTestSuite) TestStart_Success() {
	userID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().
		List(mock.Anything, mock.MatchedBy(func(f domainworkout.WorkoutFilter) bool {
			return f.UserID == userID && f.Since.IsZero()
		})).
		Return([]*domainworkout.Workout{}, nil)

	var createdID uuid.UUID
	s.expectTx()
	s.workoutRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(w domainworkout.Workout) bool {
			return w.UserID == userID && w.TemplateID == uuid.Nil && w.FinishedAt.IsZero() && w.Version == 1
		})).
		RunAndReturn(func(_ context.Context, w domainworkout.Workout) (*domainworkout.Workout, error) {
			createdID = w.ID
			return &w, nil
		})
	s.workoutRepo.EXPECT().BatchInsertExercises(mock.Anything, []domainworkout.WorkoutExercise(nil)).Return(nil)

	created, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{UserID: userID})
	s.Require().NoError(err)
	s.Equal(createdID, created.ID)
	s.Equal(userID, created.UserID)
	s.Empty(created.Exercises)
}

func (s *ServiceTestSuite) TestStart_FromPublicTemplate() {
	userID := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())
	exerciseID1 := uuid.Must(uuid.NewV7())
	exerciseID2 := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{}, nil)
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).Return(&domaintemplate.Template{
		ID:              templateID,
		Name:            "Push Day",
		IsPublic:        true,
		CreatedByUserID: uuid.Must(uuid.NewV7()),
		Exercises: []domaintemplate.TemplateExercise{
			{TemplateID: templateID, ExerciseID: exerciseID1, SortOrder: 1, PlannedSets: 3},
			{TemplateID: templateID, ExerciseID: exerciseID2, SortOrder: 2, PlannedSets: 4},
		},
	}, nil)

	var createdID uuid.UUID
	s.expectTx()
	s.workoutRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(w domainworkout.Workout) bool {
			return w.TemplateID == templateID
		})).
		RunAndReturn(func(_ context.Context, w domainworkout.Workout) (*domainworkout.Workout, error) {
			createdID = w.ID
			return &w, nil
		})
	s.workoutRepo.EXPECT().
		BatchInsertExercises(mock.Anything, mock.MatchedBy(func(exercises []domainworkout.WorkoutExercise) bool {
			return len(exercises) == 2 &&
				exercises[0].WorkoutID == createdID &&
				exercises[0].ExerciseID == exerciseID1 &&
				exercises[0].SortOrder == 1 &&
				exercises[0].Notes == "" &&
				exercises[1].ExerciseID == exerciseID2 &&
				exercises[1].SortOrder == 2
		})).
		Return(nil)

	created, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{
		UserID:     userID,
		TemplateID: templateID,
	})
	s.Require().NoError(err)
	s.Equal(createdID, created.ID)
	s.Require().Len(created.Exercises, 2)
	s.Equal(exerciseID1, created.Exercises[0].ExerciseID)
	s.Equal(exerciseID2, created.Exercises[1].ExerciseID)
}

func (s *ServiceTestSuite) TestStart_FromPrivateOwnedTemplate() {
	userID := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{}, nil)
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).Return(&domaintemplate.Template{
		ID:              templateID,
		Name:            "Push Day",
		CreatedByUserID: userID,
		Exercises: []domaintemplate.TemplateExercise{
			{TemplateID: templateID, ExerciseID: exerciseID, SortOrder: 1, PlannedSets: 3},
		},
	}, nil)

	s.expectTx()
	s.workoutRepo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, w domainworkout.Workout) (*domainworkout.Workout, error) {
			return &w, nil
		})
	s.workoutRepo.EXPECT().
		BatchInsertExercises(mock.Anything, mock.AnythingOfType("[]workout.WorkoutExercise")).
		Return(nil)

	created, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{
		UserID:     userID,
		TemplateID: templateID,
	})
	s.Require().NoError(err)
	s.Require().Len(created.Exercises, 1)
	s.Equal(exerciseID, created.Exercises[0].ExerciseID)
}

func (s *ServiceTestSuite) TestStart_FromPrivateTemplateByOther() {
	userID := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{}, nil)
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).Return(&domaintemplate.Template{
		ID:              templateID,
		Name:            "Push Day",
		CreatedByUserID: uuid.Must(uuid.NewV7()),
	}, nil)

	_, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{
		UserID:     userID,
		TemplateID: templateID,
	})
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestStart_TemplateNotFound() {
	userID := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{}, nil)
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	_, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{
		UserID:     userID,
		TemplateID: templateID,
	})
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestStart_ActiveWorkoutExists() {
	userID := uuid.Must(uuid.NewV7())
	finished := time.Now().Add(-time.Hour)
	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{
		{ID: uuid.Must(uuid.NewV7()), UserID: userID, FinishedAt: finished},
		{ID: uuid.Must(uuid.NewV7()), UserID: userID},
	}, nil)

	_, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{UserID: userID})
	s.Require().ErrorIs(err, domainworkout.ErrActiveWorkout)
	s.workoutRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestStart_InvalidUserID() {
	_, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{})
	s.Require().ErrorIs(err, domainworkout.ErrInvalidUserID)
	s.workoutRepo.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
	s.workoutRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestStart_ListError() {
	userID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).
		Return(nil, errors.New("db down"))

	_, err := s.svc.Start(context.Background(), serviceworkout.StartWorkoutCommand{UserID: userID})
	s.Require().Error(err)
	s.workoutRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestGetActive_Success() {
	userID := uuid.Must(uuid.NewV7())
	activeID := uuid.Must(uuid.NewV7())
	finished := time.Now().Add(-time.Hour)
	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{
		{ID: uuid.Must(uuid.NewV7()), UserID: userID, FinishedAt: finished},
		{ID: activeID, UserID: userID},
	}, nil)

	got, err := s.svc.GetActive(context.Background(), userID)
	s.Require().NoError(err)
	s.Equal(activeID, got.ID)
}

func (s *ServiceTestSuite) TestGetActive_None() {
	userID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]*domainworkout.Workout{}, nil)

	_, err := s.svc.GetActive(context.Background(), userID)
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *ServiceTestSuite) TestGet_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	expected := &domainworkout.Workout{ID: workoutID, UserID: userID}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(expected, nil)

	got, err := s.svc.Get(context.Background(), workoutID, userID)
	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *ServiceTestSuite) TestGet_NotOwner() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := s.svc.Get(context.Background(), workoutID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *ServiceTestSuite) TestGet_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	_, err := s.svc.Get(context.Background(), workoutID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *ServiceTestSuite) TestList_Success() {
	userID := uuid.Must(uuid.NewV7())
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	expected := []*domainworkout.Workout{{ID: uuid.Must(uuid.NewV7()), UserID: userID}}
	s.workoutRepo.EXPECT().List(mock.Anything, domainworkout.WorkoutFilter{UserID: userID, Since: since}).
		Return(expected, nil)

	got, err := s.svc.List(context.Background(), userID, since)
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestList_NilUserRejected() {
	_, err := s.svc.List(context.Background(), uuid.Nil, time.Time{})
	s.Require().ErrorIs(err, domainworkout.ErrInvalidUserID)
	s.workoutRepo.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestAddExercise_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	expected := &domainworkout.WorkoutExercise{
		ID:         uuid.Must(uuid.NewV7()),
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
		SortOrder:  1,
	}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: userID}, nil)
	s.workoutRepo.EXPECT().AddExercise(mock.Anything, workoutID, exerciseID).Return(expected, nil)

	got, err := s.svc.AddExercise(context.Background(), serviceworkout.AddExerciseCommand{
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
	}, userID)
	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *ServiceTestSuite) TestAddExercise_NotOwner() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := s.svc.AddExercise(context.Background(), serviceworkout.AddExerciseCommand{
		WorkoutID:  workoutID,
		ExerciseID: uuid.Must(uuid.NewV7()),
	}, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "AddExercise", mock.Anything, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestAddExercise_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	_, err := s.svc.AddExercise(context.Background(), serviceworkout.AddExerciseCommand{
		WorkoutID:  workoutID,
		ExerciseID: uuid.Must(uuid.NewV7()),
	}, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *ServiceTestSuite) TestLogSet_IsPR() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	workoutExerciseID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	saved := domainworkout.WorkoutSet{
		ID:                uuid.Must(uuid.NewV7()),
		WorkoutExerciseID: workoutExerciseID,
		SetNumber:         1,
		WeightKg:          100,
		Reps:              5,
	}

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:     workoutID,
		UserID: userID,
		Exercises: []domainworkout.WorkoutExercise{
			{ID: workoutExerciseID, WorkoutID: workoutID, ExerciseID: exerciseID},
		},
	}, nil)
	s.expectTx()
	s.workoutRepo.EXPECT().
		LogSet(mock.Anything, workoutExerciseID, mock.MatchedBy(func(set domainworkout.WorkoutSet) bool {
			return set.WeightKg == 100 && set.Reps == 5 && set.IsWarmup
		})).
		Return(&saved, nil)
	s.progress.EXPECT().GetBest1RM(mock.Anything, exerciseID, userID).Return(0.0, nil)
	s.progress.EXPECT().
		Upsert1RM(mock.Anything, mock.MatchedBy(func(p domainprogress.Progress1RM) bool {
			return p.ExerciseID == exerciseID &&
				p.UserID == userID &&
				p.Estimated1RM == 116.7 &&
				time.Since(p.Date) < time.Minute
		})).
		Return(nil)

	result, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          100,
		Reps:              5,
		RPE:               8,
		RestSeconds:       90,
		IsWarmup:          true,
	}, userID)
	s.Require().NoError(err)
	s.True(result.IsPR)
	s.InDelta(116.7, result.Estimated1RM, 0.0001)
	s.Equal(saved, result.WorkoutSet)
}

func (s *ServiceTestSuite) TestLogSet_NotPR() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	workoutExerciseID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:     workoutID,
		UserID: userID,
		Exercises: []domainworkout.WorkoutExercise{
			{ID: workoutExerciseID, WorkoutID: workoutID, ExerciseID: exerciseID},
		},
	}, nil)
	s.expectTx()
	s.workoutRepo.EXPECT().LogSet(mock.Anything, workoutExerciseID, mock.Anything).
		Return(&domainworkout.WorkoutSet{SetNumber: 1, WeightKg: 60, Reps: 10}, nil)
	s.progress.EXPECT().GetBest1RM(mock.Anything, exerciseID, userID).Return(200.0, nil)

	result, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          60,
		Reps:              10,
	}, userID)
	s.Require().NoError(err)
	s.False(result.IsPR)
	s.InDelta(80.0, result.Estimated1RM, 0.0001)
	s.progress.AssertNotCalled(s.T(), "Upsert1RM", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestLogSet_RoundingExactness() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	workoutExerciseID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:     workoutID,
		UserID: userID,
		Exercises: []domainworkout.WorkoutExercise{
			{ID: workoutExerciseID, WorkoutID: workoutID, ExerciseID: exerciseID},
		},
	}, nil)
	s.expectTx()
	s.workoutRepo.EXPECT().LogSet(mock.Anything, workoutExerciseID, mock.Anything).
		Return(&domainworkout.WorkoutSet{SetNumber: 1, WeightKg: 82.5, Reps: 8}, nil)
	s.progress.EXPECT().GetBest1RM(mock.Anything, exerciseID, userID).Return(0.0, nil)
	s.progress.EXPECT().
		Upsert1RM(mock.Anything, mock.MatchedBy(func(p domainprogress.Progress1RM) bool {
			return p.Estimated1RM == 104.5
		})).
		Return(nil)

	result, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          82.5,
		Reps:              8,
	}, userID)
	s.Require().NoError(err)
	s.True(result.IsPR)
	s.InDelta(104.5, result.Estimated1RM, 0.0001)
}

func (s *ServiceTestSuite) TestLogSet_NotOwner() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: uuid.Must(uuid.NewV7()),
		WeightKg:          100,
		Reps:              5,
	}, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "LogSet", mock.Anything, mock.Anything, mock.Anything)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestLogSet_WorkoutNotFound() {
	s.workoutRepo.EXPECT().FindByID(mock.Anything, mock.Anything).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	_, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         uuid.Must(uuid.NewV7()),
		WorkoutExerciseID: uuid.Must(uuid.NewV7()),
		WeightKg:          100,
		Reps:              5,
	}, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *ServiceTestSuite) TestLogSet_WorkoutExerciseNotFound() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:        workoutID,
		UserID:    userID,
		Exercises: []domainworkout.WorkoutExercise{},
	}, nil)

	_, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: uuid.Must(uuid.NewV7()),
		WeightKg:          100,
		Reps:              5,
	}, userID)
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutExerciseNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "LogSet", mock.Anything, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestLogSet_InvalidValues() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	workoutExerciseID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:     workoutID,
		UserID: userID,
		Exercises: []domainworkout.WorkoutExercise{
			{ID: workoutExerciseID, WorkoutID: workoutID, ExerciseID: exerciseID},
		},
	}, nil)

	cases := []struct {
		name string
		cmd  serviceworkout.LogSetCommand
		want error
	}{
		{
			name: "zero weight",
			cmd:  serviceworkout.LogSetCommand{WeightKg: 0, Reps: 5},
			want: domainworkout.ErrInvalidWeight,
		},
		{
			name: "zero reps",
			cmd:  serviceworkout.LogSetCommand{WeightKg: 100, Reps: 0},
			want: domainworkout.ErrInvalidReps,
		},
		{
			name: "rpe out of range",
			cmd:  serviceworkout.LogSetCommand{WeightKg: 100, Reps: 5, RPE: 11},
			want: domainworkout.ErrInvalidRPE,
		},
		{
			name: "negative rest seconds",
			cmd:  serviceworkout.LogSetCommand{WeightKg: 100, Reps: 5, RestSeconds: -1},
			want: domainworkout.ErrInvalidRestSeconds,
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tc.cmd.WorkoutID = workoutID
			tc.cmd.WorkoutExerciseID = workoutExerciseID
			_, err := s.svc.LogSet(context.Background(), tc.cmd, userID)
			s.Require().ErrorIs(err, tc.want)
		})
	}
	s.workoutRepo.AssertNotCalled(s.T(), "LogSet", mock.Anything, mock.Anything, mock.Anything)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestLogSet_UpsertError() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	workoutExerciseID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).Return(&domainworkout.Workout{
		ID:     workoutID,
		UserID: userID,
		Exercises: []domainworkout.WorkoutExercise{
			{ID: workoutExerciseID, WorkoutID: workoutID, ExerciseID: exerciseID},
		},
	}, nil)
	s.expectTx()
	s.workoutRepo.EXPECT().LogSet(mock.Anything, workoutExerciseID, mock.Anything).
		Return(&domainworkout.WorkoutSet{SetNumber: 1, WeightKg: 100, Reps: 5}, nil)
	s.progress.EXPECT().GetBest1RM(mock.Anything, exerciseID, userID).Return(0.0, nil)
	s.progress.EXPECT().Upsert1RM(mock.Anything, mock.Anything).Return(errors.New("progress db down"))

	_, err := s.svc.LogSet(context.Background(), serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          100,
		Reps:              5,
	}, userID)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestFinish_Success() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	finished := domainworkout.Workout{ID: workoutID, UserID: userID}

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: userID}, nil)
	s.workoutRepo.EXPECT().Finish(mock.Anything, workoutID, mock.AnythingOfType("time.Time")).
		Return(&finished, nil)
	s.queue.EXPECT().EnqueueVolumeCalc(mock.Anything, finished).Return(nil)

	got, err := s.svc.Finish(context.Background(), serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID}, userID)
	s.Require().NoError(err)
	s.Equal(workoutID, got.ID)
}

func (s *ServiceTestSuite) TestFinish_NotOwner() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := s.svc.Finish(
		context.Background(),
		serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID},
		uuid.Must(uuid.NewV7()),
	)
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
	s.workoutRepo.AssertNotCalled(s.T(), "Finish", mock.Anything, mock.Anything, mock.Anything)
	s.queue.AssertNotCalled(s.T(), "EnqueueVolumeCalc", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestFinish_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	_, err := s.svc.Finish(
		context.Background(),
		serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID},
		uuid.Must(uuid.NewV7()),
	)
	s.Require().ErrorIs(err, domainworkout.ErrWorkoutNotFound)
	s.queue.AssertNotCalled(s.T(), "EnqueueVolumeCalc", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestFinish_EnqueueError() {
	userID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	finished := domainworkout.Workout{ID: workoutID, UserID: userID}

	s.workoutRepo.EXPECT().FindByID(mock.Anything, workoutID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: userID}, nil)
	s.workoutRepo.EXPECT().Finish(mock.Anything, workoutID, mock.AnythingOfType("time.Time")).
		Return(&finished, nil)
	s.queue.EXPECT().EnqueueVolumeCalc(mock.Anything, finished).Return(errors.New("redis down"))

	_, err := s.svc.Finish(context.Background(), serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID}, userID)
	s.Require().Error(err)
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
