package workout

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

const (
	// epleyDivisor is the rep denominator of the Epley 1RM formula.
	epleyDivisor = 30.0
	// oneDecimal scales an e1rm value for rounding to one decimal place.
	oneDecimal = 10.0
)

type WorkoutRepository interface {
	List(ctx context.Context, filter domainworkout.WorkoutFilter) ([]*domainworkout.Workout, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domainworkout.Workout, error)
	Create(ctx context.Context, w domainworkout.Workout) (*domainworkout.Workout, error)
	AddExercise(ctx context.Context, workoutID, exerciseID uuid.UUID) (*domainworkout.WorkoutExercise, error)
	BatchInsertExercises(ctx context.Context, exercises []domainworkout.WorkoutExercise) error
	LogSet(
		ctx context.Context,
		workoutExerciseID uuid.UUID,
		set domainworkout.WorkoutSet,
	) (*domainworkout.WorkoutSet, error)
	Update(ctx context.Context, w domainworkout.Workout) (*domainworkout.Workout, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*domainworkout.Workout, error)
}

// TemplateRepository reads templates for starting a workout from a template.
type TemplateRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domaintemplate.Template, error)
}

// ProgressRepository reads and writes the user's 1RM progress for the
// per-set PR check.
type ProgressRepository interface {
	GetBest1RM(ctx context.Context, exerciseID, userID uuid.UUID) (float64, error)
	Upsert1RM(ctx context.Context, p domainprogress.Progress1RM) error
}

// VolumeQueue enqueues the background volume recalculation of a finished
// workout.
type VolumeQueue interface {
	EnqueueVolumeCalc(ctx context.Context, workout domainworkout.Workout) error
}

// TransactionManager runs a function inside a transaction.
type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type WorkoutService struct {
	workouts    WorkoutRepository
	templates   TemplateRepository
	progress    ProgressRepository
	volumeQueue VolumeQueue
	trManager   TransactionManager
}

func NewWorkoutService(
	workouts WorkoutRepository,
	templates TemplateRepository,
	progress ProgressRepository,
	volumeQueue VolumeQueue,
	trManager TransactionManager,
) *WorkoutService {
	return &WorkoutService{
		workouts:    workouts,
		templates:   templates,
		progress:    progress,
		volumeQueue: volumeQueue,
		trManager:   trManager,
	}
}

// Start begins a new workout for the user. When a template is provided, its
// exercises are copied into the workout; a zero TemplateID starts a workout
// without one.
func (s *WorkoutService) Start(ctx context.Context, cmd StartWorkoutCommand) (*domainworkout.Workout, error) {
	workout, err := domainworkout.NewWorkout(cmd.UserID, cmd.TemplateID)
	if err != nil {
		return nil, err
	}

	active, err := s.findActive(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, domainworkout.ErrActiveWorkout
	}

	var exercises []domainworkout.WorkoutExercise
	if cmd.TemplateID != uuid.Nil {
		exercises, err = s.copyTemplateExercises(ctx, workout.ID, cmd.TemplateID, cmd.UserID)
		if err != nil {
			return nil, err
		}
	}

	var created *domainworkout.Workout
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		created, err = s.workouts.Create(ctx, workout)
		if err != nil {
			return err
		}
		if err = s.workouts.BatchInsertExercises(ctx, exercises); err != nil {
			return err
		}
		created.Exercises = exercises
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// GetActive returns the user's active (unfinished) workout.
func (s *WorkoutService) GetActive(ctx context.Context, userID uuid.UUID) (*domainworkout.Workout, error) {
	workouts, err := s.workouts.List(ctx, domainworkout.WorkoutFilter{UserID: &userID})
	if err != nil {
		return nil, err
	}
	for _, w := range workouts {
		if w.FinishedAt == nil {
			return w, nil
		}
	}
	return nil, domainworkout.ErrWorkoutNotFound
}

// Get returns the user's workout by ID, including its exercises and sets.
func (s *WorkoutService) Get(ctx context.Context, id, userID uuid.UUID) (*domainworkout.Workout, error) {
	return s.ownedWorkout(ctx, id, userID)
}

// List returns the user's workouts, optionally only those updated after
// since.
func (s *WorkoutService) List(
	ctx context.Context,
	userID uuid.UUID,
	since time.Time,
) ([]*domainworkout.Workout, error) {
	if userID == uuid.Nil {
		return nil, domainworkout.ErrInvalidUserID
	}
	return s.workouts.List(ctx, domainworkout.WorkoutFilter{UserID: &userID, Since: since})
}

// AddExercise appends an exercise to the user's workout.
func (s *WorkoutService) AddExercise(
	ctx context.Context,
	cmd AddExerciseCommand,
	userID uuid.UUID,
) (*domainworkout.WorkoutExercise, error) {
	if _, err := s.ownedWorkout(ctx, cmd.WorkoutID, userID); err != nil {
		return nil, err
	}
	return s.workouts.AddExercise(ctx, cmd.WorkoutID, cmd.ExerciseID)
}

// LogSet records a set, computes its estimated one-rep max with the Epley
// formula, and upserts a new best in the user's 1RM progress when the set is
// a PR.
func (s *WorkoutService) LogSet(
	ctx context.Context,
	cmd LogSetCommand,
	userID uuid.UUID,
) (*domainworkout.SetResult, error) {
	workout, err := s.ownedWorkout(ctx, cmd.WorkoutID, userID)
	if err != nil {
		return nil, err
	}

	exerciseID, err := findExerciseID(workout, cmd.WorkoutExerciseID)
	if err != nil {
		return nil, err
	}

	var (
		rpe         *int
		restSeconds *int
	)
	if cmd.RPE != 0 {
		rpe = &cmd.RPE
	}
	if cmd.RestSeconds != 0 {
		restSeconds = &cmd.RestSeconds
	}
	set, err := domainworkout.NewWorkoutSet(
		cmd.WorkoutExerciseID, cmd.WeightKg, cmd.Reps, rpe, restSeconds, cmd.IsWarmup,
	)
	if err != nil {
		return nil, err
	}

	e1rm := estimate1RM(set.WeightKg, set.Reps)

	var (
		saved *domainworkout.WorkoutSet
		isPR  bool
	)
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		saved, err = s.workouts.LogSet(ctx, cmd.WorkoutExerciseID, set)
		if err != nil {
			return err
		}
		var prevBest float64
		prevBest, err = s.progress.GetBest1RM(ctx, exerciseID, userID)
		if err != nil {
			return err
		}
		isPR = prevBest == 0 || e1rm > prevBest
		if isPR {
			var p domainprogress.Progress1RM
			p, err = domainprogress.NewProgress1RM(exerciseID, userID, time.Now(), e1rm)
			if err != nil {
				return err
			}
			if err = s.progress.Upsert1RM(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &domainworkout.SetResult{WorkoutSet: *saved, IsPR: isPR, Estimated1RM: e1rm}, nil
}

// Finish completes the user's workout and enqueues its volume
// recalculation.
func (s *WorkoutService) Finish(
	ctx context.Context,
	cmd FinishWorkoutCommand,
	userID uuid.UUID,
) (*domainworkout.Workout, error) {
	if _, err := s.ownedWorkout(ctx, cmd.WorkoutID, userID); err != nil {
		return nil, err
	}

	workout, err := s.workouts.Finish(ctx, cmd.WorkoutID, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.volumeQueue.EnqueueVolumeCalc(ctx, *workout); err != nil {
		return nil, fmt.Errorf("enqueue volume calc: %w", err)
	}
	return workout, nil
}

// ownedWorkout loads a workout and enforces ownership. Non-owners see
// ErrWorkoutNotFound so that the workout's existence is not leaked.
func (s *WorkoutService) ownedWorkout(ctx context.Context, id, userID uuid.UUID) (*domainworkout.Workout, error) {
	w, err := s.workouts.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.UserID != userID {
		return nil, domainworkout.ErrWorkoutNotFound
	}
	return w, nil
}

// findActive reports whether the user already has an unfinished workout.
func (s *WorkoutService) findActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	workouts, err := s.workouts.List(ctx, domainworkout.WorkoutFilter{UserID: &userID})
	if err != nil {
		return false, err
	}
	for _, w := range workouts {
		if w.FinishedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

// copyTemplateExercises loads a template and converts its exercises into
// workout exercises for the new workout. Templates that are neither public
// nor owned by the user are treated as not found so their existence is not
// leaked.
func (s *WorkoutService) copyTemplateExercises(
	ctx context.Context,
	workoutID, templateID, userID uuid.UUID,
) ([]domainworkout.WorkoutExercise, error) {
	t, err := s.templates.FindByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if !t.IsPublic && t.CreatedByUserID != userID {
		return nil, domaintemplate.ErrTemplateNotFound
	}

	exercises := make([]domainworkout.WorkoutExercise, 0, len(t.Exercises))
	for _, e := range t.Exercises {
		exercises = append(exercises, domainworkout.WorkoutExercise{
			ID:         uuid.Must(uuid.NewV7()),
			WorkoutID:  workoutID,
			ExerciseID: e.ExerciseID,
			SortOrder:  e.SortOrder,
			Notes:      "",
		})
	}
	return exercises, nil
}

// findExerciseID returns the exercise backing the given workout exercise.
func findExerciseID(workout *domainworkout.Workout, workoutExerciseID uuid.UUID) (uuid.UUID, error) {
	for _, we := range workout.Exercises {
		if we.ID == workoutExerciseID {
			return we.ExerciseID, nil
		}
	}
	return uuid.Nil, domainworkout.ErrWorkoutExerciseNotFound
}

// estimate1RM computes the estimated one-rep max with the Epley formula,
// rounded to one decimal place.
func estimate1RM(weightKg float64, reps int) float64 {
	e1rm := weightKg * (1 + float64(reps)/epleyDivisor)
	return math.Round(e1rm*oneDecimal) / oneDecimal
}
