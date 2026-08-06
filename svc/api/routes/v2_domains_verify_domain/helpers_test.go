package handler_test

import (
	"fmt"
	"net/http"
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
}

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

	// Failed is the state this endpoint exists for, so it is the default here.
	req := seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		EnvironmentID:      environment.ID,
		Domain:             randomDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusFailed,
		VerificationToken:  uid.Secure(24),
		TargetCname:        fmt.Sprintf("%s.cname.unkey.local", uid.DNS1035(16)),
		OwnershipVerified:  false,
		CnameVerified:      false,
		VerificationError:  "domain verification timed out after 24 hours",
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
	}
}

// verifiedDomain mutates a seed into the verified state the 412 gate rejects.
func verifiedDomain(req *seed.CreateCustomDomainRequest) {
	req.VerificationStatus = db.CustomDomainsVerificationStatusVerified
	req.OwnershipVerified = true
	req.CnameVerified = true
	req.VerificationError = ""
}

// Parallel packages share one database, so a hardcoded slug would collide across runs.
func randomSlug() string {
	return uid.DNS1035(16)
}

// randomDomain keeps names unique across runs sharing one database.
func randomDomain() string {
	return uid.DNS1035(16) + ".example.com"
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
