package template

import (
	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

// toCreateCommand maps the HTTP request to the service create command.
func toCreateCommand(req dto.TemplateCreateRequest, userID uuid.UUID) servicetemplate.CreateTemplateCommand {
	cmd := servicetemplate.CreateTemplateCommand{
		Name:        req.Name,
		Description: req.Description,
		UserID:      userID,
		Exercises:   make([]servicetemplate.TemplateExerciseCommand, 0, len(req.Exercises)),
	}
	for _, e := range req.Exercises {
		cmd.Exercises = append(cmd.Exercises, servicetemplate.TemplateExerciseCommand{
			ExerciseID:  e.ExerciseID,
			PlannedSets: e.PlannedSets,
		})
	}
	return cmd
}

// toUpdateCommand maps the HTTP request to the service update command.
func toUpdateCommand(
	req dto.TemplateUpdateRequest,
	templateID uuid.UUID,
	userID uuid.UUID,
) servicetemplate.UpdateTemplateCommand {
	cmd := servicetemplate.UpdateTemplateCommand{
		ID:          templateID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Exercises != nil {
		exercises := make([]servicetemplate.TemplateExerciseCommand, 0, len(*req.Exercises))
		for _, e := range *req.Exercises {
			exercises = append(exercises, servicetemplate.TemplateExerciseCommand{
				ExerciseID:  e.ExerciseID,
				PlannedSets: e.PlannedSets,
			})
		}
		cmd.Exercises = &exercises
	}
	return cmd
}

// toTemplateResponse maps the domain template to the HTTP response.
func toTemplateResponse(t domaintemplate.Template) dto.TemplateResponse {
	resp := dto.TemplateResponse{
		ID:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		IsPublic:        t.IsPublic,
		CreatedByUserID: t.CreatedByUserID,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		Version:         t.Version,
	}
	for _, e := range t.Exercises {
		resp.Exercises = append(resp.Exercises, dto.TemplateExerciseResponse{
			ExerciseID:  e.ExerciseID,
			SortOrder:   e.SortOrder,
			PlannedSets: e.PlannedSets,
		})
	}
	for _, m := range t.Media {
		resp.Media = append(resp.Media, dto.TemplateMediaResponse{
			ID:        m.ID,
			MediaType: string(m.MediaType),
			SortOrder: m.SortOrder,
			S3Key:     m.S3Key,
		})
	}
	return resp
}

// toTemplateListResponse maps a list of domain templates to HTTP responses.
func toTemplateListResponse(templates []domaintemplate.Template) []dto.TemplateResponse {
	resp := make([]dto.TemplateResponse, 0, len(templates))
	for _, t := range templates {
		resp = append(resp, toTemplateResponse(t))
	}
	return resp
}
