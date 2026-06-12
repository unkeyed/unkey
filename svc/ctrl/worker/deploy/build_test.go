package deploy

import (
	"fmt"
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
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
			name:           "fork ref deployed by concrete SHA fetches from the fork",
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

// TestValidateGitBuildParams pins the boundary guard that stops untrusted
// repo/SHA inputs from smuggling a URL fragment or path traversal into the git
// context URL. The public v2 API accepts commitSha as a free string, so this
// is the last line of defense before buildGitContextURL.
func TestValidateGitBuildParams(t *testing.T) {
	tests := []struct {
		name    string
		params  gitBuildParams
		wantErr bool
	}{
		{
			name:   "valid base build",
			params: gitBuildParams{Repository: "acme/app", CommitSHA: "deadbeefdeadbeef"},
		},
		{
			name:   "valid fork build by sha",
			params: gitBuildParams{Repository: "acme/app", ForkRepository: "contributor/app", CommitSHA: "cafebabe"},
		},
		{
			name:   "valid PR build with empty sha",
			params: gitBuildParams{Repository: "acme/app", PrNumber: 42},
		},
		{
			name:    "empty repository",
			params:  gitBuildParams{CommitSHA: "deadbeef"},
			wantErr: true,
		},
		{
			name:    "repository missing owner",
			params:  gitBuildParams{Repository: "app", CommitSHA: "deadbeef"},
			wantErr: true,
		},
		{
			name:    "repository with path traversal",
			params:  gitBuildParams{Repository: "acme/app/../evil", CommitSHA: "deadbeef"},
			wantErr: true,
		},
		{
			name:    "fork repository with url fragment",
			params:  gitBuildParams{Repository: "acme/app", ForkRepository: "contributor/app#evil", CommitSHA: "deadbeef"},
			wantErr: true,
		},
		{
			name:    "commit sha with subdir traversal",
			params:  gitBuildParams{Repository: "acme/app", CommitSHA: "deadbeef:../subdir"},
			wantErr: true,
		},
		{
			name:    "commit sha with fragment",
			params:  gitBuildParams{Repository: "acme/app", CommitSHA: "deadbeef#evil"},
			wantErr: true,
		},
		{
			name:    "commit sha too short",
			params:  gitBuildParams{Repository: "acme/app", CommitSHA: "dead"},
			wantErr: true,
		},
		{
			name:    "commit sha non-hex",
			params:  gitBuildParams{Repository: "acme/app", CommitSHA: "zzzzzzz"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitBuildParams(tt.params)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBuildGitSolverOptions_EnvSecretGating proves env vars only become a
// BuildKit secret when present.
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

// TestResolveCloneToken covers the credential tiers and, for fork builds, that
// probing and scoping target the repo BuildKit actually clones (base for live
// PRs, fork for fork refs deployed by SHA).
func TestResolveCloneToken(t *testing.T) {
	const base = "acme/app"

	forkPR := gitBuildParams{
		InstallationID: 1,
		Repository:     base,
		ForkRepository: "contributor/app",
		PrNumber:       42,
	}
	sameOwnerForkRef := gitBuildParams{
		InstallationID: 1,
		Repository:     base,
		ForkRepository: "acme/app-fork",
		PrNumber:       0,
	}
	differentOwnerForkRef := gitBuildParams{
		InstallationID: 1,
		Repository:     base,
		ForkRepository: "contributor/app",
		PrNumber:       0,
	}

	t.Run("trusted build gets a token scoped to its own repo without probing", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh}

		token, tokenless, err := w.resolveCloneToken(gitBuildParams{InstallationID: 1, Repository: base}, false)
		require.NoError(t, err)
		require.False(t, tokenless)
		require.NotEmpty(t, token.Token)
		require.Empty(t, gh.probed)
		require.Equal(t, []string{base}, gh.scopedTo)
	})

	t.Run("unauthenticated dev mode is tokenless", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh, allowUnauthenticatedDeployments: true}

		_, tokenless, err := w.resolveCloneToken(gitBuildParams{Repository: base}, true)
		require.NoError(t, err)
		require.True(t, tokenless)
		require.Empty(t, gh.probed)
		require.Empty(t, gh.scopedTo)
	})

	t.Run("fork PR with public base is tokenless and probes the base", func(t *testing.T) {
		gh := &stubGitHub{public: map[string]bool{base: true}}
		w := &Workflow{github: gh}

		_, tokenless, err := w.resolveCloneToken(forkPR, true)
		require.NoError(t, err)
		require.True(t, tokenless)
		require.Equal(t, []string{base}, gh.probed)
	})

	t.Run("fork PR with private base scopes token to the base", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh}

		_, tokenless, err := w.resolveCloneToken(forkPR, true)
		require.NoError(t, err)
		require.False(t, tokenless)
		require.Equal(t, []string{base}, gh.scopedTo)
	})

	t.Run("fork PR probe failure falls back to a base-scoped token", func(t *testing.T) {
		gh := &stubGitHub{probeErr: fmt.Errorf("rate limited")}
		w := &Workflow{github: gh}

		_, tokenless, err := w.resolveCloneToken(forkPR, true)
		require.NoError(t, err)
		require.False(t, tokenless)
		require.Equal(t, []string{base}, gh.scopedTo)
	})

	t.Run("public fork ref is tokenless and probes the fork", func(t *testing.T) {
		gh := &stubGitHub{public: map[string]bool{"contributor/app": true}}
		w := &Workflow{github: gh}

		_, tokenless, err := w.resolveCloneToken(differentOwnerForkRef, true)
		require.NoError(t, err)
		require.True(t, tokenless)
		require.Equal(t, []string{"contributor/app"}, gh.probed)
	})

	t.Run("private same-owner fork ref scopes token to the fork", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh}

		_, tokenless, err := w.resolveCloneToken(sameOwnerForkRef, true)
		require.NoError(t, err)
		require.False(t, tokenless)
		require.Equal(t, []string{"acme/app-fork"}, gh.scopedTo)
	})

	t.Run("casing-drifted own repo is same-owner, not a different installation", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh}

		// ForkRepository arrives from dashboard, while the
		// connected repo carries GitHub's canonical casing (acme). Must not be
		// classified as a different-owner fork and fail terminally.
		casingDrift := gitBuildParams{
			InstallationID: 1,
			Repository:     base,
			ForkRepository: "Acme/app-fork",
			PrNumber:       0,
		}
		_, tokenless, err := w.resolveCloneToken(casingDrift, true)
		require.NoError(t, err)
		require.False(t, tokenless)
		require.Equal(t, []string{"Acme/app-fork"}, gh.scopedTo)
	})

	t.Run("private different-owner fork ref fails fast with a terminal error", func(t *testing.T) {
		gh := &stubGitHub{}
		w := &Workflow{github: gh}

		_, _, err := w.resolveCloneToken(differentOwnerForkRef, true)
		require.Error(t, err)
		require.True(t, restate.IsTerminalError(err))
		require.Contains(t, err.Error(), "contributor/app")
		require.Empty(t, gh.scopedTo)
	})

	t.Run("different-owner fork probe failure is a retryable error", func(t *testing.T) {
		gh := &stubGitHub{probeErr: fmt.Errorf("rate limited")}
		w := &Workflow{github: gh}

		_, _, err := w.resolveCloneToken(differentOwnerForkRef, true)
		require.Error(t, err)
		require.False(t, restate.IsTerminalError(err))
		require.Empty(t, gh.scopedTo)
	})
}

// stubGitHub records visibility probes and token scopes. Embedding the
// interface panics on any other method, asserting resolveCloneToken touches
// nothing else.
type stubGitHub struct {
	githubclient.GitHubClient
	public   map[string]bool
	probeErr error
	probed   []string
	scopedTo []string
}

func (s *stubGitHub) IsRepoPublic(repo string) (bool, error) {
	s.probed = append(s.probed, repo)
	if s.probeErr != nil {
		return false, s.probeErr
	}
	return s.public[repo], nil
}

func (s *stubGitHub) GetScopedInstallationToken(_ int64, repo string, _ map[string]string) (githubclient.InstallationToken, error) {
	s.scopedTo = append(s.scopedTo, repo)
	return githubclient.InstallationToken{Token: "scoped"}, nil
}
