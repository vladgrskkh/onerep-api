package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
)

type BodyWeightRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewBodyWeightRepo(pool *pgxpool.Pool) *BodyWeightRepo {
	return &BodyWeightRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *BodyWeightRepo) List(
	ctx context.Context,
	userID uuid.UUID,
	since *time.Time,
) ([]domainbodyweight.BodyWeight, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, user_id, weight_kg, measured_at, created_at, updated_at, version FROM gym.body_weights WHERE user_id = @user_id`,
	)
	args := pgx.NamedArgs{"user_id": userID}

	if since != nil {
		query.WriteString(` AND updated_at > @since`)
		args["since"] = *since
	}
	query.WriteString(` ORDER BY measured_at DESC`)

	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query.String(), args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weights []domainbodyweight.BodyWeight
	for rows.Next() {
		var bw domainbodyweight.BodyWeight
		if scanErr := rows.Scan(
			&bw.ID,
			&bw.UserID,
			&bw.WeightKg,
			&bw.MeasuredAt,
			&bw.CreatedAt,
			&bw.UpdatedAt,
			&bw.Version,
		); scanErr != nil {
			return nil, scanErr
		}
		weights = append(weights, bw)
	}
	return weights, rows.Err()
}

func (r *BodyWeightRepo) Create(
	ctx context.Context,
	bw domainbodyweight.BodyWeight,
) (domainbodyweight.BodyWeight, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		INSERT INTO gym.body_weights (id, user_id, weight_kg, measured_at, created_at, updated_at, version)
		VALUES (@id, @user_id, @weight_kg, @measured_at, @created_at, @updated_at, @version)
	`, pgx.NamedArgs{
		"id":          bw.ID,
		"user_id":     bw.UserID,
		"weight_kg":   bw.WeightKg,
		"measured_at": bw.MeasuredAt,
		"created_at":  bw.CreatedAt,
		"updated_at":  bw.UpdatedAt,
		"version":     bw.Version,
	}); err != nil {
		return domainbodyweight.BodyWeight{}, err
	}
	return bw, nil
}
