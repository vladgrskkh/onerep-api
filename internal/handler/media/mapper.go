package media

import (
	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	exercisedto "github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	templatedto "github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
)

// toExerciseMediaResponse maps the created exercise media to the HTTP
// response.
func toExerciseMediaResponse(m *domainexercise.ExerciseMedia) exercisedto.ExerciseMediaResponse {
	return exercisedto.ExerciseMediaResponse{
		ID:        m.ID,
		MediaType: string(m.MediaType),
		SortOrder: m.SortOrder,
		S3Key:     m.S3Key,
	}
}

// toTemplateMediaResponse maps the created template media to the HTTP
// response.
func toTemplateMediaResponse(m *domaintemplate.TemplateMedia) templatedto.TemplateMediaResponse {
	return templatedto.TemplateMediaResponse{
		ID:        m.ID,
		MediaType: string(m.MediaType),
		SortOrder: m.SortOrder,
		S3Key:     m.S3Key,
	}
}
