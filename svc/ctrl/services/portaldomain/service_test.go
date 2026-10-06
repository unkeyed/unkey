package portaldomain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	testBearer      = "test-token"
	testCnameDomain = "portal.unkey.local"
)

func TestAddPortalDomainInsertsRowAndAuditLog(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	res, err := svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(res.Msg.GetTargetCname(), "."+testCnameDomain), res.Msg.GetTargetCname())
	require.NotEmpty(t, res.Msg.GetVerificationToken())

	stored, err := f.database.FindPortalDomainById(ctx, res.Msg.GetDomainId())
	require.NoError(t, err)
	require.Equal(t, f.workspaceID, stored.WorkspaceID)
	require.Equal(t, f.portalID, stored.PortalID)
	require.Equal(t, f.domain, stored.Domain)
	require.Equal(t, res.Msg.GetTargetCname(), stored.TargetCname)
	require.Equal(t, db.PortalDomainsVerificationStatusPending, stored.VerificationStatus)
	require.True(t, stored.InvocationID.Valid, "the invocation id must be stored so delete can cancel it")

	logged := f.findAuditEvent(t, f.workspaceID, auditlog.PortalDomainCreateEvent)
	require.Equal(t, "user_test", logged.Actor.ID)
	require.Len(t, logged.Targets, 1)
	require.Equal(t, string(auditlog.PortalDomainResourceType), logged.Targets[0].Type)
	require.Equal(t, res.Msg.GetDomainId(), logged.Targets[0].ID)
	require.Equal(t, f.portalID, logged.Targets[0].Meta["portalId"])
}

// An environment without a portal app or portal CNAME domain must refuse
// portal domains up front rather than accept rows it can never route.
func TestAddPortalDomainRequiresPortalConfig(t *testing.T) {
	cases := map[string]func(*Config){
		"no environment":  func(c *Config) { c.EnvironmentID = "" },
		"no cname domain": func(c *Config) { c.CnameDomain = "" },
	}
	for name, unset := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			f := newFixture(t)
			cfg := f.config(t, acceptingIngress(t))
			unset(&cfg)

			_, err := New(cfg).AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
			requireCode(t, connect.CodeFailedPrecondition, err)
			require.Equal(t, 0, f.countPortalDomains(t, f.workspaceID))
			f.requireNoAuditLogs(t, f.workspaceID)
		})
	}
}

func TestAddPortalDomainEnforcesCap(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	for range maxPortalDomainsPerWorkspace {
		f.insertPortalDomain(t, f.workspaceID, randomDomain(), db.PortalDomainsVerificationStatusPending)
	}

	_, err := svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	requireCode(t, connect.CodeResourceExhausted, err)
	require.Equal(t, maxPortalDomainsPerWorkspace, f.countPortalDomains(t, f.workspaceID))
}

func TestAddPortalDomainRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	_, err := svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	require.NoError(t, err)

	_, err = svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, strings.ToUpper(f.domain)))
	requireCode(t, connect.CodeAlreadyExists, err)
	require.Equal(t, 1, f.countPortalDomains(t, f.workspaceID))
}

// Covers AE3: one hostname routes one way, so a workspace holding it as a
// deploy domain cannot also attach it to its portal.
func TestAddPortalDomainRejectsHostnameHeldAsCustomDomain(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	require.NoError(t, f.database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
		ID:                    uid.New(uid.DomainPrefix),
		WorkspaceID:           f.workspaceID,
		ProjectID:             uid.New(uid.ProjectPrefix),
		AppID:                 uid.New(uid.AppPrefix),
		EnvironmentID:         uid.New(uid.EnvironmentPrefix),
		Domain:                f.domain,
		ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus:    db.CustomDomainsVerificationStatusPending,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             time.Now().UnixMilli(),
	}))

	_, err := svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	requireCode(t, connect.CodeAlreadyExists, err)
	require.Equal(t, 0, f.countPortalDomains(t, f.workspaceID))
	f.requireNoAuditLogs(t, f.workspaceID)
}

// The submit runs inside the insert transaction, so a rejected submit leaves
// nothing behind and retrying the create is the recovery.
func TestAddPortalDomainRollsBackWhenIngressRejects(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	var submits atomic.Int32
	_, err := New(f.config(t, rejectingIngress(t, &submits))).AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	requireCode(t, connect.CodeInternal, err)
	require.Equal(t, int32(1), submits.Load())
	require.Equal(t, 0, f.countPortalDomains(t, f.workspaceID))
	f.requireNoAuditLogs(t, f.workspaceID)

	_, err = f.newService(t).AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	require.NoError(t, err)
}

func TestDeleteVerifiedPortalDomain(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	cancels := &cancelRecorder{} //nolint:exhaustruct
	cfg := f.config(t, acceptingIngress(t))
	cfg.RestateAdmin = restateadmin.New(restateadmin.Config{BaseURL: cancels.serve(t), APIKey: ""})
	svc := New(cfg)

	created, err := svc.AddPortalDomain(ctx, f.addRequest(f.workspaceID, f.domain))
	require.NoError(t, err)
	domainID := created.Msg.GetDomainId()
	row, err := f.database.FindPortalDomainById(ctx, domainID)
	require.NoError(t, err)

	f.setStatus(t, domainID, db.PortalDomainsVerificationStatusVerified)
	f.insertRoute(t, f.domain)
	f.insertChallenge(t, domainID, f.workspaceID)
	f.insertCertificate(t, f.workspaceID, f.domain)

	_, err = svc.DeletePortalDomain(ctx, f.deleteRequest(f.workspaceID, domainID))
	require.NoError(t, err)

	require.Equal(t, []string{row.InvocationID.String}, cancels.ids())
	_, err = f.database.FindPortalDomainById(ctx, domainID)
	require.True(t, db.IsNotFound(err), "the row must be gone, got: %v", err)
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM frontline_routes WHERE fully_qualified_domain_name = ?", f.domain))
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM acme_challenges WHERE domain_id = ?", domainID))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM certificates WHERE hostname = ?", f.domain),
		"the certificate stays, as it does for deploy domains")

	logged := f.findAuditEvent(t, f.workspaceID, auditlog.PortalDomainDeleteEvent)
	require.Equal(t, domainID, logged.Targets[0].ID)
}

// Workspace B holds the verified row and its route. Workspace A's unverified
// claim on the same hostname shares the portal project, so deleting it must
// not touch the route.
func TestDeleteUnverifiedPortalDomainLeavesOtherTenantsRoute(t *testing.T) {
	for _, status := range []db.PortalDomainsVerificationStatus{
		db.PortalDomainsVerificationStatusPending,
		db.PortalDomainsVerificationStatusFailed,
	} {
		t.Run(string(status), func(t *testing.T) {
			ctx := context.Background()
			f := newFixture(t)
			svc := f.newService(t)

			other := f.seeder.CreateWorkspace(ctx).ID
			f.insertPortalDomain(t, other, f.domain, db.PortalDomainsVerificationStatusVerified)
			f.insertRoute(t, f.domain)

			mine := f.insertPortalDomain(t, f.workspaceID, f.domain, status)

			_, err := svc.DeletePortalDomain(ctx, f.deleteRequest(f.workspaceID, mine))
			require.NoError(t, err)

			require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM frontline_routes WHERE fully_qualified_domain_name = ?", f.domain))
			require.Equal(t, 1, f.countPortalDomains(t, other))
		})
	}
}

// Another workspace's id or another portal's id reads as NotFound, so ids
// cannot be probed, and nothing is deleted.
func TestDeletePortalDomainNotFound(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	domainID := f.insertPortalDomain(t, f.workspaceID, f.domain, db.PortalDomainsVerificationStatusPending)

	otherWorkspace := f.deleteRequest(uid.New(uid.WorkspacePrefix), domainID)
	_, err := svc.DeletePortalDomain(ctx, otherWorkspace)
	requireCode(t, connect.CodeNotFound, err)

	otherPortal := f.deleteRequest(f.workspaceID, domainID)
	otherPortal.Msg.PortalId = uid.New(uid.PortalPrefix)
	_, err = svc.DeletePortalDomain(ctx, otherPortal)
	requireCode(t, connect.CodeNotFound, err)

	require.Equal(t, 1, f.countPortalDomains(t, f.workspaceID))
}

func TestRetryPortalDomainVerification(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.newService(t)

	domainID := f.insertPortalDomain(t, f.workspaceID, f.domain, db.PortalDomainsVerificationStatusFailed)

	res, err := svc.RetryVerification(ctx, f.retryRequest(f.workspaceID, domainID))
	require.NoError(t, err)
	require.Equal(t, ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING, res.Msg.GetStatus())

	row, err := f.database.FindPortalDomainById(ctx, domainID)
	require.NoError(t, err)
	require.Equal(t, db.PortalDomainsVerificationStatusPending, row.VerificationStatus)
	require.True(t, row.InvocationID.Valid)
	f.findAuditEvent(t, f.workspaceID, auditlog.PortalDomainVerifyEvent)

	f.setStatus(t, domainID, db.PortalDomainsVerificationStatusVerified)
	_, err = svc.RetryVerification(ctx, f.retryRequest(f.workspaceID, domainID))
	requireCode(t, connect.CodeFailedPrecondition, err)
}

// fixture is a seeded tenant workspace plus the portal app's environment,
// which lives in a separate workspace.
type fixture struct {
	database        db.Database
	seeder          *seed.Seeder
	workspaceID     string
	portalID        string
	portalProjectID string
	portalAppID     string
	portalEnvID     string
	domain          string
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	ctx := context.Background()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	portalWorkspace := seeder.CreateWorkspace(ctx).ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: portalWorkspace,
		Name:        "Portal",
		Slug:        uid.DNS1035(16),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: portalWorkspace,
		ProjectID:   project.ID,
		Name:        "Portal",
		Slug:        uid.DNS1035(16),
	})
	env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: portalWorkspace,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Portal production",
	})

	return fixture{
		database:        database,
		seeder:          seeder,
		workspaceID:     seeder.Resources.UserWorkspace.ID,
		portalID:        uid.New(uid.PortalPrefix),
		portalProjectID: project.ID,
		portalAppID:     app.ID,
		portalEnvID:     env.ID,
		domain:          randomDomain(),
	}
}

func (f fixture) config(t *testing.T, ingressURL string) Config {
	t.Helper()

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: f.database})
	require.NoError(t, err)

	return Config{
		Database:      f.database,
		Restate:       restateingress.NewClient(ingressURL),
		RestateAdmin:  nil,
		Auditlogs:     auditlogSvc,
		CnameDomain:   testCnameDomain,
		EnvironmentID: f.portalEnvID,
		Bearer:        testBearer,
	}
}

func (f fixture) newService(t *testing.T) *Service {
	t.Helper()
	return New(f.config(t, acceptingIngress(t)))
}

func testActor() *ctrlv1.ActorInfo {
	return &ctrlv1.ActorInfo{
		Id:        "user_test",
		Name:      "Test User",
		Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
		RemoteIp:  "127.0.0.1",
		UserAgent: "test-agent",
		Meta:      map[string]string{},
	}
}

func authorized[T any](msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+testBearer)
	return req
}

func (f fixture) addRequest(workspaceID, domain string) *connect.Request[ctrlv1.AddPortalDomainRequest] {
	return authorized(&ctrlv1.AddPortalDomainRequest{
		WorkspaceId: workspaceID,
		PortalId:    f.portalID,
		Domain:      domain,
		Actor:       testActor(),
	})
}

func (f fixture) deleteRequest(workspaceID, domainID string) *connect.Request[ctrlv1.DeletePortalDomainRequest] {
	return authorized(&ctrlv1.DeletePortalDomainRequest{
		WorkspaceId: workspaceID,
		PortalId:    f.portalID,
		DomainId:    domainID,
		Actor:       testActor(),
	})
}

func (f fixture) retryRequest(workspaceID, domainID string) *connect.Request[ctrlv1.RetryPortalDomainVerificationRequest] {
	return authorized(&ctrlv1.RetryPortalDomainVerificationRequest{
		WorkspaceId: workspaceID,
		PortalId:    f.portalID,
		DomainId:    domainID,
		Actor:       testActor(),
	})
}

func (f fixture) insertPortalDomain(t *testing.T, workspaceID, domain string, status db.PortalDomainsVerificationStatus) string {
	t.Helper()

	id := uid.New(uid.PortalDomainPrefix)
	require.NoError(t, f.database.InsertPortalDomain(context.Background(), db.InsertPortalDomainParams{
		ID:                    id,
		WorkspaceID:           workspaceID,
		PortalID:              f.portalID,
		Domain:                domain,
		VerificationStatus:    status,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.DNS1035(16) + "." + testCnameDomain,
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             time.Now().UnixMilli(),
	}))
	return id
}

func (f fixture) setStatus(t *testing.T, domainID string, status db.PortalDomainsVerificationStatus) {
	t.Helper()
	require.NoError(t, f.database.UpdatePortalDomainVerificationStatus(context.Background(), db.UpdatePortalDomainVerificationStatusParams{
		ID:                 domainID,
		VerificationStatus: status,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))
}

// insertRoute routes domain to the portal app, as verification does.
func (f fixture) insertRoute(t *testing.T, domain string) {
	t.Helper()
	require.NoError(t, f.database.InsertFrontlineRoute(context.Background(), db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                f.portalProjectID,
		AppID:                    f.portalAppID,
		DeploymentID:             "",
		EnvironmentID:            f.portalEnvID,
		FullyQualifiedDomainName: domain,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                time.Now().UnixMilli(),
		UpdatedAt:                sql.NullInt64{},
	}))
}

func (f fixture) insertChallenge(t *testing.T, domainID, workspaceID string) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, f.database.InsertAcmeChallenge(context.Background(), db.InsertAcmeChallengeParams{
		DomainID:      domainID,
		WorkspaceID:   workspaceID,
		Token:         "",
		ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ExpiresAt:     now,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{},
	}))
}

func (f fixture) insertCertificate(t *testing.T, workspaceID, hostname string) {
	t.Helper()
	require.NoError(t, f.database.InsertCertificate(context.Background(), db.InsertCertificateParams{
		ID:                  uid.New(uid.TestPrefix),
		WorkspaceID:         workspaceID,
		Hostname:            hostname,
		Certificate:         "cert",
		EncryptedPrivateKey: "key",
		CreatedAt:           time.Now().UnixMilli(),
		UpdatedAt:           sql.NullInt64{},
	}))
}

func (f fixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, f.database.RW().QueryRowContext(context.Background(), query, args...).Scan(&n))
	return n
}

func (f fixture) countPortalDomains(t *testing.T, workspaceID string) int {
	t.Helper()
	return f.count(t, "SELECT COUNT(*) FROM portal_domains WHERE workspace_id = ?", workspaceID)
}

func (f fixture) requireNoAuditLogs(t *testing.T, workspaceID string) {
	t.Helper()
	rows, err := f.database.ListClickhouseOutboxByWorkspace(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

// findAuditEvent returns the single outbox entry carrying event.
func (f fixture) findAuditEvent(t *testing.T, workspaceID string, event auditlog.AuditLogEvent) auditlog.Event {
	t.Helper()

	rows, err := f.database.ListClickhouseOutboxByWorkspace(context.Background(), workspaceID)
	require.NoError(t, err)

	var matches []auditlog.Event
	for _, row := range rows {
		var logged auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &logged))
		if logged.Event == string(event) {
			matches = append(matches, logged)
		}
	}
	require.Len(t, matches, 1, "expected exactly one %s entry", event)
	return matches[0]
}

func requireCode(t *testing.T, want connect.Code, err error) {
	t.Helper()
	require.Error(t, err)
	connectErr, ok := errors.AsType[*connect.Error](err)
	require.True(t, ok, "expected a connect error, got %T: %v", err, err)
	require.Equal(t, want, connectErr.Code(), connectErr.Message())
}

// acceptingIngress answers every send with an accepted invocation.
func acceptingIngress(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/send") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{
			"invocationId": uid.New("inv"),
			"status":       "Accepted",
		}))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// rejectingIngress refuses every send with a 500, a permanent error to
// TxRetry, and counts the attempts.
func rejectingIngress(t *testing.T, submits *atomic.Int32) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/send") {
			submits.Add(1)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// cancelRecorder is a Restate admin API that accepts and records cancels.
type cancelRecorder struct {
	mu        sync.Mutex
	cancelled []string
}

func (c *cancelRecorder) serve(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := strings.CutPrefix(r.URL.Path, "/invocations/")
		id, isCancel := strings.CutSuffix(id, "/cancel")
		if !ok || !isCancel {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		c.mu.Lock()
		c.cancelled = append(c.cancelled, id)
		c.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func (c *cancelRecorder) ids() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.cancelled...)
}

// randomDomain keeps names unique across runs sharing one database.
func randomDomain() string {
	return uid.DNS1035(16) + ".example.com"
}
