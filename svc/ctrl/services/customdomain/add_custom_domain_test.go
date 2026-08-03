package customdomain

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

var errInjectedAuditInsert = errors.New("injected audit insert failure")

// TestAddCustomDomainRollsBackWhenAuditInsertFails verifies the production
// guarantee that the custom domain row and its audit outbox row commit together.
// Without the shared transaction a failed audit insert would leave an orphaned
// domain that never appears in the audit trail.
func TestAddCustomDomainRollsBackWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	const bearer = "test-token"

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Atomic AddCustomDomain",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("project"), "_", "-")),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "Atomic App",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("app"), "_", "-")),
		DefaultBranch: "main",
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	domain := strings.ToLower(strings.ReplaceAll(uid.New("d"), "_", "")) + ".example.com"

	// No Restate client and no Domain Connect key: the failing audit insert aborts
	// the transaction before either is reached.
	svc := New(Config{
		Database:     database,
		Restate:      nil,
		RestateAdmin: nil,
		Auditlogs: failingAuditLogService{
			t:             t,
			workspaceID:   workspaceID,
			projectID:     project.ID,
			appID:         app.ID,
			environmentID: environment.ID,
			domain:        domain,
		},
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     bearer,
	})

	req := connect.NewRequest(&ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   workspaceID,
		ProjectId:     project.ID,
		AppId:         app.ID,
		EnvironmentId: environment.ID,
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
	req.Header().Set("Authorization", "Bearer "+bearer)

	_, err = svc.AddCustomDomain(ctx, req)
	require.Error(t, err)
	require.ErrorIs(t, err, errInjectedAuditInsert)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	require.Equal(t, 0, countRows(t, ctx, database.RW(), `
		SELECT COUNT(*)
		FROM custom_domains
		WHERE workspace_id = ? AND domain = ?
	`, workspaceID, domain))

	outboxRows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)
	require.Empty(t, outboxRows)
}

// TestAddCustomDomainRequiresActor pins the assertion that ctrl refuses to write
// an unattributed domain, so no caller can bypass the audit trail by omitting
// the actor.
func TestAddCustomDomainRequiresActor(t *testing.T) {
	ctx := context.Background()

	const bearer = "test-token"
	svc := New(Config{
		Database:                   nil,
		Restate:                    nil,
		RestateAdmin:               nil,
		Auditlogs:                  nil,
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     bearer,
	})

	req := connect.NewRequest(&ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   uid.New(uid.WorkspacePrefix),
		ProjectId:     uid.New(uid.ProjectPrefix),
		AppId:         uid.New(uid.AppPrefix),
		EnvironmentId: uid.New(uid.EnvironmentPrefix),
		Domain:        "api.example.com",
		Actor:         nil,
	})
	req.Header().Set("Authorization", "Bearer "+bearer)

	_, err := svc.AddCustomDomain(ctx, req)
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
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
