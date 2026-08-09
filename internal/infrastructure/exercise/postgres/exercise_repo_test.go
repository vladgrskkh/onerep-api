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

	"github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/exercise/postgres"
)

type ExerciseRepoTestSuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      *postgres.ExerciseRepo
	trManager *manager.Manager
	ctx       context.Context
	created   []uuid.UUID
}

func (s *ExerciseRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewExerciseRepo(pool)
	s.trManager = manager.Must(trmpgx.NewDefaultFactory(pool))
	s.ctx = context.Background()
}

func (s *ExerciseRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		for _, id := range s.created {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.exercises WHERE id = $1`, id)
		}
		s.pool.Close()
	}
}

func (s *ExerciseRepoTestSuite) newTestExercise(name string) exercise.Exercise {
	ex, err := exercise.NewExercise(name, "Description", "Notes", uuid.New())
	s.Require().NoError(err)
	ex.Media = []exercise.ExerciseMedia{
		{ID: uuid.Must(uuid.NewV7()), ExerciseID: ex.ID, MediaType: exercise.MediaTypeVideo, SortOrder: 0, S3Key: "exercises/test-video.mp4"},
		{ID: uuid.Must(uuid.NewV7()), ExerciseID: ex.ID, MediaType: exercise.MediaTypePhoto, SortOrder: 1, S3Key: "exercises/test-photo.jpg"},
	}
	ex.MuscleGroups = []exercise.ExerciseMuscleGroup{
		{ExerciseID: ex.ID, MuscleGroupID: 1, IsPrimary: false},
		{ExerciseID: ex.ID, MuscleGroupID: 2, IsPrimary: true},
	}
	return ex
}

func (s *ExerciseRepoTestSuite) createTestExercise(ex exercise.Exercise) exercise.Exercise {
	var created exercise.Exercise
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, ex)
		if err != nil {
			return err
		}
		if err = s.repo.ReplaceMedia(ctx, created.ID, ex.Media); err != nil {
			return err
		}
		return s.repo.ReplaceMuscleGroups(ctx, created.ID, ex.MuscleGroups)
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)
	return created
}

func (s *ExerciseRepoTestSuite) updateTestExercise(ex exercise.Exercise) (exercise.Exercise, error) {
	var updated exercise.Exercise
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		updated, err = s.repo.Update(ctx, ex)
		return err
	})
	return updated, err
}

func (s *ExerciseRepoTestSuite) TestCreateAndFindByID() {
	ex := s.newTestExercise("bench-press-" + uuid.NewString())
	created := s.createTestExercise(ex)
	s.Equal(ex.ID, created.ID)

	found, err := s.repo.FindByID(s.ctx, ex.ID)
	s.Require().NoError(err)
	s.Equal(ex.Name, found.Name)
	s.Equal("Description", found.Description)
	s.Equal("Notes", found.Notes)
	s.False(found.IsBuiltIn)
	s.NotNil(found.CreatedByUserID)
	s.Equal(1, found.Version)
	s.Len(found.Media, 2)
	s.Equal(exercise.MediaTypeVideo, found.Media[0].MediaType)
	s.Equal(0, found.Media[0].SortOrder)
	s.Equal(exercise.MediaTypePhoto, found.Media[1].MediaType)
	s.Equal(1, found.Media[1].SortOrder)
	s.Len(found.MuscleGroups, 2)
	s.True(found.MuscleGroups[0].IsPrimary)
	s.Equal(2, found.MuscleGroups[0].MuscleGroupID)
	s.False(found.MuscleGroups[1].IsPrimary)
	s.Equal(1, found.MuscleGroups[1].MuscleGroupID)
}

func (s *ExerciseRepoTestSuite) TestCreate_HeaderOnly() {
	ex := s.newTestExercise("HeaderOnly-" + uuid.NewString())
	var created exercise.Exercise
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, ex)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(ex.Name, found.Name)
	s.Empty(found.Media)
	s.Empty(found.MuscleGroups)
}

func (s *ExerciseRepoTestSuite) TestBatchInsertMedia_AndMuscleGroups() {
	ex := s.newTestExercise("BatchInsert-" + uuid.NewString())
	ex.Media = nil
	ex.MuscleGroups = nil
	created := s.createTestExercise(ex)

	media := []exercise.ExerciseMedia{
		{ID: uuid.Must(uuid.NewV7()), ExerciseID: created.ID, MediaType: exercise.MediaTypePhoto, SortOrder: 0, S3Key: "exercises/batch-1.jpg"},
		{ID: uuid.Must(uuid.NewV7()), ExerciseID: created.ID, MediaType: exercise.MediaTypeVideo, SortOrder: 1, S3Key: "exercises/batch-2.mp4"},
	}
	groups := []exercise.ExerciseMuscleGroup{
		{ExerciseID: created.ID, MuscleGroupID: 1, IsPrimary: false},
		{ExerciseID: created.ID, MuscleGroupID: 2, IsPrimary: true},
	}

	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		if err := s.repo.BatchInsertMedia(ctx, media); err != nil {
			return err
		}
		return s.repo.BatchInsertMuscleGroups(ctx, groups)
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Len(found.Media, 2)
	s.Equal("exercises/batch-1.jpg", found.Media[0].S3Key)
	s.Equal("exercises/batch-2.mp4", found.Media[1].S3Key)
	s.Len(found.MuscleGroups, 2)
	s.True(found.MuscleGroups[0].IsPrimary)
	s.Equal(2, found.MuscleGroups[0].MuscleGroupID)
	s.False(found.MuscleGroups[1].IsPrimary)
	s.Equal(1, found.MuscleGroups[1].MuscleGroupID)
}

func (s *ExerciseRepoTestSuite) TestInsertMedia_AndMuscleGroup_Single() {
	ex := s.newTestExercise("SingleInsert-" + uuid.NewString())
	ex.Media = nil
	ex.MuscleGroups = nil
	created := s.createTestExercise(ex)

	err := s.repo.InsertMedia(s.ctx, exercise.ExerciseMedia{
		ID:         uuid.Must(uuid.NewV7()),
		ExerciseID: created.ID,
		MediaType:  exercise.MediaTypePhoto,
		SortOrder:  0,
		S3Key:      "exercises/single.jpg",
	})
	s.Require().NoError(err)
	err = s.repo.InsertMuscleGroup(s.ctx, exercise.ExerciseMuscleGroup{
		ExerciseID:    created.ID,
		MuscleGroupID: 3,
		IsPrimary:     true,
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Len(found.Media, 1)
	s.Equal("exercises/single.jpg", found.Media[0].S3Key)
	s.Len(found.MuscleGroups, 1)
	s.Equal(3, found.MuscleGroups[0].MuscleGroupID)
	s.True(found.MuscleGroups[0].IsPrimary)
}

func (s *ExerciseRepoTestSuite) TestFindByID_NotFound() {
	_, err := s.repo.FindByID(s.ctx, uuid.New())
	s.ErrorIs(err, exercise.ErrExerciseNotFound)
}

func (s *ExerciseRepoTestSuite) TestList_Search() {
	ex := s.createTestExercise(s.newTestExercise("SearchableName-" + uuid.NewString()))
	s.createTestExercise(s.newTestExercise("UnrelatedName-" + uuid.NewString()))

	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(ex.ID, listed[0].ID)
}

func (s *ExerciseRepoTestSuite) TestList_MuscleGroupFilter() {
	ex := s.newTestExercise("QuadsExercise-" + uuid.NewString())
	ex.MuscleGroups = []exercise.ExerciseMuscleGroup{{ExerciseID: ex.ID, MuscleGroupID: 2, IsPrimary: true}}
	s.createTestExercise(ex)
	s.createTestExercise(s.newTestExercise("ChestExercise-" + uuid.NewString()))

	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(ex.ID, listed[0].ID)

	listed, err = s.repo.List(s.ctx, exercise.ExerciseFilter{MuscleGroup: "quads"})
	s.Require().NoError(err)
	s.True(len(listed) >= 1)
	found := false
	for _, l := range listed {
		if l.ID == ex.ID {
			found = true
		}
	}
	s.True(found)
}

func (s *ExerciseRepoTestSuite) TestList_SinceFilter() {
	ex := s.newTestExercise("SinceFilter-" + uuid.NewString())
	ex.UpdatedAt = time.Now().Add(-2 * time.Hour)
	ex.CreatedAt = time.Now().Add(-2 * time.Hour)
	s.createTestExercise(ex)

	since := time.Now().Add(-3 * time.Hour)
	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name, Since: &since})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(ex.ID, listed[0].ID)

	since = time.Now().Add(-time.Hour)
	listed, err = s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name, Since: &since})
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *ExerciseRepoTestSuite) TestList_IsBuiltInFilter() {
	ex := s.newTestExercise("BuiltInExercise-" + uuid.NewString())
	ex.IsBuiltIn = true
	s.createTestExercise(ex)

	isBuiltIn := true
	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name, IsBuiltIn: &isBuiltIn})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(ex.ID, listed[0].ID)

	isBuiltIn = false
	listed, err = s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name, IsBuiltIn: &isBuiltIn})
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *ExerciseRepoTestSuite) TestList_ExcludesSoftDeleted() {
	ex := s.createTestExercise(s.newTestExercise("SoftDeleted-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, ex.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name})
	s.Require().NoError(err)
	s.Empty(listed)

	_, err = s.repo.FindByID(s.ctx, ex.ID)
	s.ErrorIs(err, exercise.ErrExerciseNotFound)
}

func (s *ExerciseRepoTestSuite) TestUpdate_HeaderKeepsChildren() {
	ex := s.createTestExercise(s.newTestExercise("UpdateKeepsChildren-" + uuid.NewString()))

	ex.Name = "Updated-" + uuid.NewString()
	ex.Notes = "Updated notes"

	updated, err := s.updateTestExercise(ex)
	s.Require().NoError(err)
	s.Equal(2, updated.Version)

	found, err := s.repo.FindByID(s.ctx, ex.ID)
	s.Require().NoError(err)
	s.Equal(ex.Name, found.Name)
	s.Equal("Updated notes", found.Notes)
	s.Equal(2, found.Version)
	s.Len(found.Media, 2)
	s.Len(found.MuscleGroups, 2)
}

func (s *ExerciseRepoTestSuite) TestUpdate_ReplacesChildren() {
	ex := s.createTestExercise(s.newTestExercise("UpdateChildren-" + uuid.NewString()))

	ex.Name = "Updated-" + uuid.NewString()
	ex.Notes = "Updated notes"
	ex.Media = []exercise.ExerciseMedia{
		{ID: uuid.Must(uuid.NewV7()), ExerciseID: ex.ID, MediaType: exercise.MediaTypePhoto, SortOrder: 0, S3Key: "exercises/replacement.jpg"},
	}
	ex.MuscleGroups = []exercise.ExerciseMuscleGroup{
		{ExerciseID: ex.ID, MuscleGroupID: 3, IsPrimary: true},
	}

	var updated exercise.Exercise
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		updated, err = s.repo.Update(ctx, ex)
		if err != nil {
			return err
		}
		if err = s.repo.ReplaceMedia(ctx, ex.ID, ex.Media); err != nil {
			return err
		}
		return s.repo.ReplaceMuscleGroups(ctx, ex.ID, ex.MuscleGroups)
	})
	s.Require().NoError(err)
	s.Equal(2, updated.Version)

	found, err := s.repo.FindByID(s.ctx, ex.ID)
	s.Require().NoError(err)
	s.Equal(ex.Name, found.Name)
	s.Equal("Updated notes", found.Notes)
	s.Equal(2, found.Version)
	s.Len(found.Media, 1)
	s.Equal("exercises/replacement.jpg", found.Media[0].S3Key)
	s.Len(found.MuscleGroups, 1)
	s.Equal(3, found.MuscleGroups[0].MuscleGroupID)
}

func (s *ExerciseRepoTestSuite) TestUpdate_VersionConflict() {
	ex := s.createTestExercise(s.newTestExercise("VersionConflict-" + uuid.NewString()))

	ex.Notes = "first update"
	_, err := s.updateTestExercise(ex)
	s.Require().NoError(err)

	ex.Version = 1
	_, err = s.updateTestExercise(ex)
	s.ErrorIs(err, exercise.ErrExerciseNotFound)
}

func (s *ExerciseRepoTestSuite) TestUpdate_NotFound() {
	ex, err := exercise.NewExercise("NotFound-"+uuid.NewString(), "Description", "Notes", uuid.New())
	s.Require().NoError(err)
	_, err = s.updateTestExercise(ex)
	s.ErrorIs(err, exercise.ErrExerciseNotFound)
}

func (s *ExerciseRepoTestSuite) TestSoftDelete_RemovesFromList() {
	ex := s.createTestExercise(s.newTestExercise("DeleteMe-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, ex.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, exercise.ExerciseFilter{Search: ex.Name})
	s.Require().NoError(err)
	s.Empty(listed)
}

func TestExerciseRepoSuite(t *testing.T) {
	suite.Run(t, new(ExerciseRepoTestSuite))
}
