package template

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
)

type TemplateRepository interface {
	List(ctx context.Context, filter TemplateFilter) ([]domaintemplate.Template, error)
	FindByID(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error)
	Create(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	Update(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type TemplateFilter struct {
	UserID   *uuid.UUID
	IsPublic *bool
	Since    *time.Time
}
