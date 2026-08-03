package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

type seededDomain struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	domainID      string
	domain        string
	token         string
	targetCname   string
}

// seedDomain creates a project, app, and environment, then attaches one domain to
// it. Pass a mutator to change the row before it is inserted.
func seedDomain(t *testing.T, h *testutil.Harness, mutate func(*seed.CreateCustomDomainRequest)) seededDomain {
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

	req := seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		EnvironmentID:      environment.ID,
		Domain:             randomDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
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

	row := h.CreateCustomDomain(req)

	return seededDomain{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
		domainID:      row.ID,
		domain:        row.Domain,
		token:         row.VerificationToken,
		targetCname:   row.TargetCname,
	}
}

// randomSlug produces a lowercase-dashed value. Parallel packages share one
// database, so a hardcoded slug would collide across runs.
func randomSlug() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

// randomDomain produces a unique hostname satisfying the FQDN pattern. The label is
// prefixed so it cannot start with a digit-only accident of uid.
func randomDomain() string {
	return fmt.Sprintf("d%s.example.com", strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "")))
}

// randomApexDomain produces a unique zone apex, which routes through an alias rather
// than a CNAME.
func randomApexDomain() string {
	return fmt.Sprintf("d%s.com", strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "")))
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
