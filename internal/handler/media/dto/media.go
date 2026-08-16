package dto

import "github.com/google/uuid"

type ExerciseMediaUploadRequest struct {
	MediaType   string `json:"media_type"   validate:"required,oneof=photo video"`
	ContentType string `json:"content_type" validate:"required"`
}

type TemplateMediaUploadRequest struct {
	ContentType string `json:"content_type" validate:"required"`
}

type MediaUploadResponse struct {
	ID        uuid.UUID `json:"id"`
	S3Key     string    `json:"s3_key"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int64     `json:"expires_in,omitzero"`
}
