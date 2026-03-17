package match

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchWatchPaths(t *testing.T) {
	tests := []struct {
		name         string
		patterns     []string
		changedFiles []string
		want         bool
	}{
		{
			name:         "empty patterns matches everything",
			patterns:     []string{},
			changedFiles: []string{"src/main.go"},
			want:         true,
		},
		{
			name:         "empty changed files matches nothing",
			patterns:     []string{"src/**"},
			changedFiles: []string{},
			want:         false,
		},
		{
			name:         "exact file match",
			patterns:     []string{"README.md"},
			changedFiles: []string{"README.md"},
			want:         true,
		},
		{
			name:         "glob star match",
			patterns:     []string{"*.go"},
			changedFiles: []string{"main.go"},
			want:         true,
		},
		{
			name:         "doublestar recursive match",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src/pkg/foo/bar.go"},
			want:         true,
		},
		{
			name:         "no match",
			patterns:     []string{"src/**"},
			changedFiles: []string{"docs/readme.md"},
			want:         false,
		},
		{
			name:         "multiple patterns, one matches",
			patterns:     []string{"docs/**", "src/**"},
			changedFiles: []string{"src/main.go"},
			want:         true,
		},
		{
			name:         "multiple files, one matches",
			patterns:     []string{"src/**"},
			changedFiles: []string{"docs/readme.md", "src/main.go"},
			want:         true,
		},
		{
			name:         "bad pattern is skipped",
			patterns:     []string{"[invalid"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "extension match",
			patterns:     []string{"**/*.ts"},
			changedFiles: []string{"web/app/page.ts"},
			want:         true,
		},
		{
			name:         "extension no match",
			patterns:     []string{"**/*.ts"},
			changedFiles: []string{"web/app/page.go"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchWatchPaths(tt.patterns, tt.changedFiles)
			require.Equal(t, tt.want, got)
		})
	}
}
