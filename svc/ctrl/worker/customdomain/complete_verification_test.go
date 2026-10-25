package customdomain

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestCompleteVerification(t *testing.T) {
	ctx := context.Background()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	svc := New(Config{DB: database, CnameDomain: "cname.unkey.local"})
	now := time.Now().UnixMilli()

	claim := func(workspaceID, fqdn string, status db.CustomDomainsVerificationStatus) db.CustomDomain {
		t.Helper()
		project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspaceID,
			Name:        "project",
			Slug:        slugFrom(uid.New(uid.ProjectPrefix)),
		})
		app := seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: workspaceID,
			ProjectID:   project.ID,
			Name:        "app",
			Slug:        slugFrom(uid.New(uid.AppPrefix)),
		})
		env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspaceID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "production",
			Kind:        mysqltype.EnvironmentKindProduction,
		})
		domainID := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
			ID:                    domainID,
			WorkspaceID:           workspaceID,
			ProjectID:             project.ID,
			AppID:                 app.ID,
			EnvironmentID:         env.ID,
			Domain:                fqdn,
			ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    status,
			VerificationToken:     uid.Secure(24),
			TargetCname:           uid.DNS1035(16) + ".unkey-dns.com",
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             now,
		}))
		dom, err := database.FindCustomDomainById(ctx, domainID)
		require.NoError(t, err)
		return dom
	}
	newFQDN := func() string { return uid.DNS1035(12) + ".example.com" }
	workspaceID := seeder.Resources.UserWorkspace.ID

	// The step can commit and still fail before Restate journals the result.
	// The retry must find its own route and challenge instead of failing on
	// the unique FQDN and domain_id for the rest of the invocation.
	t.Run("retry after commit", func(t *testing.T) {
		dom := claim(workspaceID, newFQDN(), db.CustomDomainsVerificationStatusVerifying)

		for range 2 {
			outcome, err := svc.completeVerification(ctx, dom, false, now)
			require.NoError(t, err)
			require.Equal(t, verifyOutcomeRouted, outcome)
		}

		route, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
		require.NoError(t, err)
		require.Equal(t, dom.AppID, route.AppID)
		got, err := database.FindCustomDomainById(ctx, dom.ID)
		require.NoError(t, err)
		require.Equal(t, db.CustomDomainsVerificationStatusVerified, got.VerificationStatus)
	})

	// A route held by an app with no verified claim of its own must not be
	// taken, and the losing domain must not be marked verified.
	t.Run("route owned by another app", func(t *testing.T) {
		fqdn := newFQDN()
		owner := claim(workspaceID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err := svc.completeVerification(ctx, owner, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeRouted, outcome)
		require.NoError(t, database.UpdateCustomDomainVerificationStatus(ctx, db.UpdateCustomDomainVerificationStatusParams{
			ID:                 owner.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusPending,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		}))

		other := claim(seeder.CreateWorkspace(ctx).ID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err = svc.completeVerification(ctx, other, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeConflict, outcome)

		route, err := database.FindFrontlineRouteByFQDN(ctx, fqdn)
		require.NoError(t, err)
		require.Equal(t, owner.AppID, route.AppID, "the existing route must stay with its owner")
		got, err := database.FindCustomDomainById(ctx, other.ID)
		require.NoError(t, err)
		require.Equal(t, db.CustomDomainsVerificationStatusVerifying, got.VerificationStatus)
	})

	// DeleteCustomDomain can land between the DNS check and this step. Writing
	// anyway would leave a route and challenge for a domain that is gone.
	t.Run("domain deleted", func(t *testing.T) {
		dom := claim(workspaceID, newFQDN(), db.CustomDomainsVerificationStatusVerifying)
		require.NoError(t, database.DeleteCustomDomainByID(ctx, dom.ID))

		outcome, err := svc.completeVerification(ctx, dom, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeDeleted, outcome)

		_, err = database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
		require.True(t, db.IsNotFound(err), "a deleted domain must not get a route")
	})

	// The DNS check passed on CNAME alone before another workspace verified the
	// domain. Without TXT proof the later claim must not take the domain over.
	t.Run("contested without TXT proof", func(t *testing.T) {
		fqdn := newFQDN()
		previous := claim(seeder.CreateWorkspace(ctx).ID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err := svc.completeVerification(ctx, previous, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeRouted, outcome)

		next := claim(seeder.CreateWorkspace(ctx).ID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err = svc.completeVerification(ctx, next, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeConflict, outcome)

		route, err := database.FindFrontlineRouteByFQDN(ctx, fqdn)
		require.NoError(t, err)
		require.Equal(t, previous.AppID, route.AppID)
		got, err := database.FindCustomDomainById(ctx, previous.ID)
		require.NoError(t, err)
		require.Equal(t, db.CustomDomainsVerificationStatusVerified, got.VerificationStatus)
	})

	// A workspace that proves ownership of a domain another workspace verified
	// takes it over: the old claim fails and the route moves.
	t.Run("takeover from verified claim in another workspace", func(t *testing.T) {
		fqdn := newFQDN()
		previous := claim(seeder.CreateWorkspace(ctx).ID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err := svc.completeVerification(ctx, previous, false, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeRouted, outcome)

		next := claim(seeder.CreateWorkspace(ctx).ID, fqdn, db.CustomDomainsVerificationStatusVerifying)
		outcome, err = svc.completeVerification(ctx, next, true, now)
		require.NoError(t, err)
		require.Equal(t, verifyOutcomeRouted, outcome)

		route, err := database.FindFrontlineRouteByFQDN(ctx, fqdn)
		require.NoError(t, err)
		require.Equal(t, next.AppID, route.AppID)
		got, err := database.FindCustomDomainById(ctx, previous.ID)
		require.NoError(t, err)
		require.Equal(t, db.CustomDomainsVerificationStatusFailed, got.VerificationStatus)
	})
}

func slugFrom(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
