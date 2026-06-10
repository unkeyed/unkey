package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGitContextURL(t *testing.T) {
	tests := []struct {
		name           string
		repository     string
		forkRepository string
		commitSHA      string
		prNumber       int64
		contextPath    string
		want           string
	}{
		{
			name:       "branch push uses commit sha on base repo",
			repository: "acme/app",
			commitSHA:  "deadbeef",
			want:       "https://github.com/acme/app.git#deadbeef",
		},
		{
			name:           "fork PR fetches refs/pull from BASE repo, not the fork",
			repository:     "acme/app",
			forkRepository: "attacker/app",
			commitSHA:      "deadbeef",
			prNumber:       42,
			// GitHub serves refs/pull/<n>/head only on the base repo. Using the
			// fork here (the old bug) produced an unresolvable URL.
			want: "https://github.com/acme/app.git#refs/pull/42/head",
		},
		{
			name:           "fork redeploy by concrete SHA fetches from the fork",
			repository:     "acme/app",
			forkRepository: "contributor/app",
			commitSHA:      "cafebabe",
			prNumber:       0,
			want:           "https://github.com/contributor/app.git#cafebabe",
		},
		{
			name:        "context subdir is appended",
			repository:  "acme/app",
			commitSHA:   "deadbeef",
			contextPath: "services/api",
			want:        "https://github.com/acme/app.git#deadbeef:services/api",
		},
		{
			name:           "fork PR with subdir still targets base repo",
			repository:     "acme/app",
			forkRepository: "attacker/app",
			commitSHA:      "deadbeef",
			prNumber:       7,
			contextPath:    "svc",
			want:           "https://github.com/acme/app.git#refs/pull/7/head:svc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildGitContextURL(tt.repository, tt.forkRepository, tt.commitSHA, tt.prNumber, tt.contextPath)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestBuildGitSolverOptions_EnvSecretGating proves env vars only become a
// BuildKit secret when present. The fork-build path passes a nil map (see
// buildDockerImageFromGit), so no env secret is registered and a fork-controlled
// Dockerfile has nothing to mount.
func TestBuildGitSolverOptions_EnvSecretGating(t *testing.T) {
	w := &Workflow{
		registryConfig: RegistryConfig{
			Repository: "registry.example.com/unkey",
			Username:   "u",
			Password:   "p",
		},
	}

	t.Run("nil env registers no env secret", func(t *testing.T) {
		opts, err := w.buildGitSolverOptions("linux/amd64", "https://github.com/acme/app.git#refs/pull/1/head", "Dockerfile", "img:tag", "ghs_token", nil)
		require.NoError(t, err)
		require.NotContains(t, opts.FrontendAttrs, "label:org.unkey.env-hash")
		require.NotContains(t, opts.FrontendAttrs, "build-arg:UNKEY_SECRETS_ID")
	})

	t.Run("non-empty env registers env secret", func(t *testing.T) {
		opts, err := w.buildGitSolverOptions("linux/amd64", "https://github.com/acme/app.git#deadbeef", "Dockerfile", "img:tag", "ghs_token", map[string]string{"FOO": "bar"})
		require.NoError(t, err)
		require.Contains(t, opts.FrontendAttrs, "label:org.unkey.env-hash")
		require.Contains(t, opts.FrontendAttrs, "build-arg:UNKEY_SECRETS_ID")
	})
}
