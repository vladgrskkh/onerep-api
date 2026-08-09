//go:build integration

package postgres_test

import (
	"context"
	"errors"
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
		if err != nil {
			return err
		}
		created.Exercises = nil
		for _, we := range w.Exercises {
			createdWE, err := s.repo.AddExercise(ctx, created.ID, we.ExerciseID)
			if err != nil {
				return err
			}
			for _, set := range we.Sets {
				createdSet, err := s.repo.LogSet(ctx, createdWE.ID, set)
				if err != nil {
					return err
				}
				createdWE.Sets = append(createdWE.Sets, createdSet)
			}
			created.Exercises = append(created.Exercises, createdWE)
		}
		return nil
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

func (s *WorkoutRepoTestSuite) TestCreate_HeaderOnly() {
	userID := uuid.New()
	w, err := domainworkout.NewWorkout(userID, nil)
	s.Require().NoError(err)
	var created domainworkout.Workout
	err = s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, w)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(w.UserID, found.UserID)
	s.Empty(found.Exercises)
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

func (s *WorkoutRepoTestSuite) insertWorkoutExercise(id, workoutID, exerciseID uuid.UUID, sortOrder int) {
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO gym.workout_exercises (id, workout_id, exercise_id, sort_order, notes)
		VALUES ($1, $2, $3, $4, '')
	`, id, workoutID, exerciseID, sortOrder)
	s.Require().NoError(err)
}

func (s *WorkoutRepoTestSuite) TestFindByID_AttributesSetsWithEqualSortOrder() {
	w, err := domainworkout.NewWorkout(uuid.New(), nil)
	s.Require().NoError(err)
	s.createTestWorkout(w)
	ex1 := s.insertTestExercise()
	ex2 := s.insertTestExercise()

	we1, err := domainworkout.NewWorkoutExercise(w.ID, ex1)
	s.Require().NoError(err)
	we1.SortOrder = 0
	s.insertWorkoutExercise(we1.ID, w.ID, ex1, 0)
	set1, err := domainworkout.NewWorkoutSet(we1.ID, 60, 10, nil, nil, false)
	s.Require().NoError(err)
	set1.SetNumber = 1
	set2, err := domainworkout.NewWorkoutSet(we1.ID, 62, 8, nil, nil, false)
	s.Require().NoError(err)
	set2.SetNumber = 2
	we1.Sets = []domainworkout.WorkoutSet{set1, set2}

	we2, err := domainworkout.NewWorkoutExercise(w.ID, ex2)
	s.Require().NoError(err)
	we2.SortOrder = 0
	s.insertWorkoutExercise(we2.ID, w.ID, ex2, 0)
	set3, err := domainworkout.NewWorkoutSet(we2.ID, 100, 5, nil, nil, false)
	s.Require().NoError(err)
	set3.SetNumber = 1
	set4, err := domainworkout.NewWorkoutSet(we2.ID, 102, 3, nil, nil, false)
	s.Require().NoError(err)
	set4.SetNumber = 2
	we2.Sets = []domainworkout.WorkoutSet{set3, set4}

	for _, we := range []domainworkout.WorkoutExercise{we1, we2} {
		for _, set := range we.Sets {
			_, err = s.repo.LogSet(s.ctx, we.ID, set)
			s.Require().NoError(err)
		}
	}

	found, err := s.repo.FindByID(s.ctx, w.ID)
	s.Require().NoError(err)
	s.Require().Len(found.Exercises, 2)

	byID := make(map[uuid.UUID]domainworkout.WorkoutExercise, len(found.Exercises))
	for _, we := range found.Exercises {
		byID[we.ID] = we
	}
	s.Equal(0, byID[we1.ID].SortOrder)
	s.Equal(0, byID[we2.ID].SortOrder)

	s.Require().Len(byID[we1.ID].Sets, 2)
	s.Equal(we1.ID, byID[we1.ID].Sets[0].WorkoutExerciseID)
	s.Equal(set1.ID, byID[we1.ID].Sets[0].ID)
	s.Equal(set2.ID, byID[we1.ID].Sets[1].ID)

	s.Require().Len(byID[we2.ID].Sets, 2)
	s.Equal(we2.ID, byID[we2.ID].Sets[0].WorkoutExerciseID)
	s.Equal(set3.ID, byID[we2.ID].Sets[0].ID)
	s.Equal(set4.ID, byID[we2.ID].Sets[1].ID)
}

func (s *WorkoutRepoTestSuite) TestCreate_RollsBackOnError() {
	w := s.newTestWorkout(uuid.New())
	w.Notes = "rollback-me"

	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		created, err := s.repo.Create(ctx, w)
		s.Require().NoError(err)
		s.created = append(s.created, created.ID)
		return errors.New("boom")
	})
	s.Require().Error(err)

	_, err = s.repo.FindByID(s.ctx, w.ID)
	s.ErrorIs(err, domainworkout.ErrWorkoutNotFound)

	var count int
	err = s.pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM gym.workouts WHERE id = $1`, w.ID).Scan(&count)
	s.Require().NoError(err)
	s.Zero(count)
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

func (s *WorkoutRepoTestSuite) TestBatchInsertExercises_AndSets() {
	userID := uuid.New()
	w, err := domainworkout.NewWorkout(userID, nil)
	s.Require().NoError(err)
	created := s.createTestWorkout(w)

	ex1 := s.insertTestExercise()
	ex2 := s.insertTestExercise()
	we1, err := domainworkout.NewWorkoutExercise(created.ID, ex1)
	s.Require().NoError(err)
	we1.SortOrder = 0
	we2, err := domainworkout.NewWorkoutExercise(created.ID, ex2)
	s.Require().NoError(err)
	we2.SortOrder = 1
	exercises := []domainworkout.WorkoutExercise{we1, we2}

	rpe := 8
	rest := 90
	set1, err := domainworkout.NewWorkoutSet(we1.ID, 80, 5, nil, &rest, false)
	s.Require().NoError(err)
	set1.SetNumber = 1
	set2, err := domainworkout.NewWorkoutSet(we1.ID, 82.5, 3, &rpe, nil, false)
	s.Require().NoError(err)
	set2.SetNumber = 2
	sets := []domainworkout.WorkoutSet{set1, set2}

	err = s.trManager.Do(s.ctx, func(ctx context.Context) error {
		if err := s.repo.BatchInsertExercises(ctx, exercises); err != nil {
			return err
		}
		return s.repo.BatchInsertSets(ctx, sets)
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Require().Len(found.Exercises, 2)
	s.Equal(0, found.Exercises[0].SortOrder)
	s.Equal(ex1, found.Exercises[0].ExerciseID)
	s.Equal(1, found.Exercises[1].SortOrder)
	s.Equal(ex2, found.Exercises[1].ExerciseID)

	s.Require().Len(found.Exercises[0].Sets, 2)
	s.Equal(1, found.Exercises[0].Sets[0].SetNumber)
	s.InEpsilon(80, found.Exercises[0].Sets[0].WeightKg, 1e-6)
	s.Equal(5, found.Exercises[0].Sets[0].Reps)
	s.Equal(90, *found.Exercises[0].Sets[0].RestSeconds)
	s.Equal(2, found.Exercises[0].Sets[1].SetNumber)
	s.Equal(8, *found.Exercises[0].Sets[1].RPE)
	s.Empty(found.Exercises[1].Sets)
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
