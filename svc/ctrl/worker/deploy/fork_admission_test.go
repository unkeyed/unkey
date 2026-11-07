package deploy

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestForkRepositoryRejection proves a caller-supplied fork repository may only
// be the app's connected repository or a same-owner fork of it. Any other
// same-owner repository is refused before the app's installation credential
// could reach it.
func TestForkRepositoryRejection(t *testing.T) {
	const base = "acme/app"

	//nolint:exhaustruct
	target := db.FindDeployTargetRow{
		AppID:                    "app_1",
		GithubRepositoryFullName: sql.NullString{String: base, Valid: true},
		GithubInstallationID:     sql.NullInt64{Int64: 1, Valid: true},
	}

	t.Run("empty fork repository is allowed", func(t *testing.T) {
		w := &Workflow{github: &stubGitHub{}}
		require.Nil(t, w.forkRepositoryRejection(target, ""))
	})

	t.Run("the connected repository is allowed", func(t *testing.T) {
		w := &Workflow{github: &stubGitHub{}}
		require.Nil(t, w.forkRepositoryRejection(target, "Acme/App"))
	})

	t.Run("a different-owner repository is left to resolveCloneToken", func(t *testing.T) {
		w := &Workflow{github: &stubGitHub{}}
		require.Nil(t, w.forkRepositoryRejection(target, "contributor/app"))
	})

	t.Run("a fork whose parent is the connected repository is allowed", func(t *testing.T) {
		gh := &stubGitHub{forkInfo: map[string]*githubclient.RepoForkInfo{
			"acme/app-fork": {Fork: true, ParentFullName: base, SourceFullName: base},
		}}
		w := &Workflow{github: gh}
		require.Nil(t, w.forkRepositoryRejection(target, "acme/app-fork"))
	})

	t.Run("a fork of a fork in the same network is allowed", func(t *testing.T) {
		gh := &stubGitHub{forkInfo: map[string]*githubclient.RepoForkInfo{
			"acme/app-fork": {Fork: true, ParentFullName: "acme/app-mid", SourceFullName: base},
		}}
		w := &Workflow{github: gh}
		require.Nil(t, w.forkRepositoryRejection(target, "acme/app-fork"))
	})

	t.Run("a same-owner repository that is not a fork is refused", func(t *testing.T) {
		gh := &stubGitHub{forkInfo: map[string]*githubclient.RepoForkInfo{
			"acme/internal-secrets": {Fork: false},
		}}
		w := &Workflow{github: gh}
		rejection := w.forkRepositoryRejection(target, "acme/internal-secrets")
		require.NotNil(t, rejection)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_FORK_REPOSITORY_NOT_RELATED, rejection.Outcome)
	})

	t.Run("a same-owner fork of an unrelated repository is refused", func(t *testing.T) {
		gh := &stubGitHub{forkInfo: map[string]*githubclient.RepoForkInfo{
			"acme/app-fork": {Fork: true, ParentFullName: "other/forked", SourceFullName: "other/forked"},
		}}
		w := &Workflow{github: gh}
		rejection := w.forkRepositoryRejection(target, "acme/app-fork")
		require.NotNil(t, rejection)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_FORK_REPOSITORY_NOT_RELATED, rejection.Outcome)
	})

	t.Run("a same-owner repository the installation cannot read is refused", func(t *testing.T) {
		w := &Workflow{github: &stubGitHub{}}
		rejection := w.forkRepositoryRejection(target, "acme/app-fork")
		require.NotNil(t, rejection)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_FORK_REPOSITORY_NOT_RELATED, rejection.Outcome)
	})

	t.Run("a fork metadata lookup failure is refused", func(t *testing.T) {
		gh := &stubGitHub{forkInfoErr: errors.New("rate limited")}
		w := &Workflow{github: gh}
		rejection := w.forkRepositoryRejection(target, "acme/app-fork")
		require.NotNil(t, rejection)
		require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_FORK_REPOSITORY_NOT_RELATED, rejection.Outcome)
	})
}
