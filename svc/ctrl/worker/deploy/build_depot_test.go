package deploy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
)

func TestIsStaleGitRefError(t *testing.T) {
	for name, stderr := range map[string]string{
		"git 2.53": `fatal: Not a valid object name 89abcdef0123456789abcdef0123456789abcdef^{commit}
From https://github.com/acme/app
 * [new branch]      release    -> origin/release
error: some local refs could not be updated; try running
 'git remote prune origin' to remove any old, conflicting branches`,
		"cannot lock ref": `fatal: Not a valid object name 0123456789abcdef0123456789abcdef01234567^{commit}
error: cannot lock ref 'refs/remotes/origin/release': 'refs/remotes/origin/release/v1' exists; cannot create 'refs/remotes/origin/release'
From https://github.com/acme/app
 ! [new branch]        release    -> origin/release  (unable to update local ref)
error: some local refs could not be updated; try running
 'git remote prune origin' to remove any old, conflicting branches`,
	} {
		err := errors.New("failed to fetch remote https://github.com/acme/app.git: exit status 1: git stderr:\n" + stderr)
		require.True(t, isStaleGitRefError(err), name)
	}

	for _, other := range []string{
		`process "/bin/sh -c npm run build" did not complete successfully: exit code: 1`,
		`failed to fetch remote https://github.com/acme/app.git: git stderr:
fatal: could not read Username for 'https://github.com': terminal prompts disabled`,
	} {
		require.False(t, isStaleGitRefError(errors.New(other)), other)
	}
}

func TestClassifySolveErrorStaleGitRef(t *testing.T) {
	gitFailure := errors.New("failed to fetch remote https://github.com/acme/app.git: exit status 1: git stderr:\n" +
		"error: some local refs could not be updated; try running\n 'git remote prune origin' to remove any old, conflicting branches")
	spoofed := errors.New(`process "/bin/sh -c echo 'some local refs could not be updated'; false" did not complete successfully: exit code: 1`)

	trusted := gitBuildParams{Repository: "acme/app"}
	forkPR := gitBuildParams{Repository: "acme/app", PrNumber: 7}
	forkRef := gitBuildParams{Repository: "acme/app", ForkRepository: "contributor/app"}

	err := classifySolveError(gitFailure, trusted)
	require.ErrorIs(t, err, errStaleGitRef)
	require.False(t, restate.IsTerminalError(err), "a stale ref must be retried after the cache reset")

	for name, params := range map[string]gitBuildParams{"fork PR": forkPR, "fork ref": forkRef} {
		for _, solveErr := range []error{gitFailure, spoofed} {
			err := classifySolveError(solveErr, params)
			require.NotErrorIs(t, err, errStaleGitRef, name)
			require.True(t, restate.IsTerminalError(err), name)
		}
	}
}

type resetRecorder struct {
	corev1connect.UnimplementedProjectServiceHandler
	projectIDs []string
	auth       []string
}

func (r *resetRecorder) ResetProject(_ context.Context, req *connect.Request[corev1.ResetProjectRequest]) (*connect.Response[corev1.ResetProjectResponse], error) {
	r.projectIDs = append(r.projectIDs, req.Msg.GetProjectId())
	r.auth = append(r.auth, req.Header().Get("Authorization"))
	return connect.NewResponse(&corev1.ResetProjectResponse{}), nil
}

func TestResetDepotProjectCache(t *testing.T) {
	recorder := &resetRecorder{}
	mux := http.NewServeMux()
	mux.Handle(corev1connect.NewProjectServiceHandler(recorder))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	w := &Workflow{
		buildConfig:    BuildConfig{Backend: BuildBackendDepot, Depot: DepotConfig{APIUrl: server.URL}},
		registryConfig: RegistryConfig{Password: "depot_org_token"},
	}

	require.NoError(t, w.resetDepotProjectCache(context.Background(), "depot_project_1"))
	require.Equal(t, []string{"depot_project_1"}, recorder.projectIDs)
	require.Equal(t, []string{"Bearer depot_org_token"}, recorder.auth)
}
