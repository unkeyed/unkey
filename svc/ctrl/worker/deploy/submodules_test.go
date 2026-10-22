package deploy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSubmoduleRepos(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want []string
	}{
		{name: "sibling relative url", url: "../lib.git", want: []string{"acme/lib"}},
		{name: "relative url to another owner", url: "../../other/lib", want: []string{"other/lib"}},
		{name: "relative url above the host", url: "../../..", want: nil},
		{name: "relative url inside the superproject", url: "./lib", want: nil},
		{name: "https github url", url: "https://github.com/acme/lib.git", want: []string{"acme/lib"}},
		{name: "https www github url", url: "https://www.github.com/acme/lib", want: []string{"acme/lib"}},
		{name: "http github url gets no token", url: "http://github.com/acme/lib.git", want: nil},
		{name: "ssh url gets no token", url: "git@github.com:acme/lib.git", want: nil},
		{name: "other host", url: "https://gitlab.com/acme/lib.git", want: nil},
		{name: "url with extra path segments", url: "https://github.com/acme/lib/tree/main", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitmodules := "[submodule \"lib\"]\n\tpath = lib\n\turl = " + tt.url + "\n"
			require.Equal(t, tt.want, parseSubmoduleRepos(gitmodules, "acme/app"))
		})
	}
}

func TestExtractUserBuildErrorSubmodule(t *testing.T) {
	err := errors.New("failed to solve: failed to read dockerfile: failed to load cache key: failed to update submodules for https://github.com/acme/app.git: exit status 128")
	require.Contains(t, extractUserBuildError(err), "Git submodule could not be cloned")
}
