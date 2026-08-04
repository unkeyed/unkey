package customdomain

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const testBearer = "test-token"

var errInjectedAuditInsert = errors.New("injected audit insert failure")

// A domain in the database always has a matching outbox row, which is only observable
// once the insert commits.
func TestAddCustomDomainWritesAuditLog(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	svc := f.newService(t)

	res, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetDomainId())
	require.NotEmpty(t, res.Msg.GetVerificationToken())

	stored, err := f.database.FindCustomDomainById(ctx, res.Msg.GetDomainId())
	require.NoError(t, err)
	require.Equal(t, f.domain, stored.Domain)
	require.Equal(t, db.CustomDomainsVerificationStatusPending, stored.VerificationStatus)
	require.True(t, stored.InvocationID.Valid,
		"a started workflow must have its invocation id persisted so it can be cancelled")

	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)
	require.Len(t, outboxRows, 1)

	var logged auditlog.Event
	require.NoError(t, json.Unmarshal(outboxRows[0].Payload, &logged))
	require.Equal(t, f.workspaceID, logged.WorkspaceID)
	require.Equal(t, string(auditlog.DomainCreateEvent), logged.Event)
	require.Equal(t, string(auditlog.UserActor), logged.Actor.Type)
	require.Equal(t, "user_test", logged.Actor.ID)
	require.Len(t, logged.Targets, 1)
	require.Equal(t, string(auditlog.DomainResourceType), logged.Targets[0].Type)
	require.Equal(t, res.Msg.GetDomainId(), logged.Targets[0].ID)
	require.Equal(t, f.domain, logged.Targets[0].Meta["domain"])
	require.Equal(t, f.environmentID, logged.Targets[0].Meta["environmentId"])
}

// Restate retries only an invocation it has accepted, so nothing is running here and
// nothing will start on its own. Answering OK would send the caller off to poll a
// verification that cannot progress. The row stays, marked `failed`, which is what
// RetryVerification resets from.
func TestAddCustomDomainFailsWhenVerificationCannotStart(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	svc := f.newServiceUnreachableRestate(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.Error(t, err, "a domain whose workflow never started must not report success")

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	stored, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err, "the domain row must survive so the caller can retry verification")
	require.Equal(t, db.CustomDomainsVerificationStatusFailed, stored.VerificationStatus,
		"a domain whose verification workflow never started must not stay pending")
	require.True(t, stored.VerificationError.Valid)
	require.NotEmpty(t, stored.VerificationError.String)
	require.False(t, stored.InvocationID.Valid, "no invocation was accepted, so none may be recorded")
}

// TestAddCustomDomainRollsBackWhenAuditInsertFails verifies the production
// guarantee that the custom domain row and its audit outbox row commit together.
// Without the shared transaction a failed audit insert would leave an orphaned
// domain that never appears in the audit trail.
func TestAddCustomDomainRollsBackWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	// No Restate client and no Domain Connect key: the failing audit insert aborts
	// the transaction before either is reached.
	svc := New(Config{
		Database:     f.database,
		Restate:      nil,
		RestateAdmin: nil,
		Auditlogs: failingAuditLogService{
			t:             t,
			workspaceID:   f.workspaceID,
			projectID:     f.projectID,
			appID:         f.appID,
			environmentID: f.environmentID,
			domain:        f.domain,
		},
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     testBearer,
	})

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.Error(t, err)
	require.ErrorIs(t, err, errInjectedAuditInsert)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	require.Equal(t, 0, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, f.workspaceID, f.domain))

	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)
	require.Empty(t, outboxRows)
}

// TestAddCustomDomainEnforcesPlanAllowance pins that custom_domains_max is
// actually read. Without this the column is decorative and any workspace can
// attach unlimited domains, free tier included.
func TestAddCustomDomainEnforcesPlanAllowance(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	svc := f.newService(t)

	// The allowance is one, so the first domain is accepted.
	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	// The second is refused even though its name is free, because the allowance
	// counts domains rather than names.
	second := randomDomain()
	_, err = svc.AddCustomDomain(ctx, f.request(second))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeResourceExhausted, connectErr.Code())

	// Nothing was written for the refused request.
	require.Equal(t, 0, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, f.workspaceID, second))
}

// TestAddCustomDomainRejectsDuplicate pins the workspace-uniqueness pre-check. It
// runs before the insert, so the caller gets AlreadyExists rather than the
// duplicate-key internal error the unique index would raise. The repeat is also
// sent uppercase: the stored name is canonical, so case must not open a second slot.
func TestAddCustomDomainRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 5)

	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	for _, repeat := range []string{f.domain, strings.ToUpper(f.domain)} {
		_, err = svc.AddCustomDomain(ctx, f.request(repeat))
		require.Error(t, err, "re-adding %q must be refused", repeat)

		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		require.Equal(t, connect.CodeAlreadyExists, connectErr.Code())
		require.Contains(t, connectErr.Message(), f.domain,
			"the message must name the colliding domain")
	}

	require.Equal(t, 1, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, f.workspaceID, f.domain))
}

// TestAddCustomDomainConcurrentDuplicateReadsTheSame pins that a name lost to a
// concurrent create reads exactly like one rejected before the insert. Two paths
// reject a duplicate, the pre-check and the unique index, and the API reflects
// whichever message ctrl sends. Asserted without caring which path fired, so it
// holds either way the race lands.
func TestAddCustomDomainConcurrentDuplicateReadsTheSame(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 5)
	svc := f.newService(t)

	want := fault.UserFacingMessage(domaingate.AlreadyAttached(f.domain))

	type result struct {
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
			results <- result{err: err}
		}()
	}
	close(start)

	var succeeded, rejected int
	for range 2 {
		r := <-results
		if r.err == nil {
			succeeded++
			continue
		}
		rejected++

		var connectErr *connect.Error
		require.ErrorAs(t, r.err, &connectErr)
		require.Equal(t, connect.CodeAlreadyExists, connectErr.Code(),
			"a duplicate must be AlreadyExists whichever path caught it")
		require.Equal(t, want, connectErr.Message(),
			"both duplicate paths must carry the gate's message, not ctrl's internal wording")
	}

	require.Equal(t, 1, succeeded, "exactly one create may win")
	require.Equal(t, 1, rejected)

	require.Equal(t, 1, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, f.workspaceID, f.domain))
}

// TestAddCustomDomainAllowanceIsWorkspaceWide pins that the allowance counts every
// domain in the workspace, not per environment. A per-environment reading would let
// one workspace multiply its allowance by adding environments.
func TestAddCustomDomainAllowanceIsWorkspaceWide(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	// A different name, a different environment, the same workspace.
	other := f.newEnvironment(t)
	_, err = svc.AddCustomDomain(ctx, f.requestIn(other, randomDomain()))
	require.Error(t, err)

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeResourceExhausted, connectErr.Code())
}

// The index is (workspace_id, domain), so a name one workspace holds must not block
// another; contention is settled later by the worker's TXT check. Also proves the
// allowance count is workspace-scoped, since the second workspace is granted one and
// the first already took one.
func TestAddCustomDomainAllowsSameDomainInAnotherWorkspace(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	first, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	other := f.newWorkspace(t, 1)
	second, err := svc.AddCustomDomain(ctx, f.requestIn(other, f.domain))
	require.NoError(t, err, "another workspace must be able to claim the same name")
	require.NotEqual(t, first.Msg.GetDomainId(), second.Msg.GetDomainId())

	// Each workspace holds its own row for the name.
	require.Equal(t, 1, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, f.workspaceID, f.domain))
	require.Equal(t, 1, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, other.workspaceID, f.domain))

	// Distinct CNAME targets: target_cname is globally unique, so a shared name must
	// not produce a shared target.
	require.Equal(t, 2, countRows(t, ctx, f.database.RW(), `
		SELECT COUNT(DISTINCT target_cname)
		FROM custom_domains
		WHERE domain = ?
	`, f.domain))
}

// TestAddCustomDomainRefusesWorkspaceWithoutLimits pins the fail-closed choice.
// Every allowance is written by billing, so a missing row means billing state is
// unknown. Defaulting it would hand out paid capacity to whichever workspace
// billing has not written yet.
func TestAddCustomDomainRefusesWorkspaceWithoutLimits(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	_, err := f.database.RW().ExecContext(ctx, "DELETE FROM `limits` WHERE workspace_id = ?", f.workspaceID)
	require.NoError(t, err)

	svc := f.newService(t)

	_, err = svc.AddCustomDomain(ctx, f.request(f.domain))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
}

// TestAddCustomDomainMissingRequiredFieldIsInternal keeps ctrl's own asserts off the
// codes the API reflects to callers. On InvalidArgument, "workspace_id is required"
// would be published as a 400.
func TestAddCustomDomainMissingRequiredFieldIsInternal(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	req := connect.NewRequest(&ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   "",
		ProjectId:     f.projectID,
		AppId:         f.appID,
		EnvironmentId: f.environmentID,
		Domain:        f.domain,
		Actor:         nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err := svc.AddCustomDomain(ctx, req)
	require.Error(t, err)

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
	require.Contains(t, connectErr.Message(), "workspace_id",
		"the detail is still useful internally, it just must not be a reflected code")
}

// TestAddCustomDomainAttributesMissingActorToSystem pins that an absent actor is
// degraded rather than rejected. Requiring one would break any caller not yet
// redeployed, and the audit entry is worth more than its attribution: the domain
// and the entry commit together, so refusing the actor would drop both.
func TestAddCustomDomainAttributesMissingActorToSystem(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	svc := f.newService(t)

	req := connect.NewRequest(&ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   f.workspaceID,
		ProjectId:     f.projectID,
		AppId:         f.appID,
		EnvironmentId: f.environmentID,
		Domain:        f.domain,
		Actor:         nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	res, err := svc.AddCustomDomain(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetDomainId())

	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)
	require.Len(t, outboxRows, 1, "the entry is still written, just unattributed")

	var logged auditlog.Event
	require.NoError(t, json.Unmarshal(outboxRows[0].Payload, &logged))
	require.Equal(t, string(auditlog.SystemActor), logged.Actor.Type)
	require.Empty(t, logged.Actor.ID)
	require.Equal(t, string(auditlog.DomainCreateEvent), logged.Event)
}

// chain is one workspace/project/app/environment path a domain can be attached to.
type chain struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

// fixture is a seeded chain plus a limits row granting customDomainsMax domains.
type fixture struct {
	database db.Database
	seeder   *seed.Seeder
	chain
	domain string
}

func newFixture(t *testing.T, customDomainsMax uint32) fixture {
	t.Helper()

	ctx := context.Background()
	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	f := fixture{
		database: database,
		seeder:   seeder,
		chain:    chain{}, //nolint:exhaustruct
		domain:   randomDomain(),
	}
	f.chain = f.seedChain(t, seeder.Resources.UserWorkspace.ID, customDomainsMax)

	return f
}

// seedChain builds a project, app, and environment under workspaceID and grants it
// an allowance.
func (f fixture) seedChain(t *testing.T, workspaceID string, customDomainsMax uint32) chain {
	t.Helper()

	ctx := context.Background()
	project := f.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Atomic AddCustomDomain",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("project"), "_", "-")),
	})
	app := f.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "Atomic App",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("app"), "_", "-")),
		DefaultBranch: "main",
	})
	environment := f.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	_, err := f.database.RW().ExecContext(ctx, `
		INSERT INTO `+"`limits`"+`
			(workspace_id, api_billable_operations_count_max_per_month, logs_retention_days_max,
			 logs_audit_retention_days_max, team_enabled, cpu_cores_max, cpu_cores_max_per_instance,
			 memory_mib_max, memory_mib_max_per_instance, storage_mib_max, storage_mib_max_per_instance,
			 builds_concurrent_max, custom_domains_max, autoscaling_replicas_max)
		VALUES (?, 150000, 7, 30, false, 10, 2, 20480, 4096, 51200, 10240, 1, ?, 0)
		ON DUPLICATE KEY UPDATE custom_domains_max = VALUES(custom_domains_max)
	`, workspaceID, customDomainsMax)
	require.NoError(t, err)

	return chain{
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

// newWorkspace seeds an unrelated workspace with its own allowance.
func (f fixture) newWorkspace(t *testing.T, customDomainsMax uint32) chain {
	t.Helper()
	return f.seedChain(t, f.seeder.CreateWorkspace(context.Background()).ID, customDomainsMax)
}

// newEnvironment adds a second environment beside f's, under the same app.
func (f fixture) newEnvironment(t *testing.T) chain {
	t.Helper()

	environment := f.seeder.CreateEnvironment(context.Background(), seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: f.workspaceID,
		ProjectID:   f.projectID,
		AppID:       f.appID,
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("env"), "_", "-")),
		Description: "Second environment",
	})

	next := f.chain
	next.environmentID = environment.ID
	return next
}

func (f fixture) request(domain string) *connect.Request[ctrlv1.AddCustomDomainRequest] {
	return f.requestIn(f.chain, domain)
}

func (f fixture) requestIn(c chain, domain string) *connect.Request[ctrlv1.AddCustomDomainRequest] {
	req := connect.NewRequest(&ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   c.workspaceID,
		ProjectId:     c.projectID,
		AppId:         c.appID,
		EnvironmentId: c.environmentID,
		Domain:        domain,
		Actor: &ctrlv1.ActorInfo{
			Id:        "user_test",
			Name:      "Test User",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
			RemoteIp:  "127.0.0.1",
			UserAgent: "test-agent",
			Meta:      map[string]string{},
		},
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)
	return req
}

func (f fixture) newService(t *testing.T) *Service {
	t.Helper()
	return f.newServiceWithRestate(t, acceptingIngress(t))
}

func (f fixture) newServiceUnreachableRestate(t *testing.T) *Service {
	t.Helper()
	return f.newServiceWithRestate(t, "http://127.0.0.1:1")
}

func (f fixture) newServiceWithRestate(t *testing.T, ingressURL string) *Service {
	t.Helper()

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: f.database})
	require.NoError(t, err)

	return New(Config{
		Database:                   f.database,
		Restate:                    restateingress.NewClient(ingressURL),
		RestateAdmin:               nil,
		Auditlogs:                  auditlogSvc,
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     testBearer,
	})
}

// acceptingIngress answers every send with an accepted invocation. A create only reports
// success once Restate has taken the invocation, so without a reachable ingress every
// test here would exercise the failure path.
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

// randomDomain keeps names unique across runs sharing one database.
func randomDomain() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("d"), "_", "")) + ".example.com"
}

type failingAuditLogService struct {
	t             *testing.T
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	domain        string
}

func (s failingAuditLogService) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	s.t.Helper()

	require.NotNil(s.t, tx)
	require.Len(s.t, logs, 1)
	require.Equal(s.t, s.workspaceID, logs[0].WorkspaceID)
	require.Equal(s.t, auditlog.DomainCreateEvent, logs[0].Event)
	require.Equal(s.t, auditlog.UserActor, logs[0].ActorType)
	require.Len(s.t, logs[0].Resources, 1)
	require.Equal(s.t, auditlog.DomainResourceType, logs[0].Resources[0].Type)
	require.Equal(s.t, s.domain, logs[0].Resources[0].Meta["domain"])
	require.Equal(s.t, s.projectID, logs[0].Resources[0].Meta["projectId"])
	require.Equal(s.t, s.appID, logs[0].Resources[0].Meta["appId"])
	require.Equal(s.t, s.environmentID, logs[0].Resources[0].Meta["environmentId"])

	// The domain row is visible inside the transaction, proving both writes share it.
	require.Equal(s.t, 1, countRows(s.t, ctx, tx, `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE id = ? AND workspace_id = ? AND domain = ?
	`, logs[0].Resources[0].ID, s.workspaceID, s.domain))

	return errInjectedAuditInsert
}

func countRows(t *testing.T, ctx context.Context, tx db.DBTX, query string, args ...any) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err)
	return count
}
