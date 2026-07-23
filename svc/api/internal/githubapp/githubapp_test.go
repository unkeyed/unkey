package githubapp_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	github "github.com/unkeyed/unkey/pkg/github"
)

// recordingClient captures the repository string passed to GetInstallationRepo
// so tests can assert Resolve normalized it before the GitHub lookup.
type recordingClient struct {
	*github.Noop
	gotRepo string
}

func (c *recordingClient) GetInstallationRepo(_ int64, repo string) (*github.RepoInfo, error) {
	c.gotRepo = repo
	return &github.RepoInfo{ID: 1, FullName: "unkeyed/unkey", DefaultBranch: "main"}, nil
}

func TestResolveNormalizesRepository(t *testing.T) {
	inputs := []string{
		"unkeyed/unkey",
		"  unkeyed/unkey  ",
		"unkeyed/unkey/",
		"unkeyed/unkey.git",
		"github.com/unkeyed/unkey",
		"https://github.com/unkeyed/unkey",
		"https://github.com/unkeyed/unkey.git",
		"https://github.com/unkeyed/unkey/",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			client := &recordingClient{Noop: github.NewNoop()}
			resolved, err := githubapp.Resolve(client, "unkey-app", []int64{12345}, in)
			require.NoError(t, err)
			require.Equal(t, "unkeyed/unkey", client.gotRepo, "repository should be normalized before the GitHub lookup")
			require.Equal(t, int64(12345), resolved.InstallationID)
		})
	}
}

// scriptedClient returns a per-installation result from GetInstallationRepo so
// tests can drive Resolve's continue-past-error loop across many installations.
type scriptedClient struct {
	*github.Noop
	results map[int64]repoResult
}

type repoResult struct {
	info *github.RepoInfo
	err  error
}

func (c *scriptedClient) GetInstallationRepo(installationID int64, _ string) (*github.RepoInfo, error) {
	r := c.results[installationID]
	return r.info, r.err
}

func TestResolveInstallationLoop(t *testing.T) {
	const repo = "unkeyed/kebap"
	granted := &github.RepoInfo{ID: 1, FullName: "unkeyed/kebap", DefaultBranch: "main"}

	t.Run("error then grant resolves via the granting installation", func(t *testing.T) {
		client := &scriptedClient{
			Noop: github.NewNoop(),
			results: map[int64]repoResult{
				111: {err: errors.New("boom-a")},
				222: {info: granted},
			},
		}
		resolved, err := githubapp.Resolve(client, "unkey-app", []int64{111, 222}, repo)
		require.NoError(t, err)
		require.Equal(t, int64(222), resolved.InstallationID)
		require.Equal(t, "unkeyed/kebap", resolved.Repository.FullName)
	})

	t.Run("all installations error is inconclusive (503) and joins detail", func(t *testing.T) {
		client := &scriptedClient{
			Noop: github.NewNoop(),
			results: map[int64]repoResult{
				111: {err: errors.New("boom-a")},
				222: {err: errors.New("boom-b")},
			},
		}
		_, err := githubapp.Resolve(client, "unkey-app", []int64{111, 222}, repo)
		require.Error(t, err)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
		require.Contains(t, err.Error(), "boom-a")
		require.Contains(t, err.Error(), "boom-b")
	})

	t.Run("all installations clean deny is not accessible (412)", func(t *testing.T) {
		client := &scriptedClient{
			Noop: github.NewNoop(),
			results: map[int64]repoResult{
				111: {},
				222: {},
			},
		}
		_, err := githubapp.Resolve(client, "unkey-app", []int64{111, 222}, repo)
		require.Error(t, err)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Precondition.PreconditionFailed.URN(), code)
	})

	t.Run("clean deny mixed with error is inconclusive (503)", func(t *testing.T) {
		client := &scriptedClient{
			Noop: github.NewNoop(),
			results: map[int64]repoResult{
				111: {},
				222: {err: errors.New("boom-b")},
			},
		}
		_, err := githubapp.Resolve(client, "unkey-app", []int64{111, 222}, repo)
		require.Error(t, err)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
	})
}
