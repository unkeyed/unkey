package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/notificationrouting"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
	"google.golang.org/protobuf/proto"
)

const testBearer = "notification-test-bearer"

type directory struct {
	workos.Directory
	byOrg map[string][]workos.Member
}

func (d *directory) Members(_ context.Context, orgID string) ([]workos.Member, error) {
	members := append([]workos.Member{}, d.byOrg[orgID]...)
	sort.Slice(members, func(i, j int) bool { return members[i].UserID < members[j].UserID })
	return members, nil
}

func (d *directory) LookupMembers(_ context.Context, orgID string, ids []string) ([]workos.Member, error) {
	var members []workos.Member
	for _, member := range d.byOrg[orgID] {
		for _, id := range ids {
			if id == member.UserID {
				members = append(members, member)
				break
			}
		}
	}
	return members, nil
}

type failingAudits struct {
	auditlogs.AuditLogService
	fail atomic.Bool
}

func (a *failingAudits) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	if err := a.AuditLogService.Insert(ctx, tx, logs); err != nil {
		return err
	}
	if a.fail.Load() {
		return errors.New("audit failure after insertion")
	}
	return nil
}

type fixture struct {
	database    db.Database
	crypto      vault.VaultServiceClient
	directory   *directory
	audits      *failingAudits
	client      ctrlv1connect.NotificationServiceClient
	workspaceID string
	orgID       string
	userID      string
	userOtherID string
	actor       *ctrlv1.ActorInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	crypto := vault.NewConnectVaultServiceClient(vaulttestutil.StartTestVaultWithMemory(t).Client)
	writer, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	f := &fixture{
		database:    database,
		crypto:      crypto,
		audits:      &failingAudits{AuditLogService: writer},
		directory:   &directory{byOrg: make(map[string][]workos.Member)},
		workspaceID: uid.New(uid.WorkspacePrefix),
		orgID:       uid.New(uid.OrgPrefix),
		userID:      uid.New(uid.TestPrefix),
		userOtherID: uid.New(uid.TestPrefix),
	}
	require.NoError(t, database.InsertWorkspace(t.Context(), db.InsertWorkspaceParams{ID: f.workspaceID, OrgID: f.orgID, Name: "Notifications", Slug: uid.New(uid.TestPrefix), K8sNamespace: uid.DNS1035()}))
	f.actor = &ctrlv1.ActorInfo{Id: f.userID, Name: "Admin", Type: ctrlv1.ActorType_ACTOR_TYPE_USER}
	f.directory.byOrg[f.orgID] = []workos.Member{{UserID: f.userID, Email: "admin@example.com", Admin: true}, {UserID: f.userOtherID, Email: "member@example.com"}}
	service := New(Config{Database: database, Vault: crypto, Directory: f.directory, Auditlogs: f.audits, Bearer: testBearer})
	_, handler := ctrlv1connect.NewNotificationServiceHandler(service)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	f.client = ctrlv1connect.NewNotificationServiceClient(server.Client(), server.URL)
	return f
}

func authorized[T any](t *testing.T, message *T) *connect.Request[T] {
	t.Helper()
	request := connect.NewRequest(message)
	request.Header().Set("Authorization", "Bearer "+testBearer)
	return request
}

func (f *fixture) destinationRequest(t *testing.T) *ctrlv1.NotificationCreateDestinationRequest {
	t.Helper()
	return &ctrlv1.NotificationCreateDestinationRequest{
		WorkspaceId:    f.workspaceID,
		Name:           "Webhook",
		Channel:        "webhook",
		Url:            "https://example.com/private-token?credential=hidden",
		Headers:        []*ctrlv1.NotificationHeader{{Name: "Authorization", Value: "Bearer hidden-header"}},
		SigningSecret:  strings.Repeat("secret", 8),
		Enabled:        true,
		IdempotencyKey: uid.New(uid.TestPrefix),
		Actor:          f.actor,
	}
}

func (f *fixture) resource(t *testing.T) notification.Resource {
	t.Helper()
	resource := notification.Resource{ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix)}
	require.NoError(t, f.database.InsertProject(t.Context(), db.InsertProjectParams{ID: resource.ProjectID, WorkspaceID: f.workspaceID, Name: "Project", Slug: uid.New(uid.TestPrefix)}))
	require.NoError(t, f.database.InsertApp(t.Context(), db.InsertAppParams{ID: resource.AppID, WorkspaceID: f.workspaceID, ProjectID: resource.ProjectID, Name: "App", Slug: uid.New(uid.TestPrefix), SourceType: db.AppsSourceTypeUnknown}))
	require.NoError(t, f.database.InsertEnvironment(t.Context(), db.InsertEnvironmentParams{ID: resource.EnvironmentID, WorkspaceID: f.workspaceID, ProjectID: resource.ProjectID, AppID: resource.AppID, Slug: uid.New(uid.TestPrefix), Kind: mysqltype.EnvironmentKindProduction}))
	return resource
}

func (f *fixture) ruleRequest(t *testing.T, targets ...*ctrlv1.NotificationTarget) *ctrlv1.NotificationCreateRuleRequest {
	t.Helper()
	return &ctrlv1.NotificationCreateRuleRequest{WorkspaceId: f.workspaceID, Name: "Deployments", NotificationType: deployment.Live.Type(), Enabled: true, Targets: targets, IdempotencyKey: uid.New(uid.TestPrefix), Actor: f.actor}
}

func TestDestinationsRequireConfiguredVault(t *testing.T) {
	f := newFixture(t)
	created, err := f.client.CreateDestination(t.Context(), authorized(t, f.destinationRequest(t)))
	require.NoError(t, err)
	service := New(Config{Database: f.database, Directory: f.directory, Auditlogs: f.audits, Bearer: testBearer})
	_, err = service.GetCatalog(t.Context(), authorized(t, &ctrlv1.NotificationGetCatalogRequest{WorkspaceId: f.workspaceID}))
	require.NoError(t, err)
	_, err = service.CreateDestination(t.Context(), authorized(t, f.destinationRequest(t)))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	_, err = service.GetDestination(t.Context(), authorized(t, &ctrlv1.NotificationGetDestinationRequest{
		WorkspaceId: f.workspaceID, DestinationId: created.Msg.GetDestination().GetId(),
	}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestDestinationCRUDRedactsSecretsAndPreservesFirstWrite(t *testing.T) {
	f := newFixture(t)
	request := f.destinationRequest(t)
	created, err := f.client.CreateDestination(t.Context(), authorized(t, request))
	require.NoError(t, err)
	id := created.Msg.GetDestination().GetId()
	require.Equal(t, "https://example.com", created.Msg.GetDestination().GetUrlHint())
	require.Equal(t, []string{"Authorization"}, created.Msg.GetDestination().GetHeaderNames())
	require.True(t, created.Msg.GetDestination().GetEnabled())
	row, err := f.database.FindNotificationDestination(t.Context(), db.FindNotificationDestinationParams{WorkspaceID: f.workspaceID, ID: id})
	require.NoError(t, err)
	for _, secret := range []string{request.GetUrl(), request.GetSigningSecret(), "hidden-header", "private-token"} {
		require.NotContains(t, string(row.Config), secret)
		require.NotContains(t, created.Msg.String(), secret)
	}
	config := &notificationv1.DestinationConfig{}
	require.NoError(t, proto.Unmarshal(row.Config, config))
	plaintext, err := f.crypto.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: f.workspaceID, Encrypted: config.GetWebhook().GetEncryptedUrl()})
	require.NoError(t, err)
	require.Equal(t, request.GetUrl(), plaintext.GetPlaintext())
	_, err = f.crypto.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: uid.New(uid.WorkspacePrefix), Encrypted: config.GetWebhook().GetEncryptedUrl()})
	require.Error(t, err)

	request.Name = "Must not replace"
	request.Url = "https://different.example.com/secret"
	request.SigningSecret = "invalid on purpose"
	retried, err := f.client.CreateDestination(t.Context(), authorized(t, request))
	require.NoError(t, err)
	require.True(t, proto.Equal(created.Msg.GetDestination(), retried.Msg.GetDestination()))
	updated, err := f.client.UpdateDestination(t.Context(), authorized(t, &ctrlv1.NotificationUpdateDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: id, Name: proto.String("Paused"), Enabled: proto.Bool(false), Headers: &ctrlv1.NotificationHeaders{}, Actor: f.actor}))
	require.NoError(t, err)
	require.False(t, updated.Msg.GetDestination().GetEnabled())
	require.Empty(t, updated.Msg.GetDestination().GetHeaderNames())
	rotation := &ctrlv1.NotificationRotateSigningSecretRequest{WorkspaceId: f.workspaceID, DestinationId: id, SigningSecret: strings.Repeat("rotated", 8), Actor: f.actor}
	for range 2 {
		rotated, err := f.client.RotateSigningSecret(t.Context(), authorized(t, rotation))
		require.NoError(t, err)
		require.NotContains(t, rotated.Msg.String(), rotation.GetSigningSecret())
	}
	row, err = f.database.FindNotificationDestination(t.Context(), db.FindNotificationDestinationParams{WorkspaceID: f.workspaceID, ID: id})
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(row.Config, config))
	plaintext, err = f.crypto.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: f.workspaceID, Encrypted: config.GetWebhook().GetEncryptedSigningSecret()})
	require.NoError(t, err)
	require.Equal(t, rotation.GetSigningSecret(), plaintext.GetPlaintext())

	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	require.NoError(t, f.database.InsertWorkspace(t.Context(), db.InsertWorkspaceParams{ID: otherWorkspaceID, OrgID: uid.New(uid.OrgPrefix), Name: "Other", Slug: uid.New(uid.TestPrefix), K8sNamespace: uid.DNS1035()}))
	_, err = f.client.GetDestination(t.Context(), authorized(t, &ctrlv1.NotificationGetDestinationRequest{WorkspaceId: otherWorkspaceID, DestinationId: id}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = f.client.UpdateDestination(t.Context(), authorized(t, &ctrlv1.NotificationUpdateDestinationRequest{WorkspaceId: otherWorkspaceID, DestinationId: id, Enabled: proto.Bool(true), Actor: f.actor}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = f.client.DeleteDestination(t.Context(), authorized(t, &ctrlv1.NotificationDeleteDestinationRequest{WorkspaceId: otherWorkspaceID, DestinationId: id, Actor: f.actor}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	audits, err := f.database.ListClickhouseOutboxByWorkspace(t.Context(), f.workspaceID)
	require.NoError(t, err)
	require.Len(t, audits, 4)
	for _, row := range audits {
		var event auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &event))
		require.Equal(t, f.actor.GetId(), event.Actor.ID)
		require.Equal(t, id, event.Targets[0].ID)
		for _, secret := range []string{"private-token", "hidden-header", rotation.GetSigningSecret()} {
			require.NotContains(t, string(row.Payload), secret)
		}
	}
	_, err = f.client.DeleteDestination(t.Context(), authorized(t, &ctrlv1.NotificationDeleteDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: id, Actor: f.actor}))
	require.NoError(t, err)
	_, err = f.client.GetDestination(t.Context(), authorized(t, &ctrlv1.NotificationGetDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: id}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDestinationValidationAndPagination(t *testing.T) {
	f := newFixture(t)
	for _, key := range []string{" padded ", "key\x00suffix", strings.Repeat("é", 65)} {
		request := f.destinationRequest(t)
		request.IdempotencyKey = key
		_, err := f.client.CreateDestination(t.Context(), authorized(t, request))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		_, err = f.client.TestDestination(t.Context(), authorized(t, &ctrlv1.NotificationTestDestinationRequest{
			WorkspaceId: f.workspaceID, DestinationId: uid.New(uid.TestPrefix), IdempotencyKey: key, Actor: f.actor,
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	for _, endpoint := range []string{"http://example.com/token", "https://user:secret@example.com/token", "https://127.0.0.1/token", "https://10.0.0.1/token", "https://[::1]/token", "https://[::ffff:127.0.0.1]/token", "https://example.com/#secret", "https://example.com:99999/token"} {
		t.Run(endpoint, func(t *testing.T) {
			request := f.destinationRequest(t)
			request.Url = endpoint
			_, err := f.client.CreateDestination(t.Context(), authorized(t, request))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.NotContains(t, err.Error(), endpoint)
		})
	}
	for _, channel := range []string{"slack", "discord", "webhook"} {
		request := f.destinationRequest(t)
		request.Channel = channel
		if channel == "slack" {
			request.Url = "https://hooks.slack.com/services/TEAM/CHANNEL/secret"
			request.Headers, request.SigningSecret = nil, ""
		}
		if channel == "discord" {
			request.Url = "https://discord.com/api/webhooks/123456/secret"
			request.Headers, request.SigningSecret = nil, ""
		}
		_, err := f.client.CreateDestination(t.Context(), authorized(t, request))
		require.NoError(t, err)
	}
	first, err := f.client.ListDestinations(t.Context(), authorized(t, &ctrlv1.NotificationListDestinationsRequest{WorkspaceId: f.workspaceID, Limit: 2}))
	require.NoError(t, err)
	require.Len(t, first.Msg.GetDestinations(), 2)
	require.True(t, first.Msg.GetHasMore())
	last, err := f.client.ListDestinations(t.Context(), authorized(t, &ctrlv1.NotificationListDestinationsRequest{WorkspaceId: f.workspaceID, Limit: 2, Cursor: first.Msg.GetNextCursor()}))
	require.NoError(t, err)
	require.Len(t, last.Msg.GetDestinations(), 1)
	require.False(t, last.Msg.GetHasMore())
	require.Empty(t, last.Msg.GetNextCursor())
	_, err = f.client.ListDestinations(t.Context(), authorized(t, &ctrlv1.NotificationListDestinationsRequest{WorkspaceId: f.workspaceID, Limit: 101}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = f.client.ListDestinations(t.Context(), connect.NewRequest(&ctrlv1.NotificationListDestinationsRequest{WorkspaceId: f.workspaceID}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	request := f.destinationRequest(t)
	request.Headers = []*ctrlv1.NotificationHeader{{Name: "unkey-signature", Value: "credential"}}
	_, err = f.client.CreateDestination(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	request.Headers = nil
	request.SigningSecret = "short"
	_, err = f.client.CreateDestination(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRuleUpdatesAndAuditWritesAreAtomic(t *testing.T) {
	f := newFixture(t)
	destination, err := f.client.CreateDestination(t.Context(), authorized(t, f.destinationRequest(t)))
	require.NoError(t, err)
	target := &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: destination.Msg.GetDestination().GetId()}}
	created, err := f.client.CreateRule(t.Context(), authorized(t, f.ruleRequest(t, target)))
	require.NoError(t, err)
	ruleID := created.Msg.GetRule().GetId()
	_, err = f.client.DeleteDestination(t.Context(), authorized(t, &ctrlv1.NotificationDeleteDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: target.GetDestinationId(), Actor: f.actor}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	update := &ctrlv1.NotificationUpdateRuleRequest{WorkspaceId: f.workspaceID, RuleId: ruleID, Name: "Replacement", NotificationType: deployment.Live.Type(), Enabled: false, Targets: []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userID}}}, Actor: f.actor}
	f.audits.fail.Store(true)
	_, err = f.client.UpdateRule(t.Context(), authorized(t, update))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	got, err := f.client.GetRule(t.Context(), authorized(t, &ctrlv1.NotificationGetRuleRequest{WorkspaceId: f.workspaceID, RuleId: ruleID}))
	require.NoError(t, err)
	require.True(t, proto.Equal(created.Msg.GetRule(), got.Msg.GetRule()))
	audits, err := f.database.ListClickhouseOutboxByWorkspace(t.Context(), f.workspaceID)
	require.NoError(t, err)
	require.Len(t, audits, 2)
	createFailed := f.destinationRequest(t)
	_, err = f.client.CreateDestination(t.Context(), authorized(t, createFailed))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	_, err = f.database.FindNotificationDestination(t.Context(), db.FindNotificationDestinationParams{WorkspaceID: f.workspaceID, ID: stableID(f.workspaceID, "destination", createFailed.GetIdempotencyKey())})
	require.True(t, db.IsNotFound(err))
	f.audits.fail.Store(false)
	updated, err := f.client.UpdateRule(t.Context(), authorized(t, update))
	require.NoError(t, err)
	require.Equal(t, "Replacement", updated.Msg.GetRule().GetName())
	require.False(t, updated.Msg.GetRule().GetEnabled())
	require.Len(t, updated.Msg.GetRule().GetTargets(), 1)
	require.Equal(t, f.userID, updated.Msg.GetRule().GetTargets()[0].GetUserId())
	_, err = f.client.DeleteDestination(t.Context(), authorized(t, &ctrlv1.NotificationDeleteDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: target.GetDestinationId(), Actor: f.actor}))
	require.NoError(t, err)
	_, err = f.client.DeleteRule(t.Context(), authorized(t, &ctrlv1.NotificationDeleteRuleRequest{WorkspaceId: f.workspaceID, RuleId: ruleID, Actor: f.actor}))
	require.NoError(t, err)
	targets, err := f.database.ListNotificationRuleTargets(t.Context(), db.ListNotificationRuleTargetsParams{WorkspaceID: f.workspaceID, SubscriptionID: ruleID})
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestPreviewUsesWorkerRoutingAndPreferences(t *testing.T) {
	f := newFixture(t)
	resource := f.resource(t)
	destination, err := f.client.CreateDestination(t.Context(), authorized(t, f.destinationRequest(t)))
	require.NoError(t, err)
	destinationID := destination.Msg.GetDestination().GetId()
	_, err = f.client.CreateRule(t.Context(), authorized(t, f.ruleRequest(t,
		&ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS}},
		&ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: destinationID}},
	)))
	require.NoError(t, err)
	scoped := f.ruleRequest(t, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userID}}, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userOtherID}})
	scoped.ProjectId, scoped.AppId, scoped.EnvironmentId = resource.ProjectID, resource.AppID, resource.EnvironmentID
	_, err = f.client.CreateRule(t.Context(), authorized(t, scoped))
	require.NoError(t, err)
	preferences, err := f.client.GetPreferences(t.Context(), authorized(t, &ctrlv1.NotificationGetPreferencesRequest{WorkspaceId: f.workspaceID, UserId: f.userID}))
	require.NoError(t, err)
	require.Len(t, preferences.Msg.GetPreferences(), len(deployment.Definitions()))
	for _, preference := range preferences.Msg.GetPreferences() {
		require.True(t, preference.GetEnabled())
	}
	update := &ctrlv1.NotificationUpdatePreferencesRequest{WorkspaceId: f.workspaceID, UserId: f.userID, Preferences: []*ctrlv1.NotificationPreference{{NotificationType: deployment.Live.Type(), Enabled: false}}, Actor: f.actor}
	_, err = f.client.UpdatePreferences(t.Context(), authorized(t, update))
	require.NoError(t, err)
	_, err = f.client.UpdateDestination(t.Context(), authorized(t, &ctrlv1.NotificationUpdateDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: destinationID, Enabled: proto.Bool(false), Actor: f.actor}))
	require.NoError(t, err)
	preview, err := f.client.PreviewRecipients(t.Context(), authorized(t, &ctrlv1.NotificationPreviewRecipientsRequest{WorkspaceId: f.workspaceID, NotificationType: deployment.Live.Type(), ProjectId: resource.ProjectID, AppId: resource.AppID, EnvironmentId: resource.EnvironmentID}))
	require.NoError(t, err)
	require.Len(t, preview.Msg.GetRecipients(), 3)
	resolver := notificationrouting.Resolver{Database: f.database, Directory: f.directory, Audiences: map[notificationv1.Audience]notificationrouting.AudienceResolver{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS: notificationrouting.WorkspaceAdmins}}
	resolved, err := resolver.Resolve(t.Context(), notification.Event{WorkspaceID: f.workspaceID, Type: deployment.Live.Type(), Resource: resource})
	require.NoError(t, err)
	for index, recipient := range preview.Msg.GetRecipients() {
		require.Equal(t, resolved[index].Key, recipient.GetKey())
		require.Equal(t, string(resolved[index].Channel), recipient.GetChannel())
		require.Equal(t, resolved[index].Reason, recipient.GetReason())
		require.Equal(t, resolved[index].Email, recipient.GetEmail())
		switch {
		case recipient.GetUserId() == f.userID:
			require.Equal(t, "email_opted_out", recipient.GetReason())
		case recipient.GetUserId() == f.userOtherID:
			require.Empty(t, recipient.GetReason())
		case recipient.GetDestinationId() == destinationID:
			require.Equal(t, "destination_paused", recipient.GetReason())
		default:
			t.Fatalf("unexpected recipient: %s", recipient.GetKey())
		}
	}
	other := f.resource(t)
	preview, err = f.client.PreviewRecipients(t.Context(), authorized(t, &ctrlv1.NotificationPreviewRecipientsRequest{WorkspaceId: f.workspaceID, NotificationType: deployment.Live.Type(), ProjectId: other.ProjectID, AppId: other.AppID, EnvironmentId: other.EnvironmentID}))
	require.NoError(t, err)
	require.Len(t, preview.Msg.GetRecipients(), 2)
	_, err = f.client.PreviewRecipients(t.Context(), authorized(t, &ctrlv1.NotificationPreviewRecipientsRequest{WorkspaceId: f.workspaceID, NotificationType: deployment.Live.Type()}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	f.audits.fail.Store(true)
	update.Preferences = []*ctrlv1.NotificationPreference{{NotificationType: deployment.Live.Type(), Enabled: true}, {NotificationType: deployment.Cancelled.Type(), Enabled: false}}
	_, err = f.client.UpdatePreferences(t.Context(), authorized(t, update))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	preferences, err = f.client.GetPreferences(t.Context(), authorized(t, &ctrlv1.NotificationGetPreferencesRequest{WorkspaceId: f.workspaceID, UserId: f.userID}))
	require.NoError(t, err)
	for _, preference := range preferences.Msg.GetPreferences() {
		require.Equal(t, preference.GetNotificationType() != deployment.Live.Type(), preference.GetEnabled())
	}
	update.UserId = f.userOtherID
	_, err = f.client.UpdatePreferences(t.Context(), authorized(t, update))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRulesValidateScopeAndSupportedRecipients(t *testing.T) {
	f := newFixture(t)
	resource, other := f.resource(t), f.resource(t)
	for _, test := range []struct {
		name   string
		change func(*ctrlv1.NotificationCreateRuleRequest)
		code   connect.Code
	}{
		{"unknown event", func(r *ctrlv1.NotificationCreateRuleRequest) { r.NotificationType = "unsupported.event" }, connect.CodeInvalidArgument},
		{"unknown audience", func(r *ctrlv1.NotificationCreateRuleRequest) {
			r.Targets = []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience_AUDIENCE_BILLING_CONTACTS}}}
		}, connect.CodeInvalidArgument},
		{"unknown member", func(r *ctrlv1.NotificationCreateRuleRequest) {
			r.Targets = []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_UserId{UserId: uid.New(uid.TestPrefix)}}}
		}, connect.CodeInvalidArgument},
		{"missing parent", func(r *ctrlv1.NotificationCreateRuleRequest) { r.AppId = resource.AppID }, connect.CodeInvalidArgument},
		{"wrong parent", func(r *ctrlv1.NotificationCreateRuleRequest) { r.ProjectId, r.AppId = other.ProjectID, resource.AppID }, connect.CodeNotFound},
		{"wrong environment", func(r *ctrlv1.NotificationCreateRuleRequest) {
			r.ProjectId, r.AppId, r.EnvironmentId = resource.ProjectID, resource.AppID, other.EnvironmentID
		}, connect.CodeNotFound},
		{"missing destination", func(r *ctrlv1.NotificationCreateRuleRequest) {
			r.Targets = []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: uid.New(uid.TestPrefix)}}}
		}, connect.CodeNotFound},
		{"too many targets", func(r *ctrlv1.NotificationCreateRuleRequest) { r.Targets = make([]*ctrlv1.NotificationTarget, 101) }, connect.CodeInvalidArgument},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := f.ruleRequest(t, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userID}})
			test.change(request)
			_, err := f.client.CreateRule(t.Context(), authorized(t, request))
			require.Equal(t, test.code, connect.CodeOf(err))
		})
	}
	require.NoError(t, f.database.MarkProjectDeleted(t.Context(), db.MarkProjectDeletedParams{ID: resource.ProjectID, DeletedAt: sql.NullInt64{Int64: 1, Valid: true}}))
	request := f.ruleRequest(t, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userID}})
	request.ProjectId = resource.ProjectID
	_, err := f.client.CreateRule(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	listed, err := f.client.ListRules(t.Context(), authorized(t, &ctrlv1.NotificationListRulesRequest{WorkspaceId: f.workspaceID}))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetRules())
	catalog, err := f.client.GetCatalog(t.Context(), authorized(t, &ctrlv1.NotificationGetCatalogRequest{WorkspaceId: f.workspaceID}))
	require.NoError(t, err)
	require.Len(t, catalog.Msg.GetEventTypes(), len(deployment.Definitions()))
	require.Equal(t, []notificationv1.Audience{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS}, catalog.Msg.GetAudiences())
	members, err := f.client.ListMembers(t.Context(), authorized(t, &ctrlv1.NotificationListMembersRequest{WorkspaceId: f.workspaceID, Limit: 1}))
	require.NoError(t, err)
	require.Len(t, members.Msg.GetMembers(), 1)
	require.True(t, members.Msg.GetHasMore())
	last, err := f.client.ListMembers(t.Context(), authorized(t, &ctrlv1.NotificationListMembersRequest{WorkspaceId: f.workspaceID, Limit: 1, Cursor: members.Msg.GetNextCursor()}))
	require.NoError(t, err)
	require.Len(t, last.Msg.GetMembers(), 1)
	require.False(t, last.Msg.GetHasMore())
	require.NotEqual(t, members.Msg.GetMembers()[0].GetUserId(), last.Msg.GetMembers()[0].GetUserId())
}

func TestConcurrentCreatesAreFirstWriteWins(t *testing.T) {
	f := newFixture(t)
	message := f.destinationRequest(t)
	requests := []*connect.Request[ctrlv1.NotificationCreateDestinationRequest]{authorized(t, proto.CloneOf(message)), authorized(t, proto.CloneOf(message))}
	requests[1].Msg.Name = "Competing name"
	results := make([]*connect.Response[ctrlv1.NotificationCreateDestinationResponse], 2)
	errors := make([]error, 2)
	var group sync.WaitGroup
	for index, request := range requests {
		group.Go(func() { results[index], errors[index] = f.client.CreateDestination(t.Context(), request) })
	}
	group.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	require.True(t, proto.Equal(results[0].Msg, results[1].Msg))
	audits, err := f.database.ListClickhouseOutboxByWorkspace(t.Context(), f.workspaceID)
	require.NoError(t, err)
	require.Len(t, audits, 1)
}

func TestRuleIdentityPaginationAndTenantIsolation(t *testing.T) {
	f := newFixture(t)
	request := f.ruleRequest(t, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS}})
	request.IdempotencyKey = strings.Repeat("k", 129)
	_, err := f.client.CreateRule(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	request.IdempotencyKey = strings.Repeat("k", 128)
	created, err := f.client.CreateRule(t.Context(), authorized(t, request))
	require.NoError(t, err)
	request.Name, request.NotificationType = "Should not replace", "unsupported.event"
	retried, err := f.client.CreateRule(t.Context(), authorized(t, request))
	require.NoError(t, err)
	require.True(t, proto.Equal(created.Msg, retried.Msg))
	request.NotificationType = deployment.Live.Type()
	request.IdempotencyKey = uid.New(uid.TestPrefix)
	_, err = f.client.CreateRule(t.Context(), authorized(t, request))
	require.NoError(t, err)
	first, err := f.client.ListRules(t.Context(), authorized(t, &ctrlv1.NotificationListRulesRequest{WorkspaceId: f.workspaceID, Limit: 1}))
	require.NoError(t, err)
	require.Len(t, first.Msg.GetRules(), 1)
	require.True(t, first.Msg.GetHasMore())
	last, err := f.client.ListRules(t.Context(), authorized(t, &ctrlv1.NotificationListRulesRequest{WorkspaceId: f.workspaceID, Limit: 1, Cursor: first.Msg.GetNextCursor()}))
	require.NoError(t, err)
	require.Len(t, last.Msg.GetRules(), 1)
	require.False(t, last.Msg.GetHasMore())
	require.NotEqual(t, first.Msg.GetRules()[0].GetId(), last.Msg.GetRules()[0].GetId())

	workspaceOtherID, orgOtherID := uid.New(uid.WorkspacePrefix), uid.New(uid.OrgPrefix)
	require.NoError(t, f.database.InsertWorkspace(t.Context(), db.InsertWorkspaceParams{ID: workspaceOtherID, OrgID: orgOtherID, Name: "Other", Slug: uid.New(uid.TestPrefix), K8sNamespace: uid.DNS1035()}))
	f.directory.byOrg[orgOtherID] = f.directory.byOrg[f.orgID]
	ruleID := created.Msg.GetRule().GetId()
	_, err = f.client.GetRule(t.Context(), authorized(t, &ctrlv1.NotificationGetRuleRequest{WorkspaceId: workspaceOtherID, RuleId: ruleID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = f.client.UpdateRule(t.Context(), authorized(t, &ctrlv1.NotificationUpdateRuleRequest{WorkspaceId: workspaceOtherID, RuleId: ruleID, Actor: f.actor}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = f.client.DeleteRule(t.Context(), authorized(t, &ctrlv1.NotificationDeleteRuleRequest{WorkspaceId: workspaceOtherID, RuleId: ruleID, Actor: f.actor}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	destinationRequest := f.destinationRequest(t)
	destinationRequest.IdempotencyKey = strings.Repeat("k", 129)
	_, err = f.client.CreateDestination(t.Context(), authorized(t, destinationRequest))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	destinationRequest.IdempotencyKey = strings.Repeat("k", 128)
	destination, err := f.client.CreateDestination(t.Context(), authorized(t, destinationRequest))
	require.NoError(t, err)
	request.WorkspaceId = workspaceOtherID
	request.Targets = []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: destination.Msg.GetDestination().GetId()}}}
	_, err = f.client.CreateRule(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	resource := f.resource(t)
	request.Targets = []*ctrlv1.NotificationTarget{{Target: &ctrlv1.NotificationTarget_UserId{UserId: f.userID}}}
	request.ProjectId = resource.ProjectID
	_, err = f.client.CreateRule(t.Context(), authorized(t, request))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = f.client.UpdatePreferences(t.Context(), authorized(t, &ctrlv1.NotificationUpdatePreferencesRequest{WorkspaceId: f.workspaceID, UserId: f.userID, Actor: f.actor, Preferences: []*ctrlv1.NotificationPreference{{NotificationType: deployment.Live.Type(), Enabled: false}}}))
	require.NoError(t, err)
	preferences, err := f.client.GetPreferences(t.Context(), authorized(t, &ctrlv1.NotificationGetPreferencesRequest{WorkspaceId: workspaceOtherID, UserId: f.userID}))
	require.NoError(t, err)
	for _, preference := range preferences.Msg.GetPreferences() {
		require.True(t, preference.GetEnabled())
	}
}

func TestDeletingDestinationCannotRaceNewRuleReference(t *testing.T) {
	f := newFixture(t)
	destination, err := f.client.CreateDestination(t.Context(), authorized(t, f.destinationRequest(t)))
	require.NoError(t, err)
	id := destination.Msg.GetDestination().GetId()
	create := authorized(t, f.ruleRequest(t, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: id}}))
	remove := authorized(t, &ctrlv1.NotificationDeleteDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: id, Actor: f.actor})
	var createError, deleteError error
	var group sync.WaitGroup
	group.Go(func() { _, createError = f.client.CreateRule(t.Context(), create) })
	group.Go(func() { _, deleteError = f.client.DeleteDestination(t.Context(), remove) })
	group.Wait()
	if createError == nil {
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(deleteError))
		_, err := f.client.GetDestination(t.Context(), authorized(t, &ctrlv1.NotificationGetDestinationRequest{WorkspaceId: f.workspaceID, DestinationId: id}))
		require.NoError(t, err)
	} else {
		require.NoError(t, deleteError)
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(createError))
		rules, err := f.client.ListRules(t.Context(), authorized(t, &ctrlv1.NotificationListRulesRequest{WorkspaceId: f.workspaceID}))
		require.NoError(t, err)
		require.Empty(t, rules.Msg.GetRules())
	}
}
