package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSpecPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid simple path", path: "/openapi.json", wantErr: false},
		{name: "valid nested path", path: "/api/v1/openapi.json", wantErr: false},
		{name: "valid path with query", path: "/openapi.json?format=yaml", wantErr: false},

		// Authority-confusion SSRF payloads.
		{name: "at-sign authority confusion", path: "@attacker.com/openapi.json", wantErr: true},
		{name: "at-sign with port", path: "@127.0.0.1:8080/openapi.json", wantErr: true},

		// Scheme-based payloads.
		{name: "absolute URL with https", path: "https://attacker.com/openapi.json", wantErr: true},
		{name: "absolute URL with http", path: "http://attacker.com/openapi.json", wantErr: true},

		// Authority reference payloads.
		{name: "double-slash authority", path: "//attacker.com/openapi.json", wantErr: true},

		// Relative paths (no leading slash).
		{name: "relative path", path: "openapi.json", wantErr: true},
		{name: "empty string", path: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := validateSpecPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, parsed)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, parsed)
		})
	}
}
