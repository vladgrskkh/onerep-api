//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/postgres"
)

type WorkoutRepoTestSuite struct {
	suite.Suite

	pool         *pgxpool.Pool
	repo         *postgres.WorkoutRepo
	trManager    *manager.Manager
	ctx          context.Context
	created      []uuid.UUID
	exercisesIDs []uuid.UUID
}

func (s *WorkoutRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewWorkoutRepo(pool)
	s.trManager = manager.Must(trmpgx.NewDefaultFactory(pool))
	s.ctx = context.Background()
}

func (s *WorkoutRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		for _, id := range s.created {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.workouts WHERE id = $1`, id)
		}
		for _, id := range s.exercisesIDs {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.exercises WHERE id = $1`, id)
		}
		s.pool.Close()
	}
}

func (s *WorkoutRepoTestSuite) insertTestExercise() uuid.UUID {
	id := uuid.Must(uuid.NewV7())
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO gym.exercises (id, name, description, created_by_user_id, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, now(), now(), 1)
	`, id, "workout-test-exercise-"+uuid.NewString(), "Fixture exercise", uuid.New())
	s.Require().NoError(err)
	s.exercisesIDs = append(s.exercisesIDs, id)
	return id
}

func (s *WorkoutRepoTestSuite) newTestWorkout(userID uuid.UUID) domainworkout.Workout {
	w, err := domainworkout.NewWorkout(userID, nil)
	s.Require().NoError(err)
	ex1 := s.insertTestExercise()
	ex2 := s.insertTestExercise()

	we1, err := domainworkout.NewWorkoutExercise(w.ID, ex1)
	s.Require().NoError(err)
	we1.SortOrder = 0
	set1, err := domainworkout.NewWorkoutSet(we1.ID, 80, 5, nil, nil, false)
	s.Require().NoError(err)
	set1.SetNumber = 1
	set2, err := domainworkout.NewWorkoutSet(we1.ID, 82.5, 3, nil, nil, false)
	s.Require().NoError(err)
	set2.SetNumber = 2
	we1.Sets = []domainworkout.WorkoutSet{set1, set2}

	we2, err := domainworkout.NewWorkoutExercise(w.ID, ex2)
	s.Require().NoError(err)
	we2.SortOrder = 1

	w.Exercises = []domainworkout.WorkoutExercise{we1, we2}
	return w
}

func (s *WorkoutRepoTestSuite) createTestWorkout(w domainworkout.Workout) domainworkout.Workout {
	var created domainworkout.Workout
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, w)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)
	return created
}

func (s *WorkoutRepoTestSuite) updateTestWorkout(w domainworkout.Workout) (domainworkout.Workout, error) {
	var updated domainworkout.Workout
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		updated, err = s.repo.Update(ctx, w)
		return err
	})
	return updated, err
}

func (s *WorkoutRepoTestSuite) TestCreateAndFindByID() {
	w := s.newTestWorkout(uuid.New())
	created := s.createTestWorkout(w)
	s.Equal(w.ID, created.ID)
	s.Equal(1, created.Version)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.Equal(w.UserID, found.UserID)
	s.Nil(found.TemplateID)
	s.False(found.StartedAt.IsZero())
	s.Nil(found.FinishedAt)
	s.Equal(1, found.Version)
	s.Require().Len(found.Exercises, 2)
	s.Equal(0, found.Exercises[0].SortOrder)
	s.Equal(1, found.Exercises[1].SortOrder)
	s.Require().Len(found.Exercises[0].Sets, 2)
	s.Equal(1, found.Exercises[0].Sets[0].SetNumber)
	s.InEpsilon(80, found.Exercises[0].Sets[0].WeightKg, 1e-6)
	s.Equal(5, found.Exercises[0].Sets[0].Reps)
	s.Equal(2, found.Exercises[0].Sets[1].SetNumber)
	s.InEpsilon(82.5, found.Exercises[0].Sets[1].WeightKg, 1e-6)
	s.Equal(3, found.Exercises[0].Sets[1].Reps)
	s.Empty(found.Exercises[1].Sets)
}

func (s *WorkoutRepoTestSuite) TestCreate_WithTemplateAndNotes() {
	userID := uuid.New()
	templateID := uuid.New()
	w, err := domainworkout.NewWorkout(userID, &templateID)
	s.Require().NoError(err)
	w.Notes = "push day"
	s.createTestWorkout(w)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.NotNil(found.TemplateID)
	s.Equal(templateID, *found.TemplateID)
	s.Equal("push day", found.Notes)
}

func (s *WorkoutRepoTestSuite) TestFindByID_NotFound() {
	_, err := s.repo.FindByID(s.ctx, uuid.New())
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *WorkoutRepoTestSuite) TestList_UserIDFilter() {
	userID := uuid.New()
	w1 := s.newTestWorkout(userID)
	w1.Notes = "first"
	s.createTestWorkout(w1)
	w2 := s.newTestWorkout(uuid.New())
	w2.Notes = "second"
	s.createTestWorkout(w2)

	listed, err := s.repo.List(s.ctx, domainworkout.WorkoutFilter{UserID: &userID})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(w1.ID, listed[0].ID)
	s.Equal("first", listed[0].Notes)
	s.Empty(listed[0].Exercises)
}

func (s *WorkoutRepoTestSuite) TestList_SinceFilter() {
	userID := uuid.New()
	w := s.newTestWorkout(userID)
	s.createTestWorkout(w)

	since := time.Now().Add(-time.Hour)
	listed, err := s.repo.List(s.ctx, domainworkout.WorkoutFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(w.ID, listed[0].ID)

	since = time.Now().Add(time.Hour)
	listed, err = s.repo.List(s.ctx, domainworkout.WorkoutFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *WorkoutRepoTestSuite) TestList_ExcludesSoftDeleted() {
	userID := uuid.New()
	w := s.createTestWorkout(s.newTestWorkout(userID))

	err := s.repo.SoftDelete(s.ctx, w.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, domainworkout.WorkoutFilter{UserID: &userID})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(w.ID, l.ID)
	}

	_, err = s.repo.FindByID(s.ctx, w.ID)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *WorkoutRepoTestSuite) TestAddExercise_IncrementsSortOrder() {
	w := s.createTestWorkout(s.newTestWorkout(uuid.New()))

	ex := s.insertTestExercise()
	we, err := s.repo.AddExercise(s.ctx, w.ID, ex)
	s.Require().NoError(err)
	s.Equal(2, we.SortOrder)
	s.Equal(w.ID, we.WorkoutID)
	s.Equal(ex, we.ExerciseID)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.Require().Len(found.Exercises, 3)
	s.Equal(2, found.Exercises[2].SortOrder)
	s.Equal(ex, found.Exercises[2].ExerciseID)
}

func (s *WorkoutRepoTestSuite) TestLogSet_IncrementsSetNumber() {
	w := s.createTestWorkout(s.newTestWorkout(uuid.New()))
	we := w.Exercises[0]

	set, err := domainworkout.NewWorkoutSet(we.ID, 100, 2, nil, nil, true)
	s.Require().NoError(err)
	logged, err := s.repo.LogSet(s.ctx, we.ID, set)
	s.Require().NoError(err)
	s.Equal(3, logged.SetNumber)
	s.Equal(we.ID, logged.WorkoutExerciseID)
	s.InEpsilon(100, logged.WeightKg, 1e-6)
	s.True(logged.IsWarmup)

	set, err = domainworkout.NewWorkoutSet(we.ID, 105, 2, nil, nil, false)
	s.Require().NoError(err)
	logged, err = s.repo.LogSet(s.ctx, we.ID, set)
	s.Require().NoError(err)
	s.Equal(4, logged.SetNumber)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.Require().Len(found.Exercises[0].Sets, 4)
	s.Equal(4, found.Exercises[0].Sets[3].SetNumber)
}

func (s *WorkoutRepoTestSuite) TestUpdate_HeaderOnly() {
	w := s.createTestWorkout(s.newTestWorkout(uuid.New()))

	w.Notes = "updated notes"
	finishedAt := time.Now()
	w.FinishedAt = &finishedAt
	updated, err := s.updateTestWorkout(w)
	s.Require().NoError(err)
	s.Equal(2, updated.Version)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.Equal("updated notes", found.Notes)
	s.NotNil(found.FinishedAt)
	s.Equal(2, found.Version)
	s.Require().Len(found.Exercises, 2)
}

func (s *WorkoutRepoTestSuite) TestUpdate_VersionConflict() {
	w := s.createTestWorkout(s.newTestWorkout(uuid.New()))

	w.Notes = "first update"
	_, err := s.updateTestWorkout(w)
	s.Require().NoError(err)

	w.Version = 1
	_, err = s.updateTestWorkout(w)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *WorkoutRepoTestSuite) TestUpdate_NotFound() {
	w, err := domainworkout.NewWorkout(uuid.New(), nil)
	s.Require().NoError(err)
	_, err = s.updateTestWorkout(w)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *WorkoutRepoTestSuite) TestFinish_SetsFinishedAtAndReturnsChildren() {
	w := s.createTestWorkout(s.newTestWorkout(uuid.New()))

	finishedAt := time.Now()
	finished, err := s.repo.Finish(s.ctx, w.ID, finishedAt)
	s.Require().NoError(err)
	s.NotNil(finished.FinishedAt)
	s.Equal(2, finished.Version)
	s.Require().Len(finished.Exercises, 2)
	s.Require().Len(finished.Exercises[0].Sets, 2)

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.NotNil(found.FinishedAt)
	s.Equal(2, found.Version)
}

func (s *WorkoutRepoTestSuite) TestFinish_NotFound() {
	_, err := s.repo.Finish(s.ctx, uuid.New(), time.Now())
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)
}

func (s *WorkoutRepoTestSuite) TestSoftDelete_RemovesFromList() {
	userID := uuid.New()
	w := s.createTestWorkout(s.newTestWorkout(userID))

	err := s.repo.SoftDelete(s.ctx, w.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, domainworkout.WorkoutFilter{UserID: &userID})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(w.ID, l.ID)
	}
}

func TestWorkoutRepoSuite(t *testing.T) {
	suite.Run(t, new(WorkoutRepoTestSuite))
}
