package postgres

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
)

type BodyWeightRepo struct {
	pool *pgxpool.Pool
}

func NewBodyWeightRepo(pool *pgxpool.Pool) *BodyWeightRepo {
	return &BodyWeightRepo{pool: pool}
}

func (r *BodyWeightRepo) List(
	ctx context.Context,
	userID uuid.UUID,
	since *time.Time,
) ([]domainbodyweight.BodyWeight, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, user_id, weight_kg, measured_at, created_at, updated_at, version FROM gym.body_weights WHERE user_id = $1`,
	)
	args := []any{userID}

	if since != nil {
		args = append(args, *since)
		query.WriteString(" AND updated_at > $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY measured_at DESC")

	rows, err := r.pool.Query(ctx, query.String(), args...)
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
	_, err := r.pool.Exec(ctx, `
		INSERT INTO gym.body_weights (id, user_id, weight_kg, measured_at, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, bw.ID, bw.UserID, bw.WeightKg, bw.MeasuredAt, bw.CreatedAt, bw.UpdatedAt, bw.Version)
	if err != nil {
		return domainbodyweight.BodyWeight{}, err
	}
	return bw, nil
}
