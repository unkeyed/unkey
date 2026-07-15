//go:build integration

package integration

import (
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/customdomain"
)

const testBearer = "test-bearer"

// TestDeleteCustomDomain_DoesNotDeleteOtherWorkspaceFrontlineRoute is a
// regression test for ENG-2950: cross-tenant deletion of another workspace's
// frontline route via DeleteCustomDomain.
//
// frontline_routes enforces UNIQUE(fully_qualified_domain_name), so exactly one
// route exists per FQDN, owned by whichever project verified the domain.
// custom_domains is only UNIQUE(workspace_id, domain), so workspace A can create
// a pending custom_domains row for an FQDN that workspace B already verified and
// routes. Deleting A's own row must not delete B's live frontline route.
func TestDeleteCustomDomain_DoesNotDeleteOtherWorkspaceFrontlineRoute(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	now := time.Now().UnixMilli()

	const fqdn = "victim.example.com"

	svc := customdomain.New(customdomain.Config{
		Database: h.DB,
		Bearer:   testBearer,
	})

	// --- Victim workspace B: verified domain + live frontline route ---
	victimWS := h.Seed.Resources.RootWorkspace.ID
	victimProject := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: victimWS,
		Name:        "victim-project",
		Slug:        uid.New("slug"),
	})
	victimApp := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   victimWS,
		ProjectID:     victimProject.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})
	victimEnv := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    victimWS,
		ProjectID:      victimProject.ID,
		AppID:          victimApp.ID,
		Slug:           "production",
		SentinelConfig: []byte("{}"),
	})

	require.NoError(t, h.DB.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        victimWS,
		ProjectID:          victimProject.ID,
		AppID:              victimApp.ID,
		EnvironmentID:      victimEnv.ID,
		Domain:             fqdn,
		ChallengeType:      db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  uid.Secure(24),
		TargetCname:        "victim-target.cname.unkey.com",
		CreatedAt:          now,
	}))

	frontlineRouteID := uid.New(uid.FrontlineRoutePrefix)
	require.NoError(t, h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       frontlineRouteID,
		ProjectID:                victimProject.ID,
		AppID:                    victimApp.ID,
		DeploymentID:             "",
		EnvironmentID:            victimEnv.ID,
		FullyQualifiedDomainName: fqdn,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
	}))

	// --- Attacker workspace A: pending custom_domains row for the same FQDN ---
	attackerWS := h.Seed.Resources.UserWorkspace.ID
	attackerProject := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: attackerWS,
		Name:        "attacker-project",
		Slug:        uid.New("slug"),
	})
	attackerApp := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   attackerWS,
		ProjectID:     attackerProject.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})
	attackerEnv := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    attackerWS,
		ProjectID:      attackerProject.ID,
		AppID:          attackerApp.ID,
		Slug:           "production",
		SentinelConfig: []byte("{}"),
	})

	attackerDomainID := uid.New(uid.DomainPrefix)
	require.NoError(t, h.DB.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                 attackerDomainID,
		WorkspaceID:        attackerWS,
		ProjectID:          attackerProject.ID,
		AppID:              attackerApp.ID,
		EnvironmentID:      attackerEnv.ID,
		Domain:             fqdn,
		ChallengeType:      db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
		VerificationToken:  uid.Secure(24),
		TargetCname:        "attacker-target.cname.unkey.com",
		CreatedAt:          now,
	}))

	// --- Attack: A deletes its own pending row for the victim's FQDN ---
	req := connect.NewRequest(&ctrlv1.DeleteCustomDomainRequest{
		WorkspaceId: attackerWS,
		ProjectId:   attackerProject.ID,
		Domain:      fqdn,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.DeleteCustomDomain(ctx, req)
	require.NoError(t, err)

	// The victim's live frontline route must still exist and be unchanged.
	route, err := h.DB.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.NoError(t, err, "victim frontline route must not be deleted")
	require.Equal(t, frontlineRouteID, route.ID)
	require.Equal(t, victimProject.ID, route.ProjectID)

	// The attacker's own custom_domains row must be gone.
	_, err = h.DB.FindCustomDomainById(ctx, attackerDomainID)
	require.True(t, db.IsNotFound(err), "attacker custom domain row should be deleted")
}

// TestDeleteCustomDomain_DeletesOwnFrontlineRoute verifies the normal path still
// works: a workspace deleting its own verified domain removes its frontline route.
func TestDeleteCustomDomain_DeletesOwnFrontlineRoute(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	now := time.Now().UnixMilli()

	const fqdn = "owned.example.com"

	svc := customdomain.New(customdomain.Config{
		Database: h.DB,
		Bearer:   testBearer,
	})

	ws := h.Seed.Resources.UserWorkspace.ID
	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: ws,
		Name:        "owner-project",
		Slug:        uid.New("slug"),
	})
	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   ws,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})
	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    ws,
		ProjectID:      project.ID,
		AppID:          app.ID,
		Slug:           "production",
		SentinelConfig: []byte("{}"),
	})

	domainID := uid.New(uid.DomainPrefix)
	require.NoError(t, h.DB.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                 domainID,
		WorkspaceID:        ws,
		ProjectID:          project.ID,
		AppID:              app.ID,
		EnvironmentID:      env.ID,
		Domain:             fqdn,
		ChallengeType:      db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  uid.Secure(24),
		TargetCname:        "owner-target.cname.unkey.com",
		CreatedAt:          now,
	}))
	require.NoError(t, h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             "",
		EnvironmentID:            env.ID,
		FullyQualifiedDomainName: fqdn,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
	}))

	req := connect.NewRequest(&ctrlv1.DeleteCustomDomainRequest{
		WorkspaceId: ws,
		ProjectId:   project.ID,
		Domain:      fqdn,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.DeleteCustomDomain(ctx, req)
	require.NoError(t, err)

	_, err = h.DB.FindFrontlineRouteByFQDN(ctx, fqdn)
	require.True(t, db.IsNotFound(err), "owner's frontline route should be deleted")

	_, err = h.DB.FindCustomDomainById(ctx, domainID)
	require.True(t, db.IsNotFound(err), "owner's custom domain row should be deleted")
}
