package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_list_domains"
)

type seededEnv struct {
	workspaceID   string
	projectID     string
	projectSlug   string
	appID         string
	appSlug       string
	environmentID string
}

func seedEnvironment(t *testing.T, h *testutil.Harness) seededEnv {
	t.Helper()

	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        randomSlug(),
	})

	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Payments API",
		Slug:        randomSlug(),
	})

	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	return seededEnv{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		projectSlug:   project.Slug,
		appID:         app.ID,
		appSlug:       app.Slug,
		environmentID: environment.ID,
	}
}

// attachDomain adds one domain to the seeded environment, returning the request so
// callers can assert against the generated token and target.
func attachDomain(t *testing.T, h *testutil.Harness, env seededEnv, mutate func(*seed.CreateCustomDomainRequest)) seed.CreateCustomDomainRequest {
	t.Helper()

	req := seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        env.workspaceID,
		ProjectID:          env.projectID,
		AppID:              env.appID,
		EnvironmentID:      env.environmentID,
		Domain:             randomDomain(),
		VerificationStatus: "",
		VerificationToken:  uid.Secure(24),
		TargetCname:        fmt.Sprintf("%s.cname.unkey.local", uid.DNS1035(16)),
		OwnershipVerified:  false,
		CnameVerified:      false,
		VerificationError:  "",
		LastCheckedAt:      0,
	}
	if mutate != nil {
		mutate(&req)
	}

	h.CreateCustomDomain(req)
	return req
}

func makeRequest(env seededEnv) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Search:      nil,
	}
}

// randomSlug produces a unique lowercase slug. Parallel packages share one
// database, so a hardcoded slug would collide across runs.
func randomSlug() string {
	return uid.DNS1035(16)
}

// randomDomain produces a unique hostname that satisfies the spec's FQDN pattern.
func randomDomain() string {
	return uid.DNS1035(16) + ".example.com"
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
