package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
)

const (
	argExerciseID = "exercise_id"
	argUserID     = "user_id"
)

type ProgressRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewProgressRepo(pool *pgxpool.Pool) *ProgressRepo {
	return &ProgressRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *ProgressRepo) Get1RM(
	ctx context.Context,
	exerciseID, userID uuid.UUID,
	from, to time.Time,
) ([]*domainprogress.Progress1RM, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT exercise_id, user_id, date, estimated_1rm FROM gym.progress_1rm WHERE exercise_id = @exercise_id AND user_id = @user_id`,
	)
	args := pgx.NamedArgs{argExerciseID: exerciseID, argUserID: userID}

	if !from.IsZero() {
		query.WriteString(` AND date >= @from`)
		args["from"] = from
	}
	if !to.IsZero() {
		query.WriteString(` AND date <= @to`)
		args["to"] = to
	}
	query.WriteString(` ORDER BY date ASC`)

	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query.String(), args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progress []*domainprogress.Progress1RM
	for rows.Next() {
		p := &domainprogress.Progress1RM{}
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
	from, to time.Time,
) ([]*domainprogress.ProgressVolume, error) {
	var query strings.Builder
	query.WriteString(
		`SELECT muscle_group_id, user_id, date, total_kg FROM gym.progress_volume WHERE user_id = @user_id`,
	)
	args := pgx.NamedArgs{argUserID: userID}

	if !from.IsZero() {
		query.WriteString(` AND date >= @from`)
		args["from"] = from
	}
	if !to.IsZero() {
		query.WriteString(` AND date <= @to`)
		args["to"] = to
	}
	query.WriteString(` ORDER BY date ASC`)

	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query.String(), args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var volumes []*domainprogress.ProgressVolume
	for rows.Next() {
		v := &domainprogress.ProgressVolume{}
		if scanErr := rows.Scan(&v.MuscleGroupID, &v.UserID, &v.Date, &v.TotalKG); scanErr != nil {
			return nil, scanErr
		}
		volumes = append(volumes, v)
	}
	return volumes, rows.Err()
}

func (r *ProgressRepo) GetBest1RM(ctx context.Context, exerciseID, userID uuid.UUID) (float64, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	var best float64
	err := conn.QueryRow(ctx, `
		SELECT COALESCE(MAX(estimated_1rm), 0) FROM gym.progress_1rm WHERE exercise_id = @exercise_id AND user_id = @user_id
	`, pgx.NamedArgs{argExerciseID: exerciseID, argUserID: userID}).Scan(&best)
	if err != nil {
		return 0, err
	}
	return best, nil
}

func (r *ProgressRepo) Upsert1RM(ctx context.Context, p domainprogress.Progress1RM) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		INSERT INTO gym.progress_1rm (exercise_id, user_id, date, estimated_1rm)
		VALUES (@exercise_id, @user_id, @date, @estimated_1rm)
		ON CONFLICT (exercise_id, user_id, date) DO UPDATE SET estimated_1rm = EXCLUDED.estimated_1rm
	`, pgx.NamedArgs{
		argExerciseID:   p.ExerciseID,
		argUserID:       p.UserID,
		"date":          p.Date,
		"estimated_1rm": p.Estimated1RM,
	})
	return err
}
