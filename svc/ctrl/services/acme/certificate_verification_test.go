package acme

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestVerifyCertificateRejectsMissingBearerToken guarantees unauthenticated
// callers cannot use the ACME verification endpoint to enumerate domains or
// populate ctrl caches.
func TestVerifyCertificateRejectsMissingBearerToken(t *testing.T) {
	svc := New(Config{
		DB:             nil,
		DomainCache:    nil,
		ChallengeCache: nil,
		Bearer:         "ctrl-token",
	})
	req := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{
		Domain: "api.example.com",
		Token:  "challenge-token",
	})

	_, err := svc.VerifyCertificate(context.Background(), req)

	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// TestVerifyCertificateRejectsInvalidBearerToken guarantees network access to
// ctrl is not enough to invoke ACME verification without the preshared token.
func TestVerifyCertificateRejectsInvalidBearerToken(t *testing.T) {
	svc := New(Config{
		DB:             nil,
		DomainCache:    nil,
		ChallengeCache: nil,
		Bearer:         "ctrl-token",
	})
	req := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{
		Domain: "api.example.com",
		Token:  "challenge-token",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := svc.VerifyCertificate(context.Background(), req)

	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// HTTP-01 answers come from the verified row a hostname resolves to, in either
// domain table, so a pending competing claim can never answer the CA. Runs
// against real MySQL to cover the query.
func TestVerifyCertificateAnswersForVerifiedDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database, verify := newVerifyService(t)
	now := time.Now().UnixMilli()
	insertChallenge := func(workspaceID, domainID, token, authorization string) {
		insertPendingChallenge(t, database, workspaceID, domainID, token, authorization)
	}

	// The pending deploy claim was created later and holds a challenge with the
	// same token, so only the verification filter keeps it from answering.
	pendingWS := uid.New(uid.WorkspacePrefix)
	pending := uid.New(uid.DomainPrefix)
	require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                    pending,
		WorkspaceID:           pendingWS,
		ProjectID:             uid.New(uid.ProjectPrefix),
		AppID:                 uid.New(uid.AppPrefix),
		EnvironmentID:         uid.New(uid.EnvironmentPrefix),
		Domain:                "portal.example.com",
		ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus:    db.CustomDomainsVerificationStatusPending,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             now,
	}))
	insertChallenge(pendingWS, pending, "tok", "auth-pending")

	tenantWS := uid.New(uid.WorkspacePrefix)
	verified := uid.New(uid.PortalDomainPrefix)
	require.NoError(t, database.InsertPortalDomain(ctx, db.InsertPortalDomainParams{
		ID:                    verified,
		WorkspaceID:           tenantWS,
		PortalID:              uid.New(uid.PortalPrefix),
		Domain:                "portal.example.com",
		VerificationStatus:    db.PortalDomainsVerificationStatusVerified,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             now - 1000,
	}))
	insertChallenge(tenantWS, verified, "tok", "auth-verified")

	res, err := verify("portal.example.com", "tok")
	require.NoError(t, err)
	require.Equal(t, "auth-verified", res.Msg.GetAuthorization())

	_, err = verify("unknown.example.com", "tok")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// Frontline forwards ACME requests for pending rows too, so a probe before
// verification must not cache the miss and 404 the CA's first real request.
func TestVerifyCertificateAnswersOnceAProbedDomainIsVerified(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database, verify := newVerifyService(t)

	workspaceID := uid.New(uid.WorkspacePrefix)
	domainID := uid.New(uid.PortalDomainPrefix)
	require.NoError(t, database.InsertPortalDomain(ctx, db.InsertPortalDomainParams{
		ID:                    domainID,
		WorkspaceID:           workspaceID,
		PortalID:              uid.New(uid.PortalPrefix),
		Domain:                "probed.example.com",
		VerificationStatus:    db.PortalDomainsVerificationStatusPending,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             time.Now().UnixMilli(),
	}))
	insertPendingChallenge(t, database, workspaceID, domainID, "tok", "auth-probed")

	_, err := verify("probed.example.com", "tok")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	require.NoError(t, database.UpdatePortalDomainVerificationStatus(ctx, db.UpdatePortalDomainVerificationStatusParams{
		ID:                 domainID,
		VerificationStatus: db.PortalDomainsVerificationStatusVerified,
		UpdatedAt:          sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))

	res, err := verify("probed.example.com", "tok")
	require.NoError(t, err)
	require.Equal(t, "auth-probed", res.Msg.GetAuthorization())
}

// newVerifyService wires VerifyCertificate to an isolated MySQL with real
// caches and returns an authenticated caller for it.
func newVerifyService(t *testing.T) (db.Database, func(domain, token string) (*connect.Response[ctrlv1.VerifyCertificateResponse], error)) {
	t.Helper()

	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	clk := clock.New()
	domainCache, err := cache.New(cache.Config[string, db.FindVerifiedDomainByHostnameRow]{Fresh: time.Minute, Stale: time.Minute, MaxSize: 100, Resource: "test_domains", Clock: clk})
	require.NoError(t, err)
	t.Cleanup(domainCache.Close)
	challengeCache, err := cache.New(cache.Config[string, db.AcmeChallenge]{Fresh: time.Minute, Stale: time.Minute, MaxSize: 100, Resource: "test_challenges", Clock: clk})
	require.NoError(t, err)
	t.Cleanup(challengeCache.Close)

	svc := New(Config{DB: database, DomainCache: domainCache, ChallengeCache: challengeCache, Bearer: "ctrl-token"})
	return database, func(domain, token string) (*connect.Response[ctrlv1.VerifyCertificateResponse], error) {
		req := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{Domain: domain, Token: token})
		req.Header().Set("Authorization", "Bearer ctrl-token")
		return svc.VerifyCertificate(context.Background(), req)
	}
}

func insertPendingChallenge(t *testing.T, database db.Database, workspaceID, domainID, token, authorization string) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, database.InsertAcmeChallenge(context.Background(), db.InsertAcmeChallengeParams{
		WorkspaceID:   workspaceID,
		DomainID:      domainID,
		Token:         token,
		Authorization: authorization,
		Status:        db.AcmeChallengesStatusPending,
		ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     0,
	}))
}
