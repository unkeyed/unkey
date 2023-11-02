package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type JsonRequest struct {
	// Prints stuff using t.Log
	Debug bool
	// For Backwards compatibility until we remove all the legacy routes
	Method         string
	Path           string
	RequestHeaders map[string]string
	Body           string
	Bearer         string
	StatusCode     int
}

func Json[TResponse any](t *testing.T, app *fiber.App, r JsonRequest) TResponse {
	t.Helper()
	if r.Debug {
		t.Logf("POST: %s", r.Path)
		t.Logf("Body: %s", r.Body)
	}
	method := "POST"
	if r.Method != "" {
		method = r.Method
	}
	require.NotEmpty(t, r.Bearer, "I bet you forgot to set the bearer token")
	req := httptest.NewRequest(method, r.Path, bytes.NewBufferString(r.Body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.Bearer))
	req.Header.Set("Content-Type", "application/json")
	if r.RequestHeaders != nil {
		for k, v := range r.RequestHeaders {
			req.Header.Set(k, v)
		}
	}

	res, err := app.Test(req)
	require.NoError(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	if r.Debug {
		t.Logf("Response: %s", string(body))
	}
	if r.StatusCode != 0 {
		require.Equal(t, r.StatusCode, res.StatusCode, "status code must match: %s", string(body))
	}

	var response TResponse

	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	return response

}

type GetRequest struct {
	// Prints stuff using t.Log
	Debug          bool
	Path           string
	RequestHeaders map[string]string
	Bearer         string
	StatusCode     int
}

func Get[TResponse any](t *testing.T, app *fiber.App, r GetRequest) TResponse {
	t.Helper()
	if r.Debug {
		t.Logf("GET: %s", r.Path)
	}
	require.NotEmpty(t, r.Bearer, "I bet you forgot to set the bearer token")
	req := httptest.NewRequest("GET", r.Path, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.Bearer))
	if r.RequestHeaders != nil {
		for k, v := range r.RequestHeaders {
			req.Header.Set(k, v)
		}
	}

	res, err := app.Test(req)
	require.NoError(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	if r.Debug {
		t.Logf("Response: %s", string(body))
	}
	if r.StatusCode != 0 {
		require.Equal(t, r.StatusCode, res.StatusCode, "status code must match: %s", string(body))
	}

	var response TResponse

	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	return response

}
