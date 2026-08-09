//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/gym/postgres"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

type TemplateRepoTestSuite struct {
	suite.Suite

	pool         *pgxpool.Pool
	repo         *postgres.TemplateRepo
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
	created, err := s.repo.Create(s.ctx, t)
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)
	return created
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

	listed, err := s.repo.List(s.ctx, servicetemplate.TemplateFilter{UserID: &userID})
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
	listed, err := s.repo.List(s.ctx, servicetemplate.TemplateFilter{UserID: &userID1, IsPublic: &isPublic})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t1.ID, listed[0].ID)

	isPublic = false
	listed, err = s.repo.List(s.ctx, servicetemplate.TemplateFilter{UserID: &userID2, IsPublic: &isPublic})
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
	listed, err := s.repo.List(s.ctx, servicetemplate.TemplateFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(t.ID, listed[0].ID)

	since = time.Now().Add(-time.Hour)
	listed, err = s.repo.List(s.ctx, servicetemplate.TemplateFilter{UserID: &userID, Since: &since})
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *TemplateRepoTestSuite) TestList_ExcludesSoftDeleted() {
	t := s.createTestTemplate(s.newTestTemplate("SoftDeletedTemplate-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, t.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, servicetemplate.TemplateFilter{})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(t.ID, l.ID)
	}

	_, err = s.repo.FindByID(s.ctx, t.ID)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
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

	updated, err := s.repo.Update(s.ctx, t)
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
	_, err := s.repo.Update(s.ctx, t)
	s.Require().NoError(err)

	t.Version = 1
	_, err = s.repo.Update(s.ctx, t)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestUpdate_NotFound() {
	t, err := domaintemplate.NewTemplate("NotFound-"+uuid.NewString(), "Description", uuid.New())
	s.Require().NoError(err)
	_, err = s.repo.Update(s.ctx, t)
	s.ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *TemplateRepoTestSuite) TestSoftDelete_RemovesFromList() {
	t := s.createTestTemplate(s.newTestTemplate("DeleteMe-" + uuid.NewString()))

	err := s.repo.SoftDelete(s.ctx, t.ID)
	s.Require().NoError(err)

	listed, err := s.repo.List(s.ctx, servicetemplate.TemplateFilter{})
	s.Require().NoError(err)
	for _, l := range listed {
		s.NotEqual(t.ID, l.ID)
	}
}

func TestTemplateRepoSuite(t *testing.T) {
	suite.Run(t, new(TemplateRepoTestSuite))
}
