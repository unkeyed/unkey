package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
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
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          randomSlug(),
		DefaultBranch: "main",
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

// makeRequest builds a create request addressing the seeded environment by id.
func makeRequest(env seededEnv, domain string) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Domain:      domain,
	}
}

// randomSlug produces a lowercase-dashed value. Parallel packages share one
// database, so a hardcoded slug would collide across runs.
func randomSlug() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

// randomDomain produces a unique hostname that satisfies the spec's FQDN
// pattern. The label cannot start with a digit-only accident of uid, so it is
// prefixed.
func randomDomain() string {
	return fmt.Sprintf("d%s.example.com", strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "")))
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
