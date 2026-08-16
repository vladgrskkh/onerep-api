package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
)

func TestPath_NoParams(t *testing.T) {
	require.Equal(t, "/v1/exercises", testutil.Path("/v1/exercises"))
}

func TestPath_SingleParam(t *testing.T) {
	require.Equal(t, "/v1/exercises/123", testutil.Path("/v1/exercises/{id}", "123"))
}

func TestPath_MultipleParams(t *testing.T) {
	require.Equal(
		t,
		"/v1/workouts/w1/exercises/e2/sets",
		testutil.Path("/v1/workouts/{id}/exercises/{exId}/sets", "w1", "e2"),
	)
}

func TestPath_ExcessValuesIgnored(t *testing.T) {
	require.Equal(t, "/v1/exercises/123", testutil.Path("/v1/exercises/{id}", "123", "ignored"))
}
