package githubapp_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	github "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// recordingClient captures the repository string passed to GetInstallationRepo
// so tests can assert Resolve normalized it before the GitHub lookup.
type recordingClient struct {
	*github.Noop
	gotRepo string
}

func (c *recordingClient) GetInstallationRepo(_ int64, repo string) (*github.RepoInfo, error) {
	c.gotRepo = repo
	return &github.RepoInfo{ID: 1, FullName: "unkeyed/unkey", DefaultBranch: "main", Private: false}, nil
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
