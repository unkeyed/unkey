package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChecklyHeartbeat_Ping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful heartbeat",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "accepts 204 No Content",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			hb := NewChecklyHeartbeat(server.URL)
			err := hb.Ping(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestChecklyHeartbeat_EmptyURL(t *testing.T) {
	hb := NewChecklyHeartbeat("")
	err := hb.Ping(context.Background())
	require.Error(t, err)
}

func TestNoop_Ping(t *testing.T) {
	hb := NewNoop()
	err := hb.Ping(context.Background())
	require.NoError(t, err)
}

func TestImplementsHeartbeat(t *testing.T) {
	var _ Heartbeat = (*ChecklyHeartbeat)(nil)
	var _ Heartbeat = (*Noop)(nil)
}
