package workout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	exercisemocks "github.com/vladgrskkh/onerep-api/internal/service/exercise/mocks"
	progressmocks "github.com/vladgrskkh/onerep-api/internal/service/progress/mocks"
	serviceworkout "github.com/vladgrskkh/onerep-api/internal/service/workout"
	workoutmocks "github.com/vladgrskkh/onerep-api/internal/service/workout/mocks"
)

// volumeDate is the UTC midnight the Aug 10 18:30 workouts in these tests
// materialize under.
func volumeDate() time.Time {
	return time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
}

type VolumeCalculatorTestSuite struct {
	suite.Suite

	calc         *serviceworkout.VolumeCalculator
	workoutRepo  *workoutmocks.MockWorkoutRepository
	exerciseRepo *exercisemocks.MockExerciseRepository
	progressRepo *progressmocks.MockProgressRepository
	trManager    *workoutmocks.MockTransactionManager
}

func (s *VolumeCalculatorTestSuite) SetupTest() {
	s.workoutRepo = workoutmocks.NewMockWorkoutRepository(s.T())
	s.exerciseRepo = exercisemocks.NewMockExerciseRepository(s.T())
	s.progressRepo = progressmocks.NewMockProgressRepository(s.T())
	s.trManager = workoutmocks.NewMockTransactionManager(s.T())
	s.calc = serviceworkout.NewVolumeCalculator(s.workoutRepo, s.exerciseRepo, s.progressRepo, s.trManager)
}

// expectTx runs the transaction closure inline.
func (s *VolumeCalculatorTestSuite) expectTx() {
	s.trManager.EXPECT().
		Do(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_AccumulatesPerMuscleGroup() {
	s.expectTx()
	userID := uuid.New()
	startedAt := time.Date(2026, time.August, 10, 18, 30, 0, 0, time.UTC)
	workout := s.workout(userID, startedAt)
	workout.Exercises[0].Sets = []domainworkout.WorkoutSet{
		{WeightKg: 100, Reps: 8},
		{WeightKg: 100, Reps: 8},
	}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, workout.Exercises[0].ExerciseID).Return(&domainexercise.Exercise{
		MuscleGroups: []domainexercise.ExerciseMuscleGroup{
			{MuscleGroupID: 1, IsPrimary: true},
			{MuscleGroupID: 2, IsPrimary: false},
		},
	}, nil)
	s.expectUpsert(1, userID, volumeDate(), 1600)
	s.expectUpsert(2, userID, volumeDate(), 1600)

	s.Require().NoError(s.calc.Recalculate(context.Background(), workout.ID, userID))
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_ExcludesWarmupSets() {
	s.expectTx()
	userID := uuid.New()
	startedAt := time.Date(2026, time.August, 10, 18, 30, 0, 0, time.UTC)
	workout := s.workout(userID, startedAt)
	workout.Exercises[0].Sets = []domainworkout.WorkoutSet{
		{WeightKg: 40, Reps: 10, IsWarmup: true},
		{WeightKg: 100, Reps: 8},
	}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, workout.Exercises[0].ExerciseID).Return(&domainexercise.Exercise{
		MuscleGroups: []domainexercise.ExerciseMuscleGroup{
			{MuscleGroupID: 1, IsPrimary: true},
		},
	}, nil)
	s.expectUpsert(1, userID, volumeDate(), 800)

	s.Require().NoError(s.calc.Recalculate(context.Background(), workout.ID, userID))
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_AggregatesAcrossExercises() {
	s.expectTx()
	userID := uuid.New()
	startedAt := time.Date(2026, time.August, 10, 18, 30, 0, 0, time.UTC)
	workout := s.workout(userID, startedAt)
	workout.Exercises[0].Sets = []domainworkout.WorkoutSet{{WeightKg: 100, Reps: 8}}
	workout.Exercises = append(workout.Exercises, domainworkout.WorkoutExercise{
		ExerciseID: uuid.Must(uuid.NewV7()),
		Sets:       []domainworkout.WorkoutSet{{WeightKg: 100, Reps: 8}},
	})
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	for _, we := range workout.Exercises {
		s.exerciseRepo.EXPECT().FindByID(mock.Anything, we.ExerciseID).Return(&domainexercise.Exercise{
			MuscleGroups: []domainexercise.ExerciseMuscleGroup{
				{MuscleGroupID: 1, IsPrimary: true},
			},
		}, nil)
	}
	s.expectUpsert(1, userID, volumeDate(), 1600)

	s.Require().NoError(s.calc.Recalculate(context.Background(), workout.ID, userID))
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_OwnershipMismatch() {
	userID := uuid.New()
	workout := s.workout(uuid.New(), time.Now())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)

	err := s.calc.Recalculate(context.Background(), workout.ID, userID)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_NoMuscleGroupsSkips() {
	s.expectTx()
	userID := uuid.New()
	workout := s.workout(userID, time.Now())
	workout.Exercises[0].Sets = []domainworkout.WorkoutSet{{WeightKg: 100, Reps: 8}}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	s.exerciseRepo.EXPECT().
		FindByID(mock.Anything, workout.Exercises[0].ExerciseID).
		Return(&domainexercise.Exercise{}, nil)

	s.Require().NoError(s.calc.Recalculate(context.Background(), workout.ID, userID))
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_EmptyWorkout() {
	s.expectTx()
	userID := uuid.New()
	workout := s.workout(userID, time.Now())
	workout.Exercises = nil
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)

	s.Require().NoError(s.calc.Recalculate(context.Background(), workout.ID, userID))
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_WorkoutRepoError() {
	userID := uuid.New()
	workout := s.workout(userID, time.Now())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(nil, domainworkout.ErrWorkoutNotFound)

	err := s.calc.Recalculate(context.Background(), workout.ID, userID)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_ExerciseRepoError() {
	userID := uuid.New()
	workout := s.workout(userID, time.Now())
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, workout.Exercises[0].ExerciseID).Return(nil, errors.New("db down"))

	err := s.calc.Recalculate(context.Background(), workout.ID, userID)
	s.ErrorContains(err, "db down")
}

func (s *VolumeCalculatorTestSuite) TestRecalculate_UpsertError() {
	s.expectTx()
	userID := uuid.New()
	workout := s.workout(userID, time.Now())
	workout.Exercises[0].Sets = []domainworkout.WorkoutSet{{WeightKg: 100, Reps: 8}}
	s.workoutRepo.EXPECT().FindByID(mock.Anything, workout.ID).Return(workout, nil)
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, workout.Exercises[0].ExerciseID).Return(&domainexercise.Exercise{
		MuscleGroups: []domainexercise.ExerciseMuscleGroup{{MuscleGroupID: 1, IsPrimary: true}},
	}, nil)
	s.progressRepo.EXPECT().
		UpsertVolume(mock.Anything, mock.Anything).
		Return(errors.New("db down"))

	err := s.calc.Recalculate(context.Background(), workout.ID, userID)
	s.ErrorContains(err, "db down")
}

// workout builds a workout with a single exercise and no sets.
func (s *VolumeCalculatorTestSuite) workout(userID uuid.UUID, startedAt time.Time) *domainworkout.Workout {
	return &domainworkout.Workout{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		StartedAt: startedAt,
		Exercises: []domainworkout.WorkoutExercise{{
			ExerciseID: uuid.Must(uuid.NewV7()),
		}},
	}
}

// expectUpsert asserts one volume upsert for the group with the workout's
// total and the started_at day truncated to UTC midnight.
// expectUpsert asserts one volume upsert for the group on the given day with
// the workout's total.
func (s *VolumeCalculatorTestSuite) expectUpsert(
	muscleGroupID int,
	userID uuid.UUID,
	date time.Time,
	total float64,
) {
	s.progressRepo.EXPECT().
		UpsertVolume(mock.Anything, mock.MatchedBy(func(p domainprogress.ProgressVolume) bool {
			return p.MuscleGroupID == muscleGroupID && p.UserID == userID && p.Date.Equal(date) && p.TotalKG == total
		})).
		Return(nil)
}

func TestVolumeCalculatorSuite(t *testing.T) {
	suite.Run(t, new(VolumeCalculatorTestSuite))
}
