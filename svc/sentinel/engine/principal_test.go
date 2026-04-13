package engine

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMetaBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    map[string]any
		wantErr bool
	}{
		{
			name:  "empty input yields empty map",
			input: []byte{},
			want:  map[string]any{},
		},
		{
			name:  "nil input yields empty map",
			input: nil,
			want:  map[string]any{},
		},
		{
			name:  "empty object yields empty map",
			input: []byte(`{}`),
			want:  map[string]any{},
		},
		{
			name:  "flat object",
			input: []byte(`{"plan":"pro","org":"acme"}`),
			want:  map[string]any{"plan": "pro", "org": "acme"},
		},
		{
			name:  "nested object",
			input: []byte(`{"features":{"analytics":true,"seats":5}}`),
			want: map[string]any{
				"features": map[string]any{
					"analytics": true,
					"seats":     float64(5),
				},
			},
		},
		{
			name:  "JSON null normalizes to empty map",
			input: []byte(`null`),
			want:  map[string]any{},
		},
		{
			name:    "JSON array at root is an error",
			input:   []byte(`[1,2,3]`),
			wantErr: true,
		},
		{
			name:    "JSON scalar at root is an error",
			input:   []byte(`"pro"`),
			wantErr: true,
		},
		{
			name:    "malformed JSON is an error",
			input:   []byte(`{"plan":`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseMetaBytes(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got, "empty metadata must serialize as {} not null")
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseMetaString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   sql.NullString
		want map[string]any
	}{
		{
			name: "null column yields empty map",
			in:   sql.NullString{Valid: false},
			want: map[string]any{},
		},
		{
			name: "valid empty string yields empty map",
			in:   sql.NullString{Valid: true, String: ""},
			want: map[string]any{},
		},
		{
			name: "valid JSON string parses",
			in:   sql.NullString{Valid: true, String: `{"plan":"pro"}`},
			want: map[string]any{"plan": "pro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseMetaString(tt.in)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.want, got)
		})
	}
}
