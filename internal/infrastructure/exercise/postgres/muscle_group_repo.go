package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	"github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

type MuscleGroupRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewMuscleGroupRepo(pool *pgxpool.Pool) *MuscleGroupRepo {
	return &MuscleGroupRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *MuscleGroupRepo) List(ctx context.Context) ([]exercise.MuscleGroup, error) {
	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, `
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
