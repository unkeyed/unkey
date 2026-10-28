package gdprerasure_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// seededWorkspace is a workspace with rows in most tables a workspace erasure
// covers, plus the ids its child tables are counted by.
type seededWorkspace struct {
	id               string
	orgID            string
	stripeCustomerID string
	projectID        string
	deploymentID     string
	connectionID     string
	legacyRootKeyID  string
}

// TestWorkspaceErasureRemovesEverythingAndSparesNeighbours guarantees that a
// GDPR workspace erasure leaves no row of the workspace in any MySQL or
// ClickHouse table, deletes its Stripe customer and WorkOS organization, leaves
// another workspace untouched, and converges: a second request deletes nothing.
func TestWorkspaceErasureRemovesEverythingAndSparesNeighbours(t *testing.T) {
	h := harness.New(t)
	internal := seedInternalWorkspace(t, h)
	subject := seedWorkspace(t, h, internal, false)
	neighbour := seedWorkspace(t, h, internal, false)

	subjectBefore := countWorkspaceRows(t, h, subject)
	require.Positive(t, subjectBefore.Workspaces)
	require.Positive(t, subjectBefore.KeyRows)
	require.Positive(t, subjectBefore.EncryptedKeys)
	require.Positive(t, subjectBefore.ConnectionAppTargets)
	require.Positive(t, subjectBefore.FrontlineRoutes)
	require.Positive(t, subjectBefore.LegacyRootKeys)
	require.Positive(t, subjectBefore.LegacyRootKeyPermissions)
	require.Positive(t, subjectBefore.CliSessions)
	require.NotEmpty(t, clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id))
	neighbourBefore := countWorkspaceRows(t, h, neighbour)
	neighbourClickHouseBefore := clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", neighbour.id)

	res, err := erase(t, h, workspaceRequest(subject.id))
	require.NoError(t, err)
	require.Equal(t, subject.stripeCustomerID, res.GetStripeCustomerId())

	require.Equal(t, db.CountWorkspaceRowsByTableRow{}, countWorkspaceRows(t, h, subject))
	require.Empty(t, clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id))
	require.Equal(t, []string{subject.stripeCustomerID}, h.Erased.StripeCustomerIDs())
	require.Equal(t, []string{subject.orgID}, h.Erased.WorkOSOrgIDs())

	require.Equal(t, neighbourBefore, countWorkspaceRows(t, h, neighbour))
	require.Equal(t, neighbourClickHouseBefore, clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", neighbour.id))

	again, err := erase(t, h, workspaceRequest(subject.id))
	require.NoError(t, err)
	requireNothingLeft(t, again)
	require.Empty(t, again.GetStripeCustomerId())
}

// TestWorkspaceErasureRefusesProtectedWorkspace guarantees that a workspace
// with delete protection is never erased: the request fails terminally before
// any step runs.
func TestWorkspaceErasureRefusesProtectedWorkspace(t *testing.T) {
	h := harness.New(t)
	internal := seedInternalWorkspace(t, h)
	subject := seedWorkspace(t, h, internal, true)
	before := countWorkspaceRows(t, h, subject)
	clickhouseBefore := clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id)

	_, err := erase(t, h, workspaceRequest(subject.id))
	require.Error(t, err)
	require.ErrorContains(t, err, "delete protection")

	require.Equal(t, before, countWorkspaceRows(t, h, subject))
	require.Equal(t, clickhouseBefore, clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id))
	require.Empty(t, h.Erased.StripeCustomerIDs())
	require.Empty(t, h.Erased.WorkOSOrgIDs())
}

// TestWorkspaceErasureFailsClosedWithoutStripeKey guarantees that a worker
// without a Stripe key refuses to erase a workspace that has a Stripe customer,
// instead of deleting its rows and stranding the customer.
func TestWorkspaceErasureFailsClosedWithoutStripeKey(t *testing.T) {
	h := harness.New(t, harness.WithoutStripe())
	internal := seedInternalWorkspace(t, h)
	subject := seedWorkspace(t, h, internal, false)
	before := countWorkspaceRows(t, h, subject)
	clickhouseBefore := clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id)

	_, err := erase(t, h, workspaceRequest(subject.id))
	require.Error(t, err)
	require.ErrorContains(t, err, "no stripe key")

	require.Equal(t, before, countWorkspaceRows(t, h, subject))
	require.Equal(t, clickhouseBefore, clickhouseCounts(t, h, "workspace_id", "workspace_id = ?", subject.id))
	require.Empty(t, h.Erased.WorkOSOrgIDs())
}

func workspaceRequest(workspaceID string) *hydrav1.GdprErasureRequest {
	return &hydrav1.GdprErasureRequest{Subject: &hydrav1.GdprErasureRequest_Workspace{
		Workspace: &hydrav1.GdprErasureWorkspace{WorkspaceId: workspaceID},
	}}
}

// internalWorkspace hosts legacy root keys, which live outside the workspace
// they grant access to.
type internalWorkspace struct {
	id         string
	keySpaceID string
}

func seedInternalWorkspace(t *testing.T, h *harness.Harness) internalWorkspace {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{WorkspaceID: ws.ID})
	return internalWorkspace{id: ws.ID, keySpaceID: api.KeyAuthID.String}
}

func seedWorkspace(t *testing.T, h *harness.Harness, internal internalWorkspace, deleteProtection bool) seededWorkspace {
	t.Helper()
	ctx := h.Ctx
	now := time.Now()

	ws := h.Seed.CreateWorkspace(ctx)
	require.NoError(t, h.DB.UpdateWorkspaceDeleteProtection(ctx, db.UpdateWorkspaceDeleteProtectionParams{
		DeleteProtection: sql.NullBool{Bool: deleteProtection, Valid: true},
		ID:               ws.ID,
	}))
	stripeCustomerID := uid.New(uid.TestPrefix)
	require.NoError(t, h.DB.SetWorkspaceStripeCustomerId(ctx, db.SetWorkspaceStripeCustomerIdParams{
		StripeCustomerID: sql.NullString{String: stripeCustomerID, Valid: true},
		WorkspaceID:      ws.ID,
	}))

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: ws.ID,
		Name:        "erasure",
		Slug:        uid.New(uid.TestPrefix),
	})
	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "api",
		Slug:        "api",
	})
	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        dbtype.EnvironmentKindProduction,
	})
	deployment := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        dbtype.DeploymentsStatusReady,
	})
	connectionID := h.Seed.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID:         ws.ID,
		ProjectID:           project.ID,
		CallerAppID:         app.ID,
		CallerEnvironmentID: env.ID,
		TargetAppID:         app.ID,
		Name:                "self",
	})
	require.NoError(t, h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             deployment.ID,
		EnvironmentID:            env.ID,
		FullyQualifiedDomainName: uid.DNS1035() + ".example.com",
		Sticky:                   db.FrontlineRoutesStickyNone,
		CreatedAt:                now.UnixMilli(),
	}))

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{WorkspaceID: ws.ID})
	identityID := h.Seed.CreateIdentity(ctx, seed.CreateIdentityRequest{
		WorkspaceID: ws.ID,
		ExternalID:  uid.New(uid.TestPrefix),
		Ratelimits:  []seed.CreateRatelimitRequest{{Name: "identity", WorkspaceID: ws.ID, Limit: 10, Duration: 1000}},
	})
	h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: ws.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  &identityID,
		Recoverable: true,
		Permissions: []seed.CreatePermissionRequest{{Name: "read", Slug: uid.New(uid.TestPrefix), WorkspaceID: ws.ID}},
		Roles:       []seed.CreateRoleRequest{{Name: uid.New(uid.TestPrefix), WorkspaceID: ws.ID}},
		Ratelimits:  []seed.CreateRatelimitRequest{{Name: "key", WorkspaceID: ws.ID, Limit: 10, Duration: 1000}},
	})
	legacyRootKey := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID:    internal.id,
		KeySpaceID:     internal.keySpaceID,
		ForWorkspaceID: &ws.ID,
		Permissions:    []seed.CreatePermissionRequest{{Name: "api.*.read_key", Slug: uid.New(uid.TestPrefix), WorkspaceID: internal.id}},
	})

	payload, err := json.Marshal(auditlog.Event{EventID: uid.New(uid.AuditLogPrefix), WorkspaceID: ws.ID, Event: "key.create"})
	require.NoError(t, err)
	require.NoError(t, h.DB.InsertClickhouseOutbox(ctx, db.InsertClickhouseOutboxParams{
		Version:     auditlog.OutboxVersionV1,
		WorkspaceID: ws.ID,
		EventID:     uid.New(uid.AuditLogPrefix),
		Payload:     payload,
		CreatedAt:   now.UnixMilli(),
	}))
	require.NoError(t, h.DB.InsertCliSession(ctx, db.InsertCliSessionParams{
		ID:               uid.New(uid.CLISessionPrefix),
		WorkspaceID:      ws.ID,
		UserID:           uid.New(uid.TestPrefix),
		UserName:         sql.NullString{String: "Ada Lovelace", Valid: true},
		Permissions:      json.RawMessage("[]"),
		RefreshTokenHash: uid.New(uid.TestPrefix),
		RefreshFamily:    uid.New(uid.TestPrefix),
		CreatedAt:        now.UnixMilli(),
		ExpiresAt:        now.Add(time.Hour).UnixMilli(),
	}))

	h.ClickHouseSeed.InsertVerifications(ctx, ws.ID, 3, now, "VALID")
	h.ClickHouseSeed.InsertRatelimits(ctx, ws.ID, 3, now, true)
	h.ClickHouseSeed.InsertFrontlineRequests(ctx, ws.ID, project.ID, app.ID, env.ID, deployment.ID, 3, now)
	require.NoError(t, h.ClickHouse.InsertAuditLogs(ctx, []schema.AuditLogV1{{
		EventID:     uid.New(uid.AuditLogPrefix),
		Time:        now.UnixMilli(),
		WorkspaceID: ws.ID,
		Bucket:      "unkey_mutations",
		Source:      auditlog.EventSourcePlatform,
		Event:       "key.create",
		ActorType:   "user",
		ActorID:     uid.New(uid.TestPrefix),
		ActorName:   "Ada Lovelace",
		ActorMeta:   json.RawMessage("{}"),
		Meta:        json.RawMessage("{}"),
	}}))

	return seededWorkspace{
		id:               ws.ID,
		orgID:            ws.OrgID,
		stripeCustomerID: stripeCustomerID,
		projectID:        project.ID,
		deploymentID:     deployment.ID,
		connectionID:     connectionID,
		legacyRootKeyID:  legacyRootKey.KeyID,
	}
}

func countWorkspaceRows(t *testing.T, h *harness.Harness, ws seededWorkspace) db.CountWorkspaceRowsByTableRow {
	t.Helper()
	counts, err := h.DB.CountWorkspaceRowsByTable(h.Ctx, db.CountWorkspaceRowsByTableParams{
		WorkspaceID:                    ws.id,
		DeviceAuthorizationWorkspaceID: sql.NullString{String: ws.id, Valid: true},
		ForWorkspaceID:                 sql.NullString{String: ws.id, Valid: true},
		LegacyRootKeyID:                ws.legacyRootKeyID,
		ConnectionID:                   ws.connectionID,
		DeploymentID:                   ws.deploymentID,
		ProjectID:                      ws.projectID,
	})
	require.NoError(t, err)
	return counts
}
