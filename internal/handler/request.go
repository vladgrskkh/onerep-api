package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// ErrValidationFailed is returned by DecodeAndValidate when the decoded
// request body fails DTO validation.
var ErrValidationFailed = errors.New("request validation failed")

func DecodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

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

// ParseRFC3339QueryParam parses an RFC 3339 query parameter. A missing
// parameter yields a zero time and a nil error.
func ParseRFC3339QueryParam(r *http.Request, key string) (time.Time, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
