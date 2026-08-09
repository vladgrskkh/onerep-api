package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/vladgrskkh/onerep-api/internal/handler/middleware"
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, id)
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

func DecodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ErrValidationFailed is returned by DecodeAndValidate when the decoded
// request body fails DTO validation.
var ErrValidationFailed = errors.New("request validation failed")

// DecodeAndValidate decodes the JSON request body into v and runs
// go-playground/validator over its validate: tags. Decode errors are returned
// as-is; validation errors are wrapped in ErrValidationFailed.
func DecodeAndValidate(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}

	validate := validator.New()
	validate.RegisterTagNameFunc(jsonFieldName)
	if err := validate.Struct(v); err != nil {
		return errors.Join(ErrValidationFailed, err)
	}
	return nil
}

// jsonFieldName returns the json tag name of a struct field so that validation
// details use the wire names (e.g. "display_name") instead of Go names.
func jsonFieldName(fld reflect.StructField) string {
	name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}
