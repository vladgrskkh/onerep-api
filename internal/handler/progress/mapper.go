package progress

import (
	"github.com/google/uuid"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler/progress/dto"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
)

// toLogBodyWeightCommand maps the HTTP request to the service command.
func toLogBodyWeightCommand(req dto.LogBodyWeightRequest, userID uuid.UUID) servicebodyweight.LogBodyWeightCommand {
	return servicebodyweight.LogBodyWeightCommand{
		UserID:     userID,
		WeightKg:   req.WeightKg,
		MeasuredAt: req.MeasuredAt,
	}
}

// toOneRMResponse maps the domain progress row to the HTTP response.
func toOneRMResponse(p *domainprogress.Progress1RM) dto.OneRMResponse {
	return dto.OneRMResponse{
		Date:         p.Date,
		Estimated1RM: p.Estimated1RM,
	}
}

// toOneRMListResponse maps domain progress rows to HTTP responses.
func toOneRMListResponse(progress []*domainprogress.Progress1RM) []dto.OneRMResponse {
	resp := make([]dto.OneRMResponse, 0, len(progress))
	for _, p := range progress {
		resp = append(resp, toOneRMResponse(p))
	}
	return resp
}

// toVolumeResponse maps the domain volume row to the HTTP response.
func toVolumeResponse(v *domainprogress.ProgressVolume) dto.VolumeResponse {
	return dto.VolumeResponse{
		Date:          v.Date,
		MuscleGroupID: v.MuscleGroupID,
		TotalKG:       v.TotalKG,
	}
}

// toVolumeListResponse maps domain volume rows to HTTP responses.
func toVolumeListResponse(volumes []*domainprogress.ProgressVolume) []dto.VolumeResponse {
	resp := make([]dto.VolumeResponse, 0, len(volumes))
	for _, v := range volumes {
		resp = append(resp, toVolumeResponse(v))
	}
	return resp
}

// toBodyWeightResponse maps the domain body weight to the HTTP response.
func toBodyWeightResponse(bw *domainbodyweight.BodyWeight) dto.BodyWeightResponse {
	return dto.BodyWeightResponse{
		ID:         bw.ID,
		WeightKg:   bw.WeightKg,
		MeasuredAt: bw.MeasuredAt,
		CreatedAt:  bw.CreatedAt,
	}
}

// toBodyWeightListResponse maps domain body weights to HTTP responses.
func toBodyWeightListResponse(weights []*domainbodyweight.BodyWeight) []dto.BodyWeightResponse {
	resp := make([]dto.BodyWeightResponse, 0, len(weights))
	for _, bw := range weights {
		resp = append(resp, toBodyWeightResponse(bw))
	}
	return resp
}
