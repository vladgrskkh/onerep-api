package postgres

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
)

type ProgressRepo struct {
	pool *pgxpool.Pool
}

func NewProgressRepo(pool *pgxpool.Pool) *ProgressRepo {
	return &ProgressRepo{pool: pool}
}

func (r *ProgressRepo) Get1RM(
	ctx context.Context,
	exerciseID, userID uuid.UUID,
	from, to *time.Time,
) ([]domainprogress.Progress1RM, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT exercise_id, user_id, date, estimated_1rm FROM gym.progress_1rm WHERE exercise_id = $1 AND user_id = $2`,
	)
	args := []any{exerciseID, userID}

	if from != nil {
		args = append(args, *from)
		query.WriteString(" AND date >= $" + strconv.Itoa(len(args)))
	}
	if to != nil {
		args = append(args, *to)
		query.WriteString(" AND date <= $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY date ASC")

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progress []domainprogress.Progress1RM
	for rows.Next() {
		var p domainprogress.Progress1RM
		if scanErr := rows.Scan(&p.ExerciseID, &p.UserID, &p.Date, &p.Estimated1RM); scanErr != nil {
			return nil, scanErr
		}
		progress = append(progress, p)
	}
	return progress, rows.Err()
}

func (r *ProgressRepo) GetVolume(
	ctx context.Context,
	userID uuid.UUID,
	from, to *time.Time,
) ([]domainprogress.ProgressVolume, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT muscle_group_id, user_id, date, total_kg FROM gym.progress_volume WHERE user_id = $1`,
	)
	args := []any{userID}

	if from != nil {
		args = append(args, *from)
		query.WriteString(" AND date >= $" + strconv.Itoa(len(args)))
	}
	if to != nil {
		args = append(args, *to)
		query.WriteString(" AND date <= $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY date ASC")

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var volumes []domainprogress.ProgressVolume
	for rows.Next() {
		var v domainprogress.ProgressVolume
		if scanErr := rows.Scan(&v.MuscleGroupID, &v.UserID, &v.Date, &v.TotalKG); scanErr != nil {
			return nil, scanErr
		}
		volumes = append(volumes, v)
	}
	return volumes, rows.Err()
}

func (r *ProgressRepo) GetBest1RM(ctx context.Context, exerciseID, userID uuid.UUID) (float64, error) {
	var best float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(estimated_1rm), 0) FROM gym.progress_1rm WHERE exercise_id = $1 AND user_id = $2
	`, exerciseID, userID).Scan(&best)
	if err != nil {
		return 0, err
	}
	return best, nil
}

func (r *ProgressRepo) Upsert1RM(ctx context.Context, p domainprogress.Progress1RM) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO gym.progress_1rm (exercise_id, user_id, date, estimated_1rm)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (exercise_id, user_id, date) DO UPDATE SET estimated_1rm = EXCLUDED.estimated_1rm
	`, p.ExerciseID, p.UserID, p.Date, p.Estimated1RM)
	return err
}
