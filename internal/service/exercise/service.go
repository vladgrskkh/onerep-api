package exercise

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

type ExerciseRepository interface {
	List(ctx context.Context, filter domainexercise.ExerciseFilter) ([]*domainexercise.Exercise, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domainexercise.Exercise, error)
	Create(ctx context.Context, ex domainexercise.Exercise) (*domainexercise.Exercise, error)
	Update(ctx context.Context, ex domainexercise.Exercise) (*domainexercise.Exercise, error)
	ReplaceMedia(ctx context.Context, exerciseID uuid.UUID, media []domainexercise.ExerciseMedia) error
	ReplaceMuscleGroups(
		ctx context.Context,
		exerciseID uuid.UUID,
		groups []domainexercise.ExerciseMuscleGroup,
	) error
	BatchInsertMuscleGroups(
		ctx context.Context,
		exerciseID uuid.UUID,
		groups []domainexercise.ExerciseMuscleGroup,
	) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type MuscleGroupRepository interface {
	List(ctx context.Context) ([]*domainexercise.MuscleGroup, error)
}

// TransactionManager runs a function inside a transaction.
type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type ExerciseService struct {
	exercises    ExerciseRepository
	muscleGroups MuscleGroupRepository
	trManager    TransactionManager
}

func NewExerciseService(
	exercises ExerciseRepository,
	muscleGroups MuscleGroupRepository,
	trManager TransactionManager,
) *ExerciseService {
	return &ExerciseService{
		exercises:    exercises,
		muscleGroups: muscleGroups,
		trManager:    trManager,
	}
}

// List returns exercises matching the command filters.
func (s *ExerciseService) List(ctx context.Context, cmd ListExercisesCommand) ([]*domainexercise.Exercise, error) {
	return s.exercises.List(ctx, cmd.Filter())
}

// Get returns a single exercise by ID, including its media and muscle groups.
func (s *ExerciseService) Get(ctx context.Context, id uuid.UUID) (*domainexercise.Exercise, error) {
	return s.exercises.FindByID(ctx, id)
}

// Create creates an exercise and attaches its muscle groups to it.
func (s *ExerciseService) Create(ctx context.Context, cmd CreateExerciseCommand) (*domainexercise.Exercise, error) {
	ex, err := domainexercise.NewExercise(cmd.Name, cmd.Description, cmd.Notes, cmd.UserID)
	if err != nil {
		return nil, err
	}

	groups, err := s.resolveMuscleGroups(ctx, cmd.MuscleGroupIDs)
	if err != nil {
		return nil, err
	}

	var created *domainexercise.Exercise
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		created, err = s.exercises.Create(ctx, ex)
		if err != nil {
			return err
		}
		if err = s.exercises.BatchInsertMuscleGroups(ctx, created.ID, groups); err != nil {
			return err
		}
		created.MuscleGroups = groups
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Update applies the non-nil fields of the command to an existing exercise
// and replaces its muscle groups when IDs are provided.
func (s *ExerciseService) Update(ctx context.Context, cmd UpdateExerciseCommand) (*domainexercise.Exercise, error) {
	ex, err := s.exercises.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if ex.IsBuiltIn {
		return nil, domainexercise.ErrCannotEditBuiltIn
	}

	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return nil, domainexercise.ErrInvalidName
		}
		ex.Name = name
	}
	if cmd.Description != nil {
		ex.Description = *cmd.Description
	}
	if cmd.Notes != nil {
		ex.Notes = *cmd.Notes
	}

	var groups []domainexercise.ExerciseMuscleGroup
	if cmd.MuscleGroupIDs != nil {
		groups, err = s.resolveMuscleGroups(ctx, *cmd.MuscleGroupIDs)
		if err != nil {
			return nil, err
		}
	}

	ex.UpdatedAt = time.Now()

	var updated *domainexercise.Exercise
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		updated, err = s.exercises.Update(ctx, *ex)
		if err != nil {
			return err
		}
		if cmd.MuscleGroupIDs != nil {
			if err = s.exercises.ReplaceMuscleGroups(ctx, ex.ID, groups); err != nil {
				return err
			}
			updated.MuscleGroups = groups
		} else {
			updated.MuscleGroups = ex.MuscleGroups
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SoftDelete marks an exercise as deleted. Built-in exercises are read-only.
func (s *ExerciseService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	ex, err := s.exercises.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if ex.IsBuiltIn {
		return domainexercise.ErrCannotEditBuiltIn
	}
	return s.exercises.SoftDelete(ctx, id)
}

// resolveMuscleGroups validates the given muscle group IDs and builds the
// join rows for an exercise. Unknown IDs are rejected so that invalid
// requests fail before touching the database.
func (s *ExerciseService) resolveMuscleGroups(
	ctx context.Context,
	ids []int,
) ([]domainexercise.ExerciseMuscleGroup, error) {
	groups := make([]domainexercise.ExerciseMuscleGroup, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, domainexercise.ErrInvalidID
		}
		groups = append(groups, domainexercise.ExerciseMuscleGroup{MuscleGroupID: id})
	}
	if len(ids) == 0 {
		return groups, nil
	}

	existing, err := s.muscleGroups.List(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[int]struct{}, len(existing))
	for _, g := range existing {
		known[g.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := known[id]; !ok {
			return nil, domainexercise.ErrMuscleGroupMissing
		}
	}
	return groups, nil
}
