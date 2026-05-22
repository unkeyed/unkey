package match

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleMatchWatchPaths() {
	changedFiles := []string{"docs/readme.md", "src/api/server.go"}

	fmt.Println(MatchWatchPaths([]string{"src/**/*.go"}, changedFiles))
	fmt.Println(MatchWatchPaths([]string{"web/**/*.tsx"}, changedFiles))

	// Output:
	// true
	// false
}

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
			name:         "empty patterns and empty changed files matches nothing",
			patterns:     []string{},
			changedFiles: []string{},
			want:         false,
		},
		{
			name:         "empty patterns and empty changed file path matches nothing",
			patterns:     []string{},
			changedFiles: []string{""},
			want:         false,
		},
		{
			name:         "empty patterns match when at least one changed file is valid",
			patterns:     []string{},
			changedFiles: []string{"", "src/main.go"},
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
			name:         "globstar recursive match",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src/pkg/foo/bar.go"},
			want:         true,
		},
		{
			name:         "globstar matches root file",
			patterns:     []string{"**"},
			changedFiles: []string{"main.go"},
			want:         true,
		},
		{
			name:         "globstar matches nested file",
			patterns:     []string{"**"},
			changedFiles: []string{"src/pkg/main.go"},
			want:         true,
		},
		{
			name:         "globstar in middle matches zero segments",
			patterns:     []string{"src/**/main.go"},
			changedFiles: []string{"src/main.go"},
			want:         true,
		},
		{
			name:         "globstar in middle matches multiple segments",
			patterns:     []string{"src/**/main.go"},
			changedFiles: []string{"src/pkg/api/main.go"},
			want:         true,
		},
		{
			name:         "multiple globstars match nested file",
			patterns:     []string{"**/**/main.go"},
			changedFiles: []string{"src/pkg/main.go"},
			want:         true,
		},
		{
			name:         "trailing globstar does not match sibling prefix",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src-main.go"},
			want:         false,
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
			name:         "multiple patterns and multiple files, one pair matches",
			patterns:     []string{"docs/**", "api/**/*.ts", "src/**/*.go"},
			changedFiles: []string{"README.md", "web/app/page.ts", "src/pkg/main.go"},
			want:         true,
		},
		{
			name:         "multiple patterns and multiple files, no pairs match",
			patterns:     []string{"docs/**", "api/**/*.ts"},
			changedFiles: []string{"README.md", "src/pkg/main.go"},
			want:         false,
		},
		{
			name:         "multiple invalid patterns are skipped and valid pattern matches later file",
			patterns:     []string{"***", "../*", "src/**"},
			changedFiles: []string{"README.md", "src/main.go"},
			want:         true,
		},
		{
			name:         "multiple invalid files are skipped and valid file matches later pattern",
			patterns:     []string{"docs/**", "src/**"},
			changedFiles: []string{"", "../main.go", "src/main.go"},
			want:         true,
		},
		{
			name:         "multiple invalid patterns and files match nothing",
			patterns:     []string{"***", "../*"},
			changedFiles: []string{"", "../main.go"},
			want:         false,
		},
		{
			name:         "bad pattern is skipped",
			patterns:     []string{"[invalid"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "bad pattern is skipped and later pattern can match",
			patterns:     []string{"[invalid", "src/**"},
			changedFiles: []string{"src/main.go"},
			want:         true,
		},
		{
			name:         "bad changed file is skipped and later file can match",
			patterns:     []string{"src/**"},
			changedFiles: []string{"", "src/main.go"},
			want:         true,
		},
		{
			name:         "extension match",
			patterns:     []string{"**/*.ts"},
			changedFiles: []string{"web/app/page.ts"},
			want:         true,
		},
		{
			name:         "recursive extension match includes direct child",
			patterns:     []string{"api/**/*.ts"},
			changedFiles: []string{"api/hello.ts"},
			want:         true,
		},
		{
			name:         "recursive extension match includes nested child",
			patterns:     []string{"api/**/*.ts"},
			changedFiles: []string{"api/hello/world.ts"},
			want:         true,
		},
		{
			name:         "single star does not cross path separator",
			patterns:     []string{"api/*.js"},
			changedFiles: []string{"api/hello/world.js"},
			want:         false,
		},
		{
			name:         "globstar must be its own path segment",
			patterns:     []string{"src/**.go"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "triple star is not supported",
			patterns:     []string{"***"},
			changedFiles: []string{"main.go"},
			want:         false,
		},
		{
			name:         "triple star segment is not supported",
			patterns:     []string{"src/***/main.go"},
			changedFiles: []string{"src/app/main.go"},
			want:         false,
		},
		{
			name:         "parent directory pattern is not supported",
			patterns:     []string{"../*"},
			changedFiles: []string{"../main.go"},
			want:         false,
		},
		{
			name:         "parent directory file path is not supported",
			patterns:     []string{"*/.."},
			changedFiles: []string{"src/.."},
			want:         false,
		},
		{
			name:         "empty pattern is skipped",
			patterns:     []string{""},
			changedFiles: []string{"main.go"},
			want:         false,
		},
		{
			name:         "root pattern is skipped",
			patterns:     []string{"/"},
			changedFiles: []string{"main.go"},
			want:         false,
		},
		{
			name:         "leading slash pattern is not supported",
			patterns:     []string{"/src/**"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "trailing slash pattern is not supported",
			patterns:     []string{"src/"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "leading slash file path is not supported",
			patterns:     []string{"src/**"},
			changedFiles: []string{"/src/main.go"},
			want:         false,
		},
		{
			name:         "trailing slash file path is not supported",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src/main.go/"},
			want:         false,
		},
		{
			name:         "double slash pattern is not supported",
			patterns:     []string{"src//main.go"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "double slash file path is not supported",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src//main.go"},
			want:         false,
		},
		{
			name:         "backslash pattern is not supported",
			patterns:     []string{"src\\*.go"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "backslash file path is not supported",
			patterns:     []string{"src/**"},
			changedFiles: []string{"src\\main.go"},
			want:         false,
		},
		{
			name:         "negative pattern is not supported",
			patterns:     []string{"!src/**"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "brace expansion is not supported",
			patterns:     []string{"{src,api}/**"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "extglob is not supported",
			patterns:     []string{"@(src|api)/**"},
			changedFiles: []string{"src/main.go"},
			want:         false,
		},
		{
			name:         "character class is not supported",
			patterns:     []string{"src/[ab].go"},
			changedFiles: []string{"src/a.go"},
			want:         false,
		},
		{
			name:         "invalid character class is skipped",
			patterns:     []string{"src/[abc/main.go"},
			changedFiles: []string{"src/a/main.go"},
			want:         false,
		},
		{
			name:         "question mark matches one character in segment",
			patterns:     []string{"src/?.go"},
			changedFiles: []string{"src/a.go"},
			want:         true,
		},
		{
			name:         "question mark does not cross path separator",
			patterns:     []string{"src/?.go"},
			changedFiles: []string{"src/ab.go"},
			want:         false,
		},
		{
			name:         "leading dot slash pattern is normalized",
			patterns:     []string{"./src/**"},
			changedFiles: []string{"src/main.go"},
			want:         true,
		},
		{
			name:         "leading dot slash file path is normalized",
			patterns:     []string{"src/**"},
			changedFiles: []string{"./src/main.go"},
			want:         true,
		},
		{
			name:         "wildcard matches dotfile",
			patterns:     []string{"*"},
			changedFiles: []string{".env"},
			want:         true,
		},
		{
			name:         "globstar matches dotfile",
			patterns:     []string{"**/*"},
			changedFiles: []string{"src/.env"},
			want:         true,
		},
		{
			name:         "explicit dot matches dotfile",
			patterns:     []string{".*"},
			changedFiles: []string{".env"},
			want:         true,
		},
		{
			name:         "leading exclamation mark matches literal file",
			patterns:     []string{"!important.md"},
			changedFiles: []string{"!important.md"},
			want:         true,
		},
		{
			name:         "braces match literal file",
			patterns:     []string{"docs/{draft}.md"},
			changedFiles: []string{"docs/{draft}.md"},
			want:         true,
		},
		{
			name:         "extglob-like characters match literal file",
			patterns:     []string{"foo+(bar).txt"},
			changedFiles: []string{"foo+(bar).txt"},
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

func TestMatchWatchPathErrors(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		file    string
	}{
		{name: "empty pattern", pattern: "", file: "main.go"},
		{name: "root pattern", pattern: "/", file: "main.go"},
		{name: "leading slash pattern", pattern: "/src/**", file: "src/main.go"},
		{name: "trailing slash pattern", pattern: "src/", file: "src/main.go"},
		{name: "double slash pattern", pattern: "src//main.go", file: "src/main.go"},
		{name: "backslash pattern", pattern: "src\\*.go", file: "src/main.go"},
		{name: "negative pattern", pattern: "!src/**", file: "src/main.go"},
		{name: "brace expansion", pattern: "{src,api}/**", file: "src/main.go"},
		{name: "extglob", pattern: "@(src|api)/**", file: "src/main.go"},
		{name: "triple star", pattern: "***", file: "main.go"},
		{name: "character class", pattern: "src/[ab].go", file: "src/a.go"},
		{name: "invalid character class", pattern: "src/[abc/main.go", file: "src/a/main.go"},
		{name: "parent directory pattern", pattern: "../*", file: "../main.go"},
		{name: "parent directory file", pattern: "*/..", file: "src/.."},
		{name: "empty file", pattern: "src/**", file: ""},
		{name: "leading slash file", pattern: "src/**", file: "/src/main.go"},
		{name: "trailing slash file", pattern: "src/**", file: "src/main.go/"},
		{name: "double slash file", pattern: "src/**", file: "src//main.go"},
		{name: "backslash file", pattern: "src/**", file: "src\\main.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := matchWatchPath(tt.pattern, tt.file)
			require.ErrorIs(t, err, errInvalidWatchPath)
			require.False(t, matched)
		})
	}
}
