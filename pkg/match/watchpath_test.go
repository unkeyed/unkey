package match

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestMatchWatchPaths(t *testing.T) {
	tests := []struct {
		name         string
		patterns     []string
		changedFiles []string
		want         bool
		wantInvalid  string
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
			name:         "invalid sibling blocks a valid match",
			patterns:     []string{"[invalid", "src/**"},
			changedFiles: []string{"src/main.go"},
			want:         false,
			wantInvalid:  "[invalid",
		},
		{
			// Order must not decide the outcome: without up-front validation the
			// loop returns on "src/**" and never parses the broken sibling.
			name:         "invalid sibling blocks a match that comes first",
			patterns:     []string{"src/**", "[invalid"},
			changedFiles: []string{"src/main.go"},
			want:         false,
			wantInvalid:  "[invalid",
		},
		{
			name:         "multiple files, one matches",
			patterns:     []string{"src/**"},
			changedFiles: []string{"docs/readme.md", "src/main.go"},
			want:         true,
		},
		{
			name:         "bad pattern is reported",
			patterns:     []string{"[invalid"},
			changedFiles: []string{"src/main.go"},
			want:         false,
			wantInvalid:  "[invalid",
		},
		{
			name:         "bad pattern is reported when there are no changed files",
			patterns:     []string{"[invalid"},
			changedFiles: []string{},
			want:         false,
			wantInvalid:  "[invalid",
		},
		{
			name:         "first bad pattern wins",
			patterns:     []string{"src/**", "{KEBAP", "[invalid"},
			changedFiles: []string{"src/main.go"},
			want:         false,
			wantInvalid:  "{KEBAP",
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
			got, err := MatchWatchPaths(tt.patterns, tt.changedFiles)
			require.Equal(t, tt.want, got)

			if tt.wantInvalid == "" {
				require.NoError(t, err)
				return
			}
			requireInvalidWatchPath(t, err, tt.wantInvalid)
		})
	}
}

func TestValidateWatchPaths(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		wantInvalid string
	}{
		{
			name:     "nil patterns",
			patterns: nil,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
		},
		{
			name:     "all valid",
			patterns: []string{"src/**", "**/*.go", "KEBAP"},
		},
		{
			name:        "unclosed bracket",
			patterns:    []string{"src/["},
			wantInvalid: "src/[",
		},
		{
			name:        "unclosed bracket range",
			patterns:    []string{"src/[a-"},
			wantInvalid: "src/[a-",
		},
		{
			name:        "unclosed brace",
			patterns:    []string{"{src,lib"},
			wantInvalid: "{src,lib",
		},
		{
			name:        "trailing backslash",
			patterns:    []string{`src\`},
			wantInvalid: `src\`,
		},
		{
			name:        "reports the first invalid pattern in input order",
			patterns:    []string{"src/**", "src/[", "{src,lib"},
			wantInvalid: "src/[",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWatchPaths(tt.patterns)

			if tt.wantInvalid == "" {
				require.NoError(t, err)
				return
			}
			requireInvalidWatchPath(t, err, tt.wantInvalid)
		})
	}
}

// requireInvalidWatchPath asserts the fault carries the validation code and names
// the pattern in the message callers surface to users.
func requireInvalidWatchPath(t *testing.T, err error, pattern string) {
	t.Helper()

	require.Error(t, err)

	code, ok := fault.GetCode(err)
	require.True(t, ok, "fault carries no code")
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), code)

	require.Contains(t, fault.UserFacingMessage(err), pattern)
}
