package exercise

import (
	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
)

// toCreateCommand maps the HTTP request to the service create command.
func toCreateCommand(req dto.ExerciseCreateRequest, userID uuid.UUID) serviceexercise.CreateExerciseCommand {
	return serviceexercise.CreateExerciseCommand{
		Name:           req.Name,
		Description:    req.Description,
		Notes:          req.Notes,
		UserID:         userID,
		MuscleGroupIDs: req.MuscleGroupIDs,
	}
}

// toUpdateCommand maps the HTTP request to the service update command.
func toUpdateCommand(req dto.ExerciseUpdateRequest, exerciseID uuid.UUID) serviceexercise.UpdateExerciseCommand {
	return serviceexercise.UpdateExerciseCommand{
		ID:             exerciseID,
		Name:           req.Name,
		Description:    req.Description,
		Notes:          req.Notes,
		MuscleGroupIDs: req.MuscleGroupIDs,
	}
}

// toExerciseResponse maps the domain exercise to the HTTP response.
func toExerciseResponse(ex *domainexercise.Exercise) dto.ExerciseResponse {
	resp := dto.ExerciseResponse{
		ID:          ex.ID,
		Name:        ex.Name,
		Description: ex.Description,
		Notes:       ex.Notes,
		IsBuiltIn:   ex.IsBuiltIn,
		CreatedAt:   ex.CreatedAt,
		UpdatedAt:   ex.UpdatedAt,
		Version:     ex.Version,
	}
	if ex.CreatedByUserID != nil {
		resp.CreatedByUserID = *ex.CreatedByUserID
	}
	for _, m := range ex.Media {
		resp.Media = append(resp.Media, dto.ExerciseMediaResponse{
			ID:        m.ID,
			MediaType: string(m.MediaType),
			SortOrder: m.SortOrder,
			S3Key:     m.S3Key,
		})
	}
	for _, g := range ex.MuscleGroups {
		resp.MuscleGroups = append(resp.MuscleGroups, dto.MuscleGroupResponse{
			ID:        g.MuscleGroupID,
			IsPrimary: g.IsPrimary,
		})
	}
	return resp
}

// toExerciseListResponse maps a list of domain exercises to HTTP responses.
func toExerciseListResponse(exercises []*domainexercise.Exercise) []dto.ExerciseResponse {
	resp := make([]dto.ExerciseResponse, 0, len(exercises))
	for _, ex := range exercises {
		resp = append(resp, toExerciseResponse(ex))
	}
	return resp
}
