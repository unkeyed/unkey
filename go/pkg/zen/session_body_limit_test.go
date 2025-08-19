package zen

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_BodySizeLimit(t *testing.T) {
	tests := []struct {
		name        string
		bodyContent string
		maxBodySize int64
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "body within limit",
			bodyContent: `{"name":"test"}`,
			maxBodySize: 100,
			wantErr:     false,
		},
		{
			name:        "body exceeds limit",
			bodyContent: strings.Repeat("x", 200),
			maxBodySize: 100,
			wantErr:     true,
			errSubstr:   "request body too large",
		},
		{
			name:        "no limit enforced when maxBodySize is 0",
			bodyContent: strings.Repeat("x", 1000),
			maxBodySize: 0,
			wantErr:     false,
		},
		{
			name:        "no limit enforced when maxBodySize is negative",
			bodyContent: strings.Repeat("x", 1000),
			maxBodySize: -1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.bodyContent))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.init(w, req, tt.maxBodySize)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, []byte(tt.bodyContent), sess.requestBody)
		})
	}
}

func TestSession_BodySizeLimitWithBindBody(t *testing.T) {
	// Test that BindBody still works correctly with body size limits
	bodyContent := `{"name":"test","value":42}`

	req := httptest.NewRequest("POST", "/", strings.NewReader(bodyContent))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req, 1024) // 1KB limit
	require.NoError(t, err)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	var data TestData
	err = sess.BindBody(&data)
	require.NoError(t, err)
	assert.Equal(t, "test", data.Name)
	assert.Equal(t, 42, data.Value)
}
