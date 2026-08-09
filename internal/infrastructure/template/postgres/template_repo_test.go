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

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/template/postgres"
)

type TemplateRepoTestSuite struct {
	suite.Suite

	pool         *pgxpool.Pool
	repo         *postgres.TemplateRepo
	trManager    *manager.Manager
	ctx          context.Context
	created      []uuid.UUID
	exercisesIDs []uuid.UUID
}

func (s *TemplateRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewTemplateRepo(pool)
	s.trManager = manager.Must(trmpgx.NewDefaultFactory(pool))
	s.ctx = context.Background()
}

func (s *TemplateRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		for _, id := range s.created {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.templates WHERE id = $1`, id)
		}
		for _, id := range s.exercisesIDs {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.exercises WHERE id = $1`, id)
		}
		s.pool.Close()
	}
}

func (s *TemplateRepoTestSuite) insertTestExercise() uuid.UUID {
	id := uuid.Must(uuid.NewV7())
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO gym.exercises (id, name, description, created_by_user_id, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, now(), now(), 1)
	`, id, "template-test-exercise-"+uuid.NewString(), "Fixture exercise", uuid.New())
	s.Require().NoError(err)
	s.exercisesIDs = append(s.exercisesIDs, id)
	return id
}

func (s *TemplateRepoTestSuite) newTestTemplate(name string) domaintemplate.Template {
	t, err := domaintemplate.NewTemplate(name, "Description", uuid.New())
	s.Require().NoError(err)
	ex1 := s.insertTestExercise()
	ex2 := s.insertTestExercise()
	t.Exercises = []domaintemplate.TemplateExercise{
		{TemplateID: t.ID, ExerciseID: ex1, SortOrder: 0, PlannedSets: 3},
		{TemplateID: t.ID, ExerciseID: ex2, SortOrder: 1, PlannedSets: 4},
	}
	t.Media = []domaintemplate.TemplateMedia{
		{ID: uuid.Must(uuid.NewV7()), TemplateID: t.ID, MediaType: domaintemplate.MediaTypePhoto, SortOrder: 0, S3Key: "templates/test-photo.jpg"},
	}
	return t
}

func (s *TemplateRepoTestSuite) createTestTemplate(t domaintemplate.Template) domaintemplate.Template {
	var created domaintemplate.Template
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, t)
		if err != nil {
			return err
		}
		if err = s.repo.ReplaceExercises(ctx, created.ID, t.Exercises); err != nil {
			return err
		}
		return s.repo.ReplaceMedia(ctx, created.ID, t.Media)
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)
	return created
}

func (s *TemplateRepoTestSuite) updateTestTemplate(t domaintemplate.Template) (domaintemplate.Template, error) {
	var updated domaintemplate.Template
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		updated, err = s.repo.Update(ctx, t)
		return err
	})
	return updated, err
}

func (s *TemplateRepoTestSuite) TestCreateAndFindByID() {
	t := s.newTestTemplate("push-day-" + uuid.NewString())
	created := s.createTestTemplate(t)
	s.Equal(t.ID, created.ID)

	found, err := s.repo.FindByID(s.ctx, t.ID)
	s.Require().NoError(err)
	s.Equal(t.Name, found.Name)
	s.Equal("Description", found.Description)
	s.False(found.IsPublic)
	s.NotNil(found.CreatedByUserID)
	s.Equal(1, found.Version)
	s.Len(found.Exercises, 2)
	s.Equal(0, found.Exercises[0].SortOrder)
	s.Equal(3, found.Exercises[0].PlannedSets)
	s.Equal(1, found.Exercises[1].SortOrder)
	s.Equal(4, found.Exercises[1].PlannedSets)
	s.Len(found.Media, 1)
	s.Equal(domaintemplate.MediaTypePhoto, found.Media[0].MediaType)
	s.Equal(0, found.Media[0].SortOrder)
	s.Equal("templates/test-photo.jpg", found.Media[0].S3Key)
}

func (s *TemplateRepoTestSuite) TestCreate_HeaderOnly() {
	t := s.newTestTemplate("HeaderOnly-" + uuid.NewString())
	var created domaintemplate.Template
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, t)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(t.Name, found.Name)
	s.Empty(found.Exercises)
	s.Empty(found.Media)
}

func (s *TemplateRepoTestSuite) TestBatchInsertExercises_AndMedia() {
	t := s.newTestTemplate("BatchInsert-" + uuid.NewString())
	t.Exercises = nil
	t.Media = nil
	var created domaintemplate.Template
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, t)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)

	ex1 := s.insertTestExercise()
	ex2 := s.insertTestExercise()
	exercises := []domaintemplate.TemplateExercise{
		{TemplateID: created.ID, ExerciseID: ex1, SortOrder: 0, PlannedSets: 3},
		{TemplateID: created.ID, ExerciseID: ex2, SortOrder: 1, PlannedSets: 4},
	}
	media := []domaintemplate.TemplateMedia{
		{ID: uuid.Must(uuid.NewV7()), TemplateID: created.ID, MediaType: domaintemplate.MediaTypePhoto, SortOrder: 0, S3Key: "templates/batch-1.jpg"},
		{ID: uuid.Must(uuid.NewV7()), TemplateID: created.ID, MediaType: domaintemplate.MediaTypePhoto, SortOrder: 1, S3Key: "templates/batch-2.jpg"},
	}

	err = s.trManager.Do(s.ctx, func(ctx context.Context) error {
		if err := s.repo.BatchInsertExercises(ctx, exercises); err != nil {
			return err
		}
		return s.repo.BatchInsertMedia(ctx, media)
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Len(found.Exercises, 2)
	s.Equal(0, found.Exercises[0].SortOrder)
	s.Equal(3, found.Exercises[0].PlannedSets)
	s.Equal(ex1, found.Exercises[0].ExerciseID)
	s.Equal(1, found.Exercises[1].SortOrder)
	s.Equal(4, found.Exercises[1].PlannedSets)
	s.Equal(ex2, found.Exercises[1].ExerciseID)
	s.Len(found.Media, 2)
	s.Equal("templates/batch-1.jpg", found.Media[0].S3Key)
	s.Equal("templates/batch-2.jpg", found.Media[1].S3Key)
}

func (s *TemplateRepoTestSuite) TestInsertExercise_AndMedia_Single() {
	t := s.newTestTemplate("SingleInsert-" + uuid.NewString())
	t.Exercises = nil
	t.Media = nil
	created := s.createTestTemplate(t)

	ex := s.insertTestExercise()
	err := s.repo.InsertExercise(s.ctx, domaintemplate.TemplateExercise{
		TemplateID:  created.ID,
		ExerciseID:  ex,
		SortOrder:   0,
		PlannedSets: 5,
	})
	s.Require().NoError(err)
	err = s.repo.InsertMedia(s.ctx, domaintemplate.TemplateMedia{
		ID:         uuid.Must(uuid.NewV7()),
		TemplateID: created.ID,
		MediaType:  domaintemplate.MediaTypePhoto,
		SortOrder:  0,
		S3Key:      "templates/single.jpg",
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Len(found.Exercises, 1)
	s.Equal(5, found.Exercises[0].PlannedSets)
	s.Len(found.Media, 1)
	s.Equal("templates/single.jpg", found.Media[0].S3Key)
}

func (s *TemplateRepoTestSuite) TestFindByID_NotFound() {
	_, err := s.repo.FindByID(s.ctx, uuid.New())
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestList_UserIDFilter() {
	userID := uuid.New()
	t1 := s.newTestTemplate("UserListA-" + uuid.NewString())
	t1.CreatedByUserID = userID
	s.createTestTemplate(t1)
	t2 := s.newTestTemplate("UserListB-" + uuid.NewString())
	t2.CreatedByUserID = uuid.New()
	s.createTestTemplate(t2)

	listed, err := s.repo.List(s.ctx, domaintemplate.TemplateFilter{UserID: &userID})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t1.ID, listed[0].ID)
	s.Empty(listed[0].Exercises)
	s.Empty(listed[0].Media)
}

func (s *TemplateRepoTestSuite) TestList_IsPublicFilter() {
	userID1 := uuid.New()
	userID2 := uuid.New()
	t1 := s.newTestTemplate("PublicTemplate-" + uuid.NewString())
	t1.IsPublic = true
	t1.CreatedByUserID = userID1
	s.createTestTemplate(t1)
	t2 := s.newTestTemplate("PrivateTemplate-" + uuid.NewString())
	t2.CreatedByUserID = userID2
	s.createTestTemplate(t2)

	isPublic := true
	listed, err := s.repo.List(s.ctx, domaintemplate.TemplateFilter{UserID: &userID1, IsPublic: &isPublic})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t1.ID, listed[0].ID)

	isPublic = false
	listed, err = s.repo.List(s.ctx, domaintemplate.TemplateFilter{UserID: &userID2, IsPublic: &isPublic})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t2.ID, listed[0].ID)
}

func (s *TemplateRepoTestSuite) TestList_SinceFilter() {
	userID := uuid.New()
	t := s.newTestTemplate("SinceTemplate-" + uuid.NewString())
	t.CreatedByUserID = userID
	t.CreatedAt = time.Now().Add(-2 * time.Hour)
	t.UpdatedAt = time.Now().Add(-2 * time.Hour)
	s.createTestTemplate(t)

	since := time.Now().Add(-3 * time.Hour)
	listed, err := s.repo.List(s.ctx, domaintemplate.TemplateFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t.ID, listed[0].ID)

	since = time.Now().Add(-time.Hour)
	listed, err = s.repo.List(s.ctx, domaintemplate.TemplateFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *TemplateRepoTestSuite) TestList_ExcludesSoftDeleted() {
	t := s.createTestTemplate(s.newTestTemplate("SoftDeletedTemplate-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, t.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, domaintemplate.TemplateFilter{})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(t.ID, l.ID)
	}

	_, err = s.repo.FindByID(s.ctx, t.ID)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestUpdate_HeaderKeepsChildren() {
	t := s.createTestTemplate(s.newTestTemplate("UpdateKeepsChildren-" + uuid.NewString()))

	t.Name = "Updated-" + uuid.NewString()
	t.IsPublic = true

	updated, err := s.updateTestTemplate(t)
	s.Require().NoError(err)
	s.Equal(2, updated.Version)

	found, err := s.repo.FindByID(s.ctx, t.ID)
	s.Require().NoError(err)
	s.Equal(t.Name, found.Name)
	s.True(found.IsPublic)
	s.Equal(2, found.Version)
	s.Len(found.Exercises, 2)
	s.Len(found.Media, 1)
}

func (s *TemplateRepoTestSuite) TestUpdate_ReplacesChildren() {
	t := s.createTestTemplate(s.newTestTemplate("UpdateChildren-" + uuid.NewString()))

	t.Name = "Updated-" + uuid.NewString()
	t.IsPublic = true
	ex := s.insertTestExercise()
	t.Exercises = []domaintemplate.TemplateExercise{
		{TemplateID: t.ID, ExerciseID: ex, SortOrder: 0, PlannedSets: 5},
	}
	t.Media = []domaintemplate.TemplateMedia{
		{ID: uuid.Must(uuid.NewV7()), TemplateID: t.ID, MediaType: domaintemplate.MediaTypePhoto, SortOrder: 0, S3Key: "templates/replacement.jpg"},
	}

	var updated domaintemplate.Template
	err := s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		updated, err = s.repo.Update(ctx, t)
		if err != nil {
			return err
		}
		if err = s.repo.ReplaceExercises(ctx, t.ID, t.Exercises); err != nil {
			return err
		}
		return s.repo.ReplaceMedia(ctx, t.ID, t.Media)
	})
	s.Require().NoError(err)
	s.Equal(2, updated.Version)

	found, err := s.repo.FindByID(s.ctx, t.ID)
	s.Require().NoError(err)
	s.Equal(t.Name, found.Name)
	s.True(found.IsPublic)
	s.Equal(2, found.Version)
	s.Len(found.Exercises, 1)
	s.Equal(5, found.Exercises[0].PlannedSets)
	s.Equal(ex, found.Exercises[0].ExerciseID)
	s.Len(found.Media, 1)
	s.Equal("templates/replacement.jpg", found.Media[0].S3Key)
}

func (s *TemplateRepoTestSuite) TestUpdate_VersionConflict() {
	t := s.createTestTemplate(s.newTestTemplate("VersionConflict-" + uuid.NewString()))

	t.Name = "first update"
	_, err := s.updateTestTemplate(t)
	s.Require().NoError(err)

	t.Version = 1
	_, err = s.updateTestTemplate(t)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestUpdate_NotFound() {
	t, err := domaintemplate.NewTemplate("NotFound-"+uuid.NewString(), "Description", uuid.New())
	s.Require().NoError(err)
	_, err = s.updateTestTemplate(t)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestSoftDelete_RemovesFromList() {
	t := s.createTestTemplate(s.newTestTemplate("DeleteMe-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, t.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, domaintemplate.TemplateFilter{})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(t.ID, l.ID)
	}
}

func TestTemplateRepoSuite(t *testing.T) {
	suite.Run(t, new(TemplateRepoTestSuite))
}
