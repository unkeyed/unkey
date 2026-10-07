package workos

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateDevice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/user_management/authorize/device", r.URL.Path)
		require.Equal(t, "Bearer sk_test", r.Header.Get("Authorization"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]string
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "client_123", payload["client_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code":"device-secret",
			"user_code":"abcd-efgh",
			"verification_uri":"https://auth.example/device",
			"verification_uri_complete":"https://auth.example/device?user_code=ABCD-EFGH",
			"expires_in":300,
			"interval":5
		}`))
	}))
	t.Cleanup(srv.Close)

	client := NewDeviceClient("sk_test", srv.URL)
	code, got, err := client.CreateDevice(context.Background(), "client_123")
	require.NoError(t, err)
	require.Equal(t, DeviceCode("device-secret"), code)
	require.Equal(t, "abcd-efgh", got.UserCode)
	require.Equal(t, 300, got.ExpiresIn)
	require.Equal(t, 5, got.Interval)
}
