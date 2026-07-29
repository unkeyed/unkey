package imageref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  string
		wantError bool
	}{
		{
			name:     "docker hub shorthand with tag",
			input:    "nginx:1.27",
			expected: "index.docker.io/library/nginx:1.27",
		},
		{
			name:     "registry reference with digest",
			input:    "ghcr.io/acme/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: "ghcr.io/acme/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			name:      "implicit latest is rejected",
			input:     "nginx",
			wantError: true,
		},
		{
			name:      "empty reference is rejected",
			input:     "  ",
			wantError: true,
		},
		{
			name:      "malformed reference is rejected",
			input:     "invalid reference:tag",
			wantError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := Normalize(testCase.input)
			if testCase.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expected, actual)
		})
	}
}

func TestNormalizeHistoricalMakesLatestExplicit(t *testing.T) {
	normalized, err := NormalizeHistorical("nginx")
	require.NoError(t, err)
	require.Equal(t, "index.docker.io/library/nginx:latest", normalized)
}
