// Package template contains the template use cases and the interfaces the
// service consumes, declared where they are used per ISP.
package template

import (
	"context"

	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
)

type TemplateRepository interface {
	List(ctx context.Context, filter domaintemplate.TemplateFilter) ([]domaintemplate.Template, error)
	FindByID(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error)
	Create(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	Update(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
