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
	Debug      bool
	Method     string
	Path       string
	Body       string
	Bearer     string
	Response   any
	StatusCode int
}

func Json(t *testing.T, app *fiber.App, r JsonRequest) {
	t.Helper()
	if r.Debug {
		t.Logf("Request: %s %s", r.Method, r.Path)
		t.Logf("Body: %s", r.Body)
	}
	req := httptest.NewRequest(r.Method, r.Path, bytes.NewBufferString(r.Body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.Bearer))
	req.Header.Set("Content-Type", "application/json")

	res, err := app.Test(req)
	require.NoError(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	if r.Debug {
		t.Logf("Response: %s", string(body))
	}
	if r.StatusCode != 0 {
		require.Equal(t, r.StatusCode, res.StatusCode, "status code must match")
	}
	if r.Response != nil {

		err = json.Unmarshal(body, &r.Response)
		require.NoError(t, err)
	}

}
