// nolint:exhaustruct
package zen

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_BindBody(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	tests := []struct {
		name        string
		requestBody string
		target      interface{}
		expected    interface{}
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid json",
			requestBody: `{"name":"John","age":30,"email":"john@example.com"}`,
			target:      &testStruct{},
			expected: &testStruct{
				Name:  "John",
				Age:   30,
				Email: "john@example.com",
			},
			wantErr: false,
		},
		{
			name:        "empty json object",
			requestBody: `{}`,
			target:      &testStruct{},
			expected:    &testStruct{},
			wantErr:     false,
		},
		{
			name:        "malformed json",
			requestBody: `{"name":"John","age":30,email:"john@example.com"}`, // missing quotes around email
			target:      &testStruct{},
			wantErr:     true,
			errSubstr:   "failed to unmarshal request body",
		},
		{
			name:        "partial json",
			requestBody: `{"name":"John"}`,
			target:      &testStruct{},
			expected: &testStruct{
				Name: "John",
			},
			wantErr: false,
		},
		{
			name:        "non-object json",
			requestBody: `["array","of","strings"]`,
			target:      &testStruct{},
			wantErr:     true,
			errSubstr:   "failed to unmarshal request body",
		},
		{
			name:        "empty body",
			requestBody: ``,
			target:      &testStruct{},
			wantErr:     true,
			errSubstr:   "failed to unmarshal request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the JSON body
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create a session
			sess := &Session{}
			err := sess.init(httptest.NewRecorder(), req, 0)
			require.NoError(t, err)

			// Call BindBody
			err = sess.BindBody(tt.target)

			// Check error conditions
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			// Verify no error for positive cases
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.target)

			// Verify request body was stored in session
			assert.Equal(t, []byte(tt.requestBody), sess.requestBody)
		})
	}
}

func TestSession_BindBody_ReadError(t *testing.T) {
	// Create a reader that returns an error
	errReader := &errorReader{err: io.ErrUnexpectedEOF}
	req := httptest.NewRequest(http.MethodPost, "/", errReader)

	// Create a session and try to init it (this should fail)
	sess := &Session{}
	err := sess.init(httptest.NewRecorder(), req, 0)

	// Verify the init error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read request body")
}

// errorReader implements io.Reader to simulate read errors
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestSession_BindBody_LargeBody(t *testing.T) {
	// Create a large data structure
	type Item struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	}

	items := make([]Item, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = Item{
			ID:    i,
			Value: "test",
		}
	}

	largeData := map[string]interface{}{
		"items": items,
	}

	// Marshal it to JSON
	jsonBody, err := json.Marshal(largeData)
	require.NoError(t, err)

	// Create request with large body
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a session
	sess := &Session{}
	err = sess.init(httptest.NewRecorder(), req, 0)
	require.NoError(t, err)

	type LargeStruct struct {
		Items []Item `json:"items"`
	}

	// Call BindBody
	var target LargeStruct
	err = sess.BindBody(&target)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 1000, len(target.Items))
}

func TestSession_BindBody_Integration(t *testing.T) {
	// Test with a fully initialized session
	jsonBody := `{"name":"Integration Test","value":42}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req, 0)
	require.NoError(t, err)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	var data TestData
	err = sess.BindBody(&data)
	require.NoError(t, err)
	assert.Equal(t, "Integration Test", data.Name)
	assert.Equal(t, 42, data.Value)
}
