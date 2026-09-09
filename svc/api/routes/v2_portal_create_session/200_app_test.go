package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// seededApp identifies the rows seedAppWithKeyspaces created. The project and
// environment ids are here because a custom_domains row requires both.
type seededApp struct {
	AppID         string
	ProjectID     string
	EnvironmentID string
}

// appMapping addresses a portal at an app rather than a keyspace.
func appMapping(appID string) portal.Mapping {
	return portal.Mapping{Type: portal.MappingTypeApp, ID: appID}
}

// seedAppWithKeyspaces creates an app, environment and current deployment whose
// gateway policy config verifies keys against the given keyspaces. This is the
// shape an app-mapped portal resolves its keyspaces from.
//
// projectID is the project the app lives in, which is also the project the
// portal derives. Pass the project owning the keyspaces unless the test is
// exercising the cross-project refusal; an empty value creates a fresh project,
// which puts the app in a different project from any keyspace seeded elsewhere.
func seedAppWithKeyspaces(t *testing.T, h *testutil.Harness, workspaceID, slugBase, projectID string, keyspaceIDs []string) seededApp {
	t.Helper()

	suffix := uid.DNS1035()

	if projectID == "" {
		projectID = h.CreateProject(seed.CreateProjectRequest{
			WorkspaceID:      workspaceID,
			Name:             slugBase + "-project",
			ID:               uid.New(uid.ProjectPrefix),
			Slug:             slugBase + "-project-" + suffix,
			DeleteProtection: false,
		}).ID
	}
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        slugBase + " app",
		Slug:        slugBase + "-app-" + suffix,
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
		Description: "production environment",
	})

	seeded := seededApp{
		AppID:         app.ID,
		ProjectID:     projectID,
		EnvironmentID: environment.ID,
	}
	redeployAppWithKeyspaces(t, h, workspaceID, seeded, keyspaceIDs)

	return seeded
}

// redeployAppWithKeyspaces points an app at a fresh deployment whose gateway
// policy config verifies keys against the given keyspaces, which is how a
// redeploy changes the keyspaces an app-mapped portal resolves to.
func redeployAppWithKeyspaces(t *testing.T, h *testutil.Harness, workspaceID string, app seededApp, keyspaceIDs []string) {
	t.Helper()

	ctx := context.Background()
	now := time.Now().UnixMilli()

	policyConfig, err := protojson.Marshal(&frontlinev1.Config{
		Policies: []*frontlinev1.Policy{
			{
				Id:      "pol_keyauth",
				Name:    "keyauth",
				Enabled: proto.Bool(true),
				Config: &frontlinev1.Policy_Keyauth{
					Keyauth: &frontlinev1.KeyAuth{
						KeySpaceIds: keyspaceIDs,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.NoError(t, db.Query.InsertDeployment(ctx, h.DB.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       "test-" + deploymentID,
		WorkspaceID:                   workspaceID,
		ProjectID:                     app.ProjectID,
		AppID:                         app.AppID,
		EnvironmentID:                 app.EnvironmentID,
		Source:                        db.DeploymentsSourceUnknown,
		ImageRequested:                sql.NullString{Valid: false},
		SentinelConfig:                policyConfig,
		EncryptedEnvironmentVariables: []byte{},
		Status:                        mysqltype.DeploymentsStatusReady,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		DeploymentTrigger:             db.DeploymentsTriggerUnknown,
		CreatedAt:                     now,
	}))

	// The app must point at this deployment so createSession can find its config.
	require.NoError(t, db.Query.UpdateAppDeployments(ctx, h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
		AppID:               app.AppID,
	}))
}

// TestCreateSessionAppMapped verifies that an app-mapped portal config resolves
// its keyspaces from the app's current deployment policy config (the keyauth
// policies' keySpaceIds) rather than from the public request.
func TestCreateSessionAppMapped(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID

	// The keyspace the app verifies keys against. It is seeded through an api so
	// the keyspace has an owning api: every scope requirement is api-scoped, and
	// a keyspace with no api is a misconfiguration the handler rejects.
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	keySpaceID := api.KeyAuthID.String

	appID := seedAppWithKeyspaces(t, h, workspaceID, "portal-app", api.ProjectID, []string{keySpaceID}).AppID

	h.SeedPortal(t, workspaceID, "app-portal", "app-portal", appMapping(appID), nil, nil)

	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Portal:     "app-portal",
		ExternalId: "user_app",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.Id)

	// The persisted grant must be scoped to the keyspace resolved from the app's
	// policy config, not anything in the request.
	code := exchangeCodeFromURL(t, res.Body.Data.Url)
	session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), hash.Sha256(code))
	require.NoError(t, err)

	var grant struct {
		KeyspaceIDs []string `json:"keyspaceIds"`
		Scopes      []string `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal(session.Scopes, &grant))
	require.Equal(t, []string{keySpaceID}, grant.KeyspaceIDs)
}

// TestCreateSessionAppMappedIgnoresAppCustomDomain pins the session URL to the
// configured base URL. A custom domain on the portal's app is a deploy domain:
// frontline routes it to that app's container, so it never reaches a portal.
// Branded portal URLs will arrive as a caller-supplied hostname validated
// against the portal's own domains, not by reading this row.
func TestCreateSessionAppMappedIgnoresAppCustomDomain(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})

	app := seedAppWithKeyspaces(t, h, workspaceID, "branded", api.ProjectID, []string{api.KeyAuthID.String})

	// domain and target_cname both carry unique constraints, and the test
	// database outlives a single run.
	require.NoError(t, db.Query.InsertCustomDomain(ctx, h.DB.RW(), db.InsertCustomDomainParams{
		ID:                    uid.New(uid.DomainPrefix),
		WorkspaceID:           workspaceID,
		ProjectID:             app.ProjectID,
		AppID:                 app.AppID,
		EnvironmentID:         app.EnvironmentID,
		Domain:                "keys-" + uid.DNS1035() + ".example.com",
		ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus:    db.CustomDomainsVerificationStatusVerified,
		VerificationToken:     "tok_" + uid.DNS1035(),
		OwnershipVerified:     true,
		CnameVerified:         true,
		TargetCname:           "cname-" + uid.DNS1035() + ".unkey.app",
		VerificationError:     sql.NullString{Valid: false, String: ""},
		DomainConnectProvider: sql.NullString{Valid: false, String: ""},
		DomainConnectUrl:      sql.NullString{Valid: false, String: ""},
		LastCheckedAt:         sql.NullInt64{Valid: false, Int64: 0},
		CreatedAt:             time.Now().UnixMilli(),
	}))

	h.SeedPortal(t, workspaceID, "branded-portal", "branded-portal", appMapping(app.AppID), nil, nil)

	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "branded-portal",
		ExternalId: "user_branded",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)

	parsed, err := url.Parse(res.Body.Data.Url)
	require.NoError(t, err)
	require.Equal(t, "portal.unkey.com", parsed.Host)

	// The URL is still well formed, so the code remains usable.
	require.NotEmpty(t, exchangeCodeFromURL(t, res.Body.Data.Url))
}

// TestCreateSessionAppMappedKeyspaceGrowth guarantees the ceiling is evaluated
// against the keyspaces the app resolves to *now*: a mint that succeeded before
// the app started verifying a second keyspace stops succeeding afterwards.
func TestCreateSessionAppMappedKeyspaceGrowth(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	granted := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	added := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})

	appID := seedAppWithKeyspaces(t, h, workspaceID, "growth", granted.ProjectID, []string{granted.KeyAuthID.String}).AppID
	h.SeedPortal(t, workspaceID, "growth-portal", "growth-portal", appMapping(appID), nil, nil)

	// The grant covers only the keyspace the app verifies today.
	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		fmt.Sprintf("api.%s.read_key", granted.ID),
		fmt.Sprintf("api.%s.read_api", granted.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	req := handler.Request{
		Portal:     "growth-portal",
		ExternalId: "user_growth",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	// The app now verifies a second keyspace the grant does not cover.
	secondAppID := seedAppWithKeyspaces(t, h, workspaceID, "growth-2", granted.ProjectID, []string{
		granted.KeyAuthID.String,
		added.KeyAuthID.String,
	}).AppID
	h.SeedPortal(t, workspaceID, "growth-portal-2", "growth-portal-2", appMapping(secondAppID), nil, nil)

	grown := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     "growth-portal-2",
		ExternalId: "user_growth",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, grown.Status,
		"a keyspace added after the grant was issued must stop the mint, got: %s", grown.RawBody)
}
