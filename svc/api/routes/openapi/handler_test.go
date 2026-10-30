package handler_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/openapi"
	"gopkg.in/yaml.v3"
)

func TestHandler_ServesEquivalentSpecifications(t *testing.T) {
	var expected map[string]any
	require.NoError(t, yaml.Unmarshal(openapi.Spec, &expected))

	for _, format := range []string{"yaml", "json"} {
		route := &handler.Handler{Format: format}
		t.Run(route.Path(), func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(route.Method(), route.Path(), nil)
			session := &zen.Session{}
			require.NoError(t, session.Init(response, request, 0))
			require.NoError(t, route.Handle(t.Context(), session))
			require.Equal(t, 200, response.Code)
			require.Equal(t, "public, max-age=3600", response.Header().Get("Cache-Control"))
			if format == "json" {
				require.Equal(t, "/openapi.json", route.Path())
				require.Equal(t, "application/json", response.Header().Get("Content-Type"))
				require.True(t, json.Valid(response.Body.Bytes()))
				var actual map[string]any
				require.NoError(t, yaml.Unmarshal(response.Body.Bytes(), &actual))
				require.Equal(t, expected, actual)
			} else {
				require.Equal(t, "/openapi.yaml", route.Path())
				require.Equal(t, "application/yaml", response.Header().Get("Content-Type"))
				require.Equal(t, openapi.Spec, response.Body.Bytes())
			}
		})
	}
}

func TestHandler_RejectsUnsupportedFormat(t *testing.T) {
	route := &handler.Handler{Format: "xml"}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(route.Method(), route.Path(), nil)
	session := &zen.Session{}
	require.NoError(t, session.Init(response, request, 0))
	require.Error(t, route.Handle(t.Context(), session))
	require.Empty(t, response.Body.Bytes())
	require.Empty(t, response.Header().Get("Cache-Control"))
}
