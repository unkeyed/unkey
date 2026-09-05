package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

func TestListDomainsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.read_domain")
	headers := authHeaders(rootKey)

	// A second app under the same project, so the "environment belongs to another
	// app" case exercises a real environment reached through the wrong parent.
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: env.workspaceID,
		ProjectID:   env.projectID,
		Name:        "Billing API",
		Slug:        randomSlug(),
	})

	otherWorkspace := h.CreateWorkspace()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: otherWorkspace.ID,
		Name:        "Other Workspace Project",
		Slug:        randomSlug(),
	})
	otherWorkspaceApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: otherWorkspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "Other Workspace App",
		Slug:        randomSlug(),
	})
	otherWorkspaceEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: otherWorkspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherWorkspaceApp.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	// A domain in the other workspace, so a leak would show up in the body rather
	// than only in the status code.
	otherWorkspaceDomain := h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        otherWorkspace.ID,
		ProjectID:          otherProject.ID,
		AppID:              otherWorkspaceApp.ID,
		EnvironmentID:      otherWorkspaceEnv.ID,
		Domain:             randomDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  "",
		TargetCname:        "",
		OwnershipVerified:  true,
		CnameVerified:      true,
		VerificationError:  "",
		LastCheckedAt:      0,
	})

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{
			name: "nonexistent environment",
			req:  handler.Request{Project: env.projectID, App: env.appID, Environment: uid.New(uid.EnvironmentPrefix), Search: nil},
		},
		{
			name: "nonexistent app",
			req:  handler.Request{Project: env.projectID, App: uid.New(uid.AppPrefix), Environment: env.environmentID, Search: nil},
		},
		{
			name: "nonexistent project",
			req:  handler.Request{Project: uid.New(uid.ProjectPrefix), App: env.appID, Environment: env.environmentID, Search: nil},
		},
		{
			name: "environment under the wrong app",
			req:  handler.Request{Project: env.projectID, App: otherApp.ID, Environment: env.environmentID, Search: nil},
		},
		{
			name: "app under the wrong project",
			req:  handler.Request{Project: otherProject.ID, App: env.appID, Environment: env.environmentID, Search: nil},
		},
		{
			name: "environment in another workspace",
			req:  handler.Request{Project: otherProject.ID, App: otherWorkspaceApp.ID, Environment: otherWorkspaceEnv.ID, Search: nil},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", res.Body.Error.Type)
			require.NotContains(t, res.RawBody, otherWorkspaceDomain.ID, "cross-workspace lookup leaked a domain id: %s", res.RawBody)
		})
	}
}
