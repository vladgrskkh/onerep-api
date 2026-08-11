//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/exercise/postgres"
)

type MuscleGroupRepoTestSuite struct {
	suite.Suite

	pool *pgxpool.Pool
	repo *postgres.MuscleGroupRepo
	ctx  context.Context
}

func (s *MuscleGroupRepoTestSuite) SetupTest() {
	pool, err := pgxpool.New(context.Background(), "postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable")
	if err != nil {
		panic(err)
	}
	s.pool = pool
	s.repo = postgres.NewMuscleGroupRepo(pool)
	s.ctx = context.Background()
}

func (s *MuscleGroupRepoTestSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *MuscleGroupRepoTestSuite) TestList_ReturnsAllSeededGroups() {
	groups, err := s.repo.List(s.ctx)
	s.Require().NoError(err)
	s.Len(groups, 15)

	expected := []string{
		"chest", "quads", "biceps", "triceps", "shoulders",
		"back", "glutes", "hamstrings", "calves", "abs",
		"forearms", "traps", "lats", "adductors", "obliques",
	}
	for i, g := range groups {
		s.Equal(i+1, g.ID)
		s.Equal(expected[i], g.Name)
		s.Equal(&exercise.MuscleGroup{ID: i + 1, Name: expected[i]}, g)
	}
}

func TestMuscleGroupRepoSuite(t *testing.T) {
	suite.Run(t, new(MuscleGroupRepoTestSuite))
}
