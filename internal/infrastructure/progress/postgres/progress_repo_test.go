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

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/progress/postgres"
)

type ProgressRepoTestSuite struct {
	suite.Suite

	pool         *pgxpool.Pool
	repo         *postgres.ProgressRepo
	trManager    *manager.Manager
	ctx          context.Context
	exerciseID   uuid.UUID
	oneRMEntries []domainprogress.Progress1RM
	volumeValues []domainprogress.ProgressVolume
}

func (s *ProgressRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewProgressRepo(pool)
	s.trManager = manager.Must(trmpgx.NewDefaultFactory(pool))
	s.ctx = context.Background()

	exerciseID := uuid.Must(uuid.NewV7())
	_, err = pool.Exec(s.ctx, `
		INSERT INTO gym.exercises (id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, version)
		VALUES ($1, $2, '', '', false, $3, now(), now(), 1)
	`, exerciseID, "progress-test-"+uuid.NewString(), uuid.New())
	s.Require().NoError(err)
	s.exerciseID = exerciseID
}

func (s *ProgressRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		for _, p := range s.oneRMEntries {
			_, _ = s.pool.Exec(context.Background(), `
				DELETE FROM gym.progress_1rm WHERE exercise_id = $1 AND user_id = $2 AND date = $3
			`, p.ExerciseID, p.UserID, p.Date)
		}
		for _, v := range s.volumeValues {
			_, _ = s.pool.Exec(context.Background(), `
				DELETE FROM gym.progress_volume WHERE muscle_group_id = $1 AND user_id = $2 AND date = $3
			`, v.MuscleGroupID, v.UserID, v.Date)
		}
		_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.exercises WHERE id = $1`, s.exerciseID)
		s.pool.Close()
	}
}

func (s *ProgressRepoTestSuite) upsertTest1RM(userID uuid.UUID, date time.Time, estimated1RM float64) {
	p, err := domainprogress.NewProgress1RM(s.exerciseID, userID, date, estimated1RM)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Upsert1RM(s.ctx, p))
	s.oneRMEntries = append(s.oneRMEntries, p)
}

func (s *ProgressRepoTestSuite) insertTestVolume(
	muscleGroupID int,
	userID uuid.UUID,
	date time.Time,
	totalKG float64,
) {
	v, err := domainprogress.NewProgressVolume(muscleGroupID, userID, date, totalKG)
	s.Require().NoError(err)
	_, err = s.pool.Exec(s.ctx, `
		INSERT INTO gym.progress_volume (muscle_group_id, user_id, date, total_kg)
		VALUES ($1, $2, $3, $4)
	`, v.MuscleGroupID, v.UserID, v.Date, v.TotalKG)
	s.Require().NoError(err)
	s.volumeValues = append(s.volumeValues, v)
}

func (s *ProgressRepoTestSuite) TestUpsert1RM_InsertThenUpdate() {
	userID := uuid.New()
	date := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	s.upsertTest1RM(userID, date, 100)
	s.upsertTest1RM(userID, date, 120)

	rows, err := s.repo.Get1RM(s.ctx, s.exerciseID, userID, time.Time{}, time.Time{})
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal(date, rows[0].Date)
	s.Equal(120.0, rows[0].Estimated1RM)

	best, err := s.repo.GetBest1RM(s.ctx, s.exerciseID, userID)
	s.Require().NoError(err)
	s.Equal(120.0, best)
}

func (s *ProgressRepoTestSuite) TestGet1RM_NoRange() {
	userID := uuid.New()
	d1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	s.upsertTest1RM(userID, d1, 100)
	s.upsertTest1RM(userID, d2, 110)
	s.upsertTest1RM(userID, d3, 120)

	rows, err := s.repo.Get1RM(s.ctx, s.exerciseID, userID, time.Time{}, time.Time{})
	s.Require().NoError(err)
	s.Require().Len(rows, 3)
	s.Equal(d1, rows[0].Date)
	s.Equal(d2, rows[1].Date)
	s.Equal(d3, rows[2].Date)
}

func (s *ProgressRepoTestSuite) TestGet1RM_WithRange() {
	userID := uuid.New()
	d1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	s.upsertTest1RM(userID, d1, 100)
	s.upsertTest1RM(userID, d2, 110)
	s.upsertTest1RM(userID, d3, 120)

	from := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	rows, err := s.repo.Get1RM(s.ctx, s.exerciseID, userID, from, to)
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal(d2, rows[0].Date)
	s.Equal(110.0, rows[0].Estimated1RM)

	rows, err = s.repo.Get1RM(s.ctx, s.exerciseID, userID, from, time.Time{})
	s.Require().NoError(err)
	s.Require().Len(rows, 2)

	rows, err = s.repo.Get1RM(s.ctx, s.exerciseID, userID, time.Time{}, to)
	s.Require().NoError(err)
	s.Require().Len(rows, 2)
}

func (s *ProgressRepoTestSuite) TestGet1RM_Empty() {
	rows, err := s.repo.Get1RM(s.ctx, s.exerciseID, uuid.New(), time.Time{}, time.Time{})
	s.Require().NoError(err)
	s.Empty(rows)
}

func (s *ProgressRepoTestSuite) TestGetBest1RM_NoRowsReturnsZero() {
	best, err := s.repo.GetBest1RM(s.ctx, s.exerciseID, uuid.New())
	s.Require().NoError(err)
	s.Equal(0.0, best)
}

func (s *ProgressRepoTestSuite) TestGetBest1RM_MaxAcrossUpserts() {
	userID := uuid.New()
	d1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	s.upsertTest1RM(userID, d1, 100)
	s.upsertTest1RM(userID, d2, 150)
	s.upsertTest1RM(userID, d3, 120)

	best, err := s.repo.GetBest1RM(s.ctx, s.exerciseID, userID)
	s.Require().NoError(err)
	s.Equal(150.0, best)
}

func (s *ProgressRepoTestSuite) TestGetBest1RM_DoesNotMixUsers() {
	userID := uuid.New()
	s.upsertTest1RM(userID, time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), 140)

	best, err := s.repo.GetBest1RM(s.ctx, s.exerciseID, uuid.New())
	s.Require().NoError(err)
	s.Equal(0.0, best)
}

func (s *ProgressRepoTestSuite) TestGetVolume_NoRange() {
	userID := uuid.New()
	d1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	s.insertTestVolume(1, userID, d1, 1000)
	s.insertTestVolume(2, userID, d2, 2000)

	volumes, err := s.repo.GetVolume(s.ctx, userID, time.Time{}, time.Time{})
	s.Require().NoError(err)
	s.Require().Len(volumes, 2)
	s.Equal(d1, volumes[0].Date)
	s.Equal(1000.0, volumes[0].TotalKG)
	s.Equal(1, volumes[0].MuscleGroupID)
	s.Equal(d2, volumes[1].Date)
	s.Equal(2000.0, volumes[1].TotalKG)
}

func (s *ProgressRepoTestSuite) TestGetVolume_WithRange() {
	userID := uuid.New()
	d1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	s.insertTestVolume(1, userID, d1, 1000)
	s.insertTestVolume(1, userID, d2, 1100)
	s.insertTestVolume(1, userID, d3, 1200)

	from := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	volumes, err := s.repo.GetVolume(s.ctx, userID, from, to)
	s.Require().NoError(err)
	s.Require().Len(volumes, 1)
	s.Equal(d2, volumes[0].Date)
	s.Equal(1100.0, volumes[0].TotalKG)
}

func (s *ProgressRepoTestSuite) TestGetVolume_Empty() {
	volumes, err := s.repo.GetVolume(s.ctx, uuid.New(), time.Time{}, time.Time{})
	s.Require().NoError(err)
	s.Empty(volumes)
}

func TestProgressRepoSuite(t *testing.T) {
	suite.Run(t, new(ProgressRepoTestSuite))
}
