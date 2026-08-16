package media

import (
	"time"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler/media/dto"
)

// toExerciseMediaUploadResponse maps an exercise media upload to the HTTP
// response.
func toExerciseMediaUploadResponse(
	media *domainexercise.ExerciseMedia,
	uploadURL string,
	expiresIn time.Duration,
) dto.MediaUploadResponse {
	return dto.MediaUploadResponse{
		ID:        media.ID,
		S3Key:     media.S3Key,
		UploadURL: uploadURL,
		ExpiresIn: int64(expiresIn / time.Second),
	}
}

// toTemplateMediaUploadResponse maps a template media upload to the HTTP
// response.
func toTemplateMediaUploadResponse(
	media *domaintemplate.TemplateMedia,
	uploadURL string,
	expiresIn time.Duration,
) dto.MediaUploadResponse {
	return dto.MediaUploadResponse{
		ID:        media.ID,
		S3Key:     media.S3Key,
		UploadURL: uploadURL,
		ExpiresIn: int64(expiresIn / time.Second),
	}
}
