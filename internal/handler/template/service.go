package template

import (
	"context"

	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

// TemplateService is the template use-case contract consumed by the handler.
type TemplateService interface {
	List(ctx context.Context, filter domaintemplate.TemplateFilter) ([]domaintemplate.Template, error)
	Get(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error)
	Create(ctx context.Context, cmd servicetemplate.CreateTemplateCommand) (domaintemplate.Template, error)
	Update(ctx context.Context, cmd servicetemplate.UpdateTemplateCommand) (domaintemplate.Template, error)
	Publish(ctx context.Context, id, userID uuid.UUID) (domaintemplate.Template, error)
	Fork(ctx context.Context, id, userID uuid.UUID) (domaintemplate.Template, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) error
}
