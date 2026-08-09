package template

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
)

type TemplateRepository interface {
	List(ctx context.Context, filter domaintemplate.TemplateFilter) ([]domaintemplate.Template, error)
	FindByID(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error)
	Create(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	Update(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	ReplaceExercises(
		ctx context.Context,
		templateID uuid.UUID,
		exercises []domaintemplate.TemplateExercise,
	) error
	ReplaceMedia(ctx context.Context, templateID uuid.UUID, media []domaintemplate.TemplateMedia) error
	BatchInsertExercises(ctx context.Context, exercises []domaintemplate.TemplateExercise) error
	BatchInsertMedia(ctx context.Context, media []domaintemplate.TemplateMedia) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// TransactionManager runs a function inside a transaction.
type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type TemplateService struct {
	templates TemplateRepository
	trManager TransactionManager
}

func NewTemplateService(
	templates TemplateRepository,
	trManager TransactionManager,
) *TemplateService {
	return &TemplateService{
		templates: templates,
		trManager: trManager,
	}
}

// List returns templates matching the filter.
func (s *TemplateService) List(
	ctx context.Context,
	filter domaintemplate.TemplateFilter,
) ([]domaintemplate.Template, error) {
	return s.templates.List(ctx, filter)
}

// Get returns a single template by ID, including its exercises and media.
func (s *TemplateService) Get(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error) {
	return s.templates.FindByID(ctx, id)
}

// Create creates a template and inserts its exercises in one transaction.
func (s *TemplateService) Create(ctx context.Context, cmd CreateTemplateCommand) (domaintemplate.Template, error) {
	t, err := domaintemplate.NewTemplate(cmd.Name, cmd.Description, cmd.UserID)
	if err != nil {
		return domaintemplate.Template{}, err
	}

	exercises := buildExercises(t.ID, cmd.Exercises)

	var created domaintemplate.Template
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		created, err = s.templates.Create(ctx, t)
		if err != nil {
			return err
		}
		if err = s.templates.BatchInsertExercises(ctx, exercises); err != nil {
			return err
		}
		created.Exercises = exercises
		return nil
	})
	if err != nil {
		return domaintemplate.Template{}, err
	}
	return created, nil
}

// Update applies the non-nil fields of the command to an existing template
// and replaces its exercises when provided, all in one transaction. Only the
// owner may update a template.
func (s *TemplateService) Update(ctx context.Context, cmd UpdateTemplateCommand) (domaintemplate.Template, error) {
	t, err := s.templates.FindByID(ctx, cmd.ID)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	if t.CreatedByUserID != cmd.UserID {
		return domaintemplate.Template{}, domaintemplate.ErrNotOwner
	}

	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return domaintemplate.Template{}, domaintemplate.ErrInvalidName
		}
		t.Name = name
	}
	if cmd.Description != nil {
		t.Description = *cmd.Description
	}

	var exercises []domaintemplate.TemplateExercise
	if cmd.Exercises != nil {
		exercises = buildExercises(t.ID, *cmd.Exercises)
	}

	t.UpdatedAt = time.Now()

	var updated domaintemplate.Template
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		updated, err = s.templates.Update(ctx, t)
		if err != nil {
			return err
		}
		if cmd.Exercises != nil {
			if err = s.templates.ReplaceExercises(ctx, t.ID, exercises); err != nil {
				return err
			}
			updated.Exercises = exercises
		} else {
			updated.Exercises = t.Exercises
		}
		return nil
	})
	if err != nil {
		return domaintemplate.Template{}, err
	}
	return updated, nil
}

// Publish makes an owned template publicly visible.
func (s *TemplateService) Publish(ctx context.Context, id, userID uuid.UUID) (domaintemplate.Template, error) {
	t, err := s.templates.FindByID(ctx, id)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	if t.CreatedByUserID != userID {
		return domaintemplate.Template{}, domaintemplate.ErrNotOwner
	}

	t.IsPublic = true
	t.UpdatedAt = time.Now()
	return s.templates.Update(ctx, t)
}

// Fork copies a template, including its exercises and media, under the given
// user as the new owner.
func (s *TemplateService) Fork(ctx context.Context, id, userID uuid.UUID) (domaintemplate.Template, error) {
	source, err := s.templates.FindByID(ctx, id)
	if err != nil {
		return domaintemplate.Template{}, err
	}

	fork, err := domaintemplate.NewTemplate(source.Name, source.Description, userID)
	if err != nil {
		return domaintemplate.Template{}, err
	}

	exercises := make([]domaintemplate.TemplateExercise, 0, len(source.Exercises))
	for _, e := range source.Exercises {
		exercises = append(exercises, domaintemplate.TemplateExercise{
			TemplateID:  fork.ID,
			ExerciseID:  e.ExerciseID,
			SortOrder:   e.SortOrder,
			PlannedSets: e.PlannedSets,
		})
	}

	media := make([]domaintemplate.TemplateMedia, 0, len(source.Media))
	for _, m := range source.Media {
		media = append(media, domaintemplate.TemplateMedia{
			ID:         uuid.Must(uuid.NewV7()),
			TemplateID: fork.ID,
			MediaType:  m.MediaType,
			SortOrder:  m.SortOrder,
			S3Key:      m.S3Key,
		})
	}

	var created domaintemplate.Template
	err = s.trManager.Do(ctx, func(ctx context.Context) error {
		created, err = s.templates.Create(ctx, fork)
		if err != nil {
			return err
		}
		if err = s.templates.BatchInsertExercises(ctx, exercises); err != nil {
			return err
		}
		if err = s.templates.BatchInsertMedia(ctx, media); err != nil {
			return err
		}
		created.Exercises = exercises
		created.Media = media
		return nil
	})
	if err != nil {
		return domaintemplate.Template{}, err
	}
	return created, nil
}

// SoftDelete marks an owned template as deleted.
func (s *TemplateService) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	t, err := s.templates.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if t.CreatedByUserID != userID {
		return domaintemplate.ErrNotOwner
	}
	return s.templates.SoftDelete(ctx, id)
}

// buildExercises converts command exercises into domain rows, assigning each
// a sequential sort order within the template.
func buildExercises(templateID uuid.UUID, items []TemplateExerciseCommand) []domaintemplate.TemplateExercise {
	exercises := make([]domaintemplate.TemplateExercise, 0, len(items))
	for i, item := range items {
		exercises = append(exercises, domaintemplate.TemplateExercise{
			TemplateID:  templateID,
			ExerciseID:  item.ExerciseID,
			SortOrder:   i + 1,
			PlannedSets: item.PlannedSets,
		})
	}
	return exercises
}
