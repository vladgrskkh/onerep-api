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

	tmpl, err := template.NewTemplate("  Push Day  ", "Chest, shoulders, triceps", createdBy)

	s.Require().NoError(err)
	s.NotEqual(uuid.Nil, tmpl.ID)
	s.Equal(uuid.Version(7), tmpl.ID.Version())
	s.Equal("Push Day", tmpl.Name)
	s.Equal("Chest, shoulders, triceps", tmpl.Description)
	s.False(tmpl.IsPublic)
	s.Equal(createdBy, tmpl.CreatedByUserID)
	s.False(tmpl.CreatedAt.IsZero())
	s.False(tmpl.UpdatedAt.IsZero())
	s.WithinDuration(time.Now(), tmpl.CreatedAt, time.Minute)
	s.True(tmpl.DeletedAt.IsZero())
	s.Equal(1, tmpl.Version)
	s.Empty(tmpl.Exercises)
	s.Empty(tmpl.Media)
}

func (s *TemplateTestSuite) TestNewTemplate_Validation() {
	createdBy := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")

	tests := []struct {
		name            string
		templateName    string
		createdByUserID uuid.UUID
		wantErr         error
	}{
		{name: "valid", templateName: "Push Day", createdByUserID: createdBy},
		{name: "blank name", templateName: "   ", createdByUserID: createdBy, wantErr: template.ErrInvalidName},
		{name: "nil user id", templateName: "Push Day", createdByUserID: uuid.Nil, wantErr: template.ErrInvalidUserID},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tmpl, err := template.NewTemplate(tt.templateName, "desc", tt.createdByUserID)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, tmpl.ID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.templateName, tmpl.Name)
		})
	}
}

func (s *TemplateTestSuite) TestMediaTypePhoto() {
	s.Equal(template.MediaTypePhoto, template.MediaType("photo"))
}

func (s *TemplateTestSuite) TestErrors() {
	s.Require().ErrorContains(template.ErrTemplateNotFound, "template not found")
	s.Require().ErrorContains(template.ErrNotOwner, "not the owner")
}

func TestTemplateSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TemplateTestSuite))
}
