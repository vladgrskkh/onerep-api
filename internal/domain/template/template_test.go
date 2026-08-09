package template_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/template"
)

type TemplateTestSuite struct {
	suite.Suite
}

func (s *TemplateTestSuite) TestNewTemplate() {
	createdBy := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	tmpl := template.NewTemplate("Push Day", "Chest, shoulders, triceps", createdBy)

	s.NotEqual(uuid.Nil, tmpl.ID)
	s.Equal(uuid.Version(7), tmpl.ID.Version())
	s.Equal("Push Day", tmpl.Name)
	s.Equal("Chest, shoulders, triceps", tmpl.Description)
	s.False(tmpl.IsPublic)
	s.Equal(createdBy, tmpl.CreatedByUserID)
	s.False(tmpl.CreatedAt.IsZero())
	s.False(tmpl.UpdatedAt.IsZero())
	s.WithinDuration(time.Now(), tmpl.CreatedAt, time.Minute)
	s.Nil(tmpl.DeletedAt)
	s.Equal(1, tmpl.Version)
	s.Empty(tmpl.Exercises)
	s.Empty(tmpl.Media)
}

func (s *TemplateTestSuite) TestMediaTypePhoto() {
	s.Equal(template.MediaType("photo"), template.MediaTypePhoto)
}

func (s *TemplateTestSuite) TestErrors() {
	s.ErrorContains(template.ErrTemplateNotFound, "template not found")
	s.ErrorContains(template.ErrNotOwner, "not the owner")
}

func TestTemplateSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TemplateTestSuite))
}
