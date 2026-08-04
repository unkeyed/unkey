package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
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

// insertCustomDomain writes the row the real ctrl service would have written. The
// mock stands in for ctrl, so nothing else creates it and the handler's reads would
// see no state.
func insertCustomDomain(t *testing.T, h *testutil.Harness, req *ctrlv1.AddCustomDomainRequest, domainID string) {
	t.Helper()

	_, err := h.DB.RW().ExecContext(context.Background(), `
		INSERT INTO custom_domains
			(id, workspace_id, project_id, app_id, environment_id, domain,
			 challenge_type, verification_status, verification_token, target_cname, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'HTTP-01', 'pending', ?, ?, ?)
	`,
		domainID,
		req.GetWorkspaceId(),
		req.GetProjectId(),
		req.GetAppId(),
		req.GetEnvironmentId(),
		req.GetDomain(),
		uid.New("vt"),
		uid.DNS1035(16)+".cname.unkey.local",
		time.Now().UnixMilli(),
	)
	require.NoError(t, err)
}

// setCustomDomainAllowance overrides the seeder's generous default so a test can
// drive the plan allowance gate.
func setCustomDomainAllowance(t *testing.T, h *testutil.Harness, workspaceID string, allowance uint32) {
	t.Helper()

	_, err := h.DB.RW().ExecContext(context.Background(),
		"UPDATE `limits` SET custom_domains_max = ? WHERE workspace_id = ?", allowance, workspaceID)
	require.NoError(t, err)
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
