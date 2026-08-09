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

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/bodyweight/postgres"
)

type BodyWeightRepoTestSuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      *postgres.BodyWeightRepo
	trManager *manager.Manager
	ctx       context.Context
	created   []uuid.UUID
}

func (s *BodyWeightRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewBodyWeightRepo(pool)
	s.trManager = manager.Must(trmpgx.NewDefaultFactory(pool))
	s.ctx = context.Background()
}

func (s *BodyWeightRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		for _, id := range s.created {
			_, _ = s.pool.Exec(context.Background(), `DELETE FROM gym.body_weights WHERE id = $1`, id)
		}
		s.pool.Close()
	}
}

func (s *BodyWeightRepoTestSuite) createTestBodyWeight(
	userID uuid.UUID,
	weightKg float64,
	measuredAt time.Time,
) domainbodyweight.BodyWeight {
	bw, err := domainbodyweight.NewBodyWeight(userID, weightKg, measuredAt)
	s.Require().NoError(err)
	bw.UpdatedAt = measuredAt

	var created domainbodyweight.BodyWeight
	err = s.trManager.Do(s.ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, bw)
		return err
	})
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)
	return created
}

func (s *BodyWeightRepoTestSuite) TestCreateAndList_RoundTrip() {
	userID := uuid.New()

	older := s.createTestBodyWeight(userID, 80.5, time.Now().Add(-48*time.Hour))
	newer := s.createTestBodyWeight(userID, 81.2, time.Now().Add(-24*time.Hour))

	listed, err := s.repo.List(s.ctx, userID, nil)
	s.Require().NoError(err)
	s.Require().Len(listed, 2)
	s.Equal(newer.ID, listed[0].ID)
	s.Equal(older.ID, listed[1].ID)
	s.Equal(userID, listed[0].UserID)
	s.Equal(newer.WeightKg, listed[0].WeightKg)
	s.Equal(newer.MeasuredAt.Unix(), listed[0].MeasuredAt.Unix())
	s.Equal(1, listed[0].Version)
}

func (s *BodyWeightRepoTestSuite) TestList_Empty() {
	listed, err := s.repo.List(s.ctx, uuid.New(), nil)
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *BodyWeightRepoTestSuite) TestList_SinceFilter() {
	userID := uuid.New()
	s.createTestBodyWeight(userID, 80.5, time.Now().Add(-48*time.Hour))

	since := time.Now().Add(-72 * time.Hour)
	listed, err := s.repo.List(s.ctx, userID, &since)
	s.Require().NoError(err)
	s.Require().Len(listed, 1)

	since = time.Now().Add(-time.Hour)
	listed, err = s.repo.List(s.ctx, userID, &since)
	s.Require().NoError(err)
	s.Empty(listed)
}

func (s *BodyWeightRepoTestSuite) TestList_DoesNotMixUsers() {
	userA := uuid.New()
	userB := uuid.New()
	s.createTestBodyWeight(userA, 80.5, time.Now().Add(-24*time.Hour))
	s.createTestBodyWeight(userB, 90.0, time.Now().Add(-24*time.Hour))

	listed, err := s.repo.List(s.ctx, userA, nil)
	s.Require().NoError(err)
	s.Require().Len(listed, 1)
	s.Equal(userA, listed[0].UserID)
}

func (s *BodyWeightRepoTestSuite) TestCreate_PersistsFields() {
	userID := uuid.New()
	bw, err := domainbodyweight.NewBodyWeight(userID, 79.9, time.Now().Add(-time.Hour))
	s.Require().NoError(err)

	created, err := s.repo.Create(s.ctx, bw)
	s.Require().NoError(err)
	s.created = append(s.created, created.ID)

	s.Equal(bw.ID, created.ID)
	s.Equal(bw.UserID, created.UserID)
	s.Equal(bw.WeightKg, created.WeightKg)
	s.Equal(1, created.Version)
	s.False(created.CreatedAt.IsZero())
	s.False(created.UpdatedAt.IsZero())
}

func TestBodyWeightRepoSuite(t *testing.T) {
	suite.Run(t, new(BodyWeightRepoTestSuite))
}
