package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

type MuscleGroupRepo struct {
	pool *pgxpool.Pool
}

func NewMuscleGroupRepo(pool *pgxpool.Pool) *MuscleGroupRepo {
	return &MuscleGroupRepo{pool: pool}
}

func (r *MuscleGroupRepo) List(ctx context.Context) ([]exercise.MuscleGroup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name FROM gym.muscle_groups ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []exercise.MuscleGroup
	for rows.Next() {
		var g exercise.MuscleGroup
		if scanErr := rows.Scan(&g.ID, &g.Name); scanErr != nil {
			return nil, scanErr
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
