package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestCreateDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	t.Run("create deployment with docker image", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"project.*.create_deployment"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			Project:         setup.Project.Slug,
			App:             "default",
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     "nginx:latest",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 201, res.Status, "expected 201, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
		observed := testutil.Receive(t, creates, 10*time.Second)
		require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID,
			"the id in the response must be the id the create ran under")
		require.Equal(t, "nginx:latest", observed.Request.GetImage().GetImage())
		require.Equal(t, "main", observed.Request.GetImage().GetCommit().GetBranch(),
			"the branch scopes sibling dedup, so an image deploy must still carry it")
	})

	// The CLI built and pushed the image itself. The commit is what it was built
	// from, not something to build again.
	t.Run("create deployment with git commit info", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			ProjectName: "test-git-project",
			Permissions: []string{"project.*.create_deployment"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			Project:         setup.Project.Slug,
			App:             "default",
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     "nginx:latest",
			GitCommit: &openapi.V2DeployGitCommit{
				AuthorAvatarUrl: ptr.P("https://avatar.example.com/johndoe.jpg"),
				AuthorHandle:    ptr.P("johndoe"),
				CommitMessage:   ptr.P("feat: add new feature"),
				CommitSha:       ptr.P("abc123def456"),
				Timestamp:       ptr.P(int64(1704067200000)),
			},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 201, res.Status, "expected 201, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
		observed := testutil.Receive(t, creates, 10*time.Second)
		require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID,
			"the id in the response must be the id the create ran under")
		require.Nil(t, observed.Request.GetGit(), "a pushed image must not be sent as a git build")
		require.Equal(t, "nginx:latest", observed.Request.GetImage().GetImage())
		commit := observed.Request.GetImage().GetCommit()
		require.Equal(t, "main", commit.GetBranch())
		require.Equal(t, "abc123def456", commit.GetCommitSha())
		require.Equal(t, "feat: add new feature", commit.GetCommitMessage())
		require.Equal(t, "johndoe", commit.GetAuthorHandle())
		require.Equal(t, "https://avatar.example.com/johndoe.jpg", commit.GetAuthorAvatarUrl())
		require.Equal(t, int64(1704067200000), commit.GetTimestamp())
	})
}

func TestCreateDeploymentWithWildcardPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		Project:         setup.Project.Slug,
		App:             "default",
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
	require.NotNil(t, res.Body)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID,
		"the id in the response must be the id the create ran under")
}

func TestCreateDeploymentWithSpecificProjectPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	// First create the project/environment setup
	setup := h.CreateTestDeploymentSetup()

	// Now create a root key with project-specific permission
	rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("project.%s.create_deployment", setup.Project.ID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Project:         setup.Project.Slug,
		App:             "default",
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
	require.NotNil(t, res.Body)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID,
		"the id in the response must be the id the create ran under")
}
