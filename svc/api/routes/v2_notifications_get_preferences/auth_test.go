package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	authjwt "github.com/unkeyed/unkey/pkg/auth/jwt"
	rootkey "github.com/unkeyed/unkey/pkg/auth/root_key"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	createDestination "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_create_destination"
	createRule "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_create_rule"
	getCatalog "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_get_catalog"
	getDestination "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_get_destination"
	getPreferences "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_get_preferences"
	getRule "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_get_rule"
	listDestinations "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_list_destinations"
	previewRecipients "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_preview_recipients"
	testDestination "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_test_destination"
	updateDestination "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_update_destination"
	updatePreferences "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_update_preferences"
)

type notificationService struct {
	ctrlv1connect.UnimplementedNotificationServiceHandler
	workspaceID   string
	destinationID string
	preferences   chan *ctrlv1.NotificationUpdatePreferencesRequest
	lists         chan *ctrlv1.NotificationListDestinationsRequest
	creates       chan *ctrlv1.NotificationCreateDestinationRequest
	updates       chan *ctrlv1.NotificationUpdateDestinationRequest
	tests         chan *ctrlv1.NotificationTestDestinationRequest
	rules         chan *ctrlv1.NotificationCreateRuleRequest
}

func (n *notificationService) GetCatalog(context.Context, *connect.Request[ctrlv1.NotificationGetCatalogRequest]) (*connect.Response[ctrlv1.NotificationGetCatalogResponse], error) {
	types := make([]string, 0)
	for _, definition := range deployment.Definitions() {
		types = append(types, definition.Type())
	}
	return connect.NewResponse(&ctrlv1.NotificationGetCatalogResponse{
		EventTypes: types,
		Channels:   []string{"email", "slack", "discord", "webhook"},
		Audiences:  []notificationv1.Audience{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS},
	}), nil
}

func (n *notificationService) GetPreferences(_ context.Context, req *connect.Request[ctrlv1.NotificationGetPreferencesRequest]) (*connect.Response[ctrlv1.NotificationGetPreferencesResponse], error) {
	n.preferences <- &ctrlv1.NotificationUpdatePreferencesRequest{WorkspaceId: req.Msg.GetWorkspaceId(), UserId: req.Msg.GetUserId()}
	return connect.NewResponse(&ctrlv1.NotificationGetPreferencesResponse{Preferences: []*ctrlv1.NotificationPreference{{NotificationType: deployment.Definitions()[0].Type(), Enabled: true}}}), nil
}

func (n *notificationService) UpdatePreferences(_ context.Context, req *connect.Request[ctrlv1.NotificationUpdatePreferencesRequest]) (*connect.Response[ctrlv1.NotificationUpdatePreferencesResponse], error) {
	n.preferences <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationUpdatePreferencesResponse{Preferences: req.Msg.GetPreferences()}), nil
}

func (n *notificationService) ListDestinations(_ context.Context, req *connect.Request[ctrlv1.NotificationListDestinationsRequest]) (*connect.Response[ctrlv1.NotificationListDestinationsResponse], error) {
	n.lists <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationListDestinationsResponse{}), nil
}

func (n *notificationService) GetDestination(_ context.Context, req *connect.Request[ctrlv1.NotificationGetDestinationRequest]) (*connect.Response[ctrlv1.NotificationGetDestinationResponse], error) {
	if req.Msg.GetWorkspaceId() != n.workspaceID || req.Msg.GetDestinationId() != n.destinationID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("destination missing"))
	}
	return connect.NewResponse(&ctrlv1.NotificationGetDestinationResponse{Destination: n.destination()}), nil
}

func (n *notificationService) GetRule(context.Context, *connect.Request[ctrlv1.NotificationGetRuleRequest]) (*connect.Response[ctrlv1.NotificationGetRuleResponse], error) {
	return nil, connect.NewError(connect.CodeNotFound, errors.New("rule missing"))
}

func (n *notificationService) CreateDestination(_ context.Context, req *connect.Request[ctrlv1.NotificationCreateDestinationRequest]) (*connect.Response[ctrlv1.NotificationCreateDestinationResponse], error) {
	n.creates <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationCreateDestinationResponse{Destination: n.destination()}), nil
}

func (n *notificationService) UpdateDestination(_ context.Context, req *connect.Request[ctrlv1.NotificationUpdateDestinationRequest]) (*connect.Response[ctrlv1.NotificationUpdateDestinationResponse], error) {
	n.updates <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationUpdateDestinationResponse{Destination: n.destination()}), nil
}

func (n *notificationService) TestDestination(_ context.Context, req *connect.Request[ctrlv1.NotificationTestDestinationRequest]) (*connect.Response[ctrlv1.NotificationTestDestinationResponse], error) {
	n.tests <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationTestDestinationResponse{DeliveryId: uid.New(uid.TestPrefix), Status: "accepted"}), nil
}

func (n *notificationService) CreateRule(_ context.Context, req *connect.Request[ctrlv1.NotificationCreateRuleRequest]) (*connect.Response[ctrlv1.NotificationCreateRuleResponse], error) {
	n.rules <- req.Msg
	return connect.NewResponse(&ctrlv1.NotificationCreateRuleResponse{Rule: &ctrlv1.NotificationRule{
		Id:               uid.New(uid.TestPrefix),
		Name:             req.Msg.GetName(),
		NotificationType: req.Msg.GetNotificationType(),
		ProjectId:        req.Msg.GetProjectId(),
		AppId:            req.Msg.GetAppId(),
		EnvironmentId:    req.Msg.GetEnvironmentId(),
		Enabled:          req.Msg.GetEnabled(),
		Targets:          req.Msg.GetTargets(),
	}}), nil
}

func (n *notificationService) destination() *ctrlv1.NotificationDestination {
	return &ctrlv1.NotificationDestination{Id: n.destinationID, Name: "Operations", Channel: "webhook", Enabled: true, UrlHint: "https://example.com", HeaderNames: []string{"Authorization"}}
}

type cliSessions struct {
	id      string
	session clisession.Session
}

func (c cliSessions) Allow(_ context.Context, id string) (clisession.Session, error) {
	if id != c.id {
		return clisession.Session{}, clisession.ErrInactive
	}
	return c.session, nil
}

func TestPublicNotificationAuthenticationAndTenantBoundaries(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	workspaceOther := h.CreateWorkspace()
	service := &notificationService{
		workspaceID:   workspaceID,
		destinationID: uid.New(uid.TestPrefix),
		preferences:   make(chan *ctrlv1.NotificationUpdatePreferencesRequest, 20),
		lists:         make(chan *ctrlv1.NotificationListDestinationsRequest, 20),
		creates:       make(chan *ctrlv1.NotificationCreateDestinationRequest, 20),
		updates:       make(chan *ctrlv1.NotificationUpdateDestinationRequest, 20),
		tests:         make(chan *ctrlv1.NotificationTestDestinationRequest, 20),
		rules:         make(chan *ctrlv1.NotificationCreateRuleRequest, 20),
	}
	_, rpcHandler := ctrlv1connect.NewNotificationServiceHandler(service)
	rpcServer := httptest.NewServer(rpcHandler)
	t.Cleanup(rpcServer.Close)
	client := ctrlv1connect.NewNotificationServiceClient(rpcServer.Client(), rpcServer.URL)

	secret := []byte(uid.Secure())
	orgID := uid.New(uid.OrgPrefix)
	orgOther := uid.New(uid.OrgPrefix)
	resolver, err := authjwt.NewResolver(authjwt.WorkspaceLookupFunc(func(_ context.Context, org string) (string, error) {
		switch org {
		case orgID:
			return workspaceID, nil
		case orgOther:
			return workspaceOther.ID, nil
		default:
			return "", authjwt.ErrWorkspaceNotFound
		}
	}), "notification-api-test", authjwt.Audience, secret)
	require.NoError(t, err)
	signer, err := tokenjwt.NewHS256Signer[authjwt.Claims](secret)
	require.NoError(t, err)
	userID := uid.New(uid.TestPrefix)
	cliID := uid.New(uid.CLISessionPrefix)
	cliResolver, err := clisession.NewResolver(cliSessions{
		id:      cliID,
		session: clisession.Session{UserID: userID, WorkspaceID: workspaceID, DeviceName: "test", ExpiresAt: h.Clock.Now().Add(time.Hour)},
	}, h.Clock, secret)
	require.NoError(t, err)
	cliSigner, err := tokenjwt.NewHS256Signer[clisession.Claims](secret)
	require.NoError(t, err)
	cliToken, err := cliSigner.Sign(clisession.Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{Issuer: clisession.Issuer, Subject: userID, Audience: []string{clisession.Audience}, IssuedAt: h.Clock.Now().Unix(), ExpiresAt: h.Clock.Now().Add(time.Hour).Unix()},
		WorkspaceID:      workspaceID,
		SessionID:        cliID,
		Permissions:      []string{"unkey:v1:" + workspaceID + ":limits#read"},
		TokenUse:         clisession.TokenUse,
	})
	require.NoError(t, err)
	stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
	stack = append(stack, middleware.WithAuthentication(middleware.AuthenticationConfig{
		Auth:             auth.New(cliResolver, workos.NewRoleMappingResolver(resolver), rootkey.NewResolver(h.Keys)),
		KeyVerifications: h.KeyVerifications,
		Region:           "test",
		Database:         h.DB,
		LimitsCache:      h.Caches.WorkspaceLimits,
		Ratelimit:        h.Ratelimit,
	}))
	get := &getPreferences.Handler{Notifications: client}
	update := &updatePreferences.Handler{Notifications: client}
	list := &listDestinations.Handler{Notifications: client}
	destination := &getDestination.Handler{Notifications: client}
	rule := &getRule.Handler{Notifications: client}
	create := &createDestination.Handler{Notifications: client}
	edit := &updateDestination.Handler{Notifications: client}
	test := &testDestination.Handler{Notifications: client}
	preview := &previewRecipients.Handler{Notifications: client}
	catalog := &getCatalog.Handler{Notifications: client}
	createRoutingRule := &createRule.Handler{Notifications: client}
	for _, route := range []zen.Route{get, update, list, destination, rule, create, edit, test, preview, catalog, createRoutingRule} {
		h.Register(route, stack...)
	}
	jwtHeaders := func(t *testing.T, org, user string, roles []string) http.Header {
		t.Helper()
		token, err := signer.Sign(authjwt.Claims{
			RegisteredClaims: tokenjwt.RegisteredClaims{Issuer: "notification-api-test", Subject: user, Audience: []string{authjwt.Audience}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Minute).Unix()},
			Org:              authjwt.OrganizationClaims{ID: org},
			Roles:            roles,
		})
		require.NoError(t, err)
		return testutil.RootKeyHeaders(token)
	}
	root := testutil.RootKeyHeaders(h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*"))
	portal := h.CreatePortalSession(workspaceID, userID, nil, []string{"keys:read"})
	for _, tc := range []struct {
		name    string
		headers http.Header
		status  int
	}{
		{name: "root wildcard", headers: root, status: http.StatusForbidden},
		{name: "portal session cookie", headers: portal, status: http.StatusBadRequest},
		{name: "dashboard viewer", headers: jwtHeaders(t, orgID, userID, []string{"viewer"}), status: http.StatusOK},
		{name: "dashboard without roles", headers: jwtHeaders(t, orgID, userID, nil), status: http.StatusForbidden},
		{name: "cli user", headers: testutil.RootKeyHeaders(cliToken), status: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := testutil.CallRoute[getPreferences.Request, getPreferences.Response](h, get, tc.headers, getPreferences.Request{})
			require.Equal(t, tc.status, response.Status, response.RawBody)
			if tc.status == http.StatusOK {
				forwarded := <-service.preferences
				require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
				require.Equal(t, userID, forwarded.GetUserId())
			}
			updated := testutil.CallRoute[updatePreferences.Request, updatePreferences.Response](h, update, tc.headers, updatePreferences.Request{
				Preferences: []openapi.NotificationPreference{{NotificationType: deployment.Definitions()[0].Type(), Enabled: false}},
			})
			require.Equal(t, tc.status, updated.Status, updated.RawBody)
			if tc.status == http.StatusOK {
				forwarded := <-service.preferences
				require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
				require.Equal(t, userID, forwarded.GetUserId())
				require.Equal(t, userID, forwarded.GetActor().GetId())
				require.False(t, updated.Body.Data[0].Enabled)
			}
			require.Empty(t, service.preferences)
		})
	}

	t.Run("no request authority", func(t *testing.T) {
		type forgedUpdate struct {
			updatePreferences.Request
			UserID      string `json:"userId"`
			WorkspaceID string `json:"workspaceId"`
		}
		res := testutil.CallRoute[forgedUpdate, openapi.BadRequestErrorResponse](h, update, jwtHeaders(t, orgID, userID, []string{"admin"}), forgedUpdate{
			Request:     updatePreferences.Request{Preferences: []openapi.NotificationPreference{}},
			UserID:      uid.New(uid.TestPrefix),
			WorkspaceID: workspaceOther.ID,
		})
		require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
		require.Empty(t, service.preferences)
		otherUser := uid.New(uid.TestPrefix)
		read := testutil.CallRoute[getPreferences.Request, getPreferences.Response](h, get, jwtHeaders(t, orgOther, otherUser, []string{"viewer"}), getPreferences.Request{})
		require.Equal(t, http.StatusOK, read.Status, read.RawBody)
		forwarded := <-service.preferences
		require.Equal(t, workspaceOther.ID, forwarded.GetWorkspaceId())
		require.Equal(t, otherUser, forwarded.GetUserId())
	})

	t.Run("shared management defaults to admin", func(t *testing.T) {
		for _, role := range []string{"admin", "developer", "basic_member", "viewer"} {
			res := testutil.CallRoute[listDestinations.Request, listDestinations.Response](h, list, jwtHeaders(t, orgID, userID, []string{role}), listDestinations.Request{})
			if role == "admin" {
				require.Equal(t, http.StatusOK, res.Status, res.RawBody)
				forwarded := <-service.lists
				require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
				require.EqualValues(t, 50, forwarded.GetLimit())
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
			}
		}
		require.Empty(t, service.lists)
	})

	t.Run("catalog needs authentication but not management permission", func(t *testing.T) {
		for _, headers := range []http.Header{
			testutil.RootKeyHeaders(h.CreateRootKey(workspaceID)),
			jwtHeaders(t, orgID, userID, []string{"viewer"}),
		} {
			res := testutil.CallRoute[getCatalog.Request, getCatalog.Response](h, catalog, headers, getCatalog.Request{})
			require.Equal(t, http.StatusOK, res.Status, res.RawBody)
			require.Len(t, res.Body.Data.EventTypes, len(deployment.Definitions()))
			require.Equal(t, []openapi.V2NotificationsGetCatalogResponseBodyDataAudiences{openapi.V2NotificationsGetCatalogResponseBodyDataAudiencesWorkspaceAdmins}, res.Body.Data.Audiences)
		}
	})

	t.Run("parameterless reads accept SDK requests without a body", func(t *testing.T) {
		headers := jwtHeaders(t, orgID, userID, []string{"viewer"})
		catalogRequest := httptest.NewRequest(catalog.Method(), catalog.Path(), nil)
		catalogRequest.Header = headers.Clone()
		catalogResponse := testutil.CallRaw[getCatalog.Response](h, catalogRequest)
		require.Equal(t, http.StatusOK, catalogResponse.Status, catalogResponse.RawBody)
		require.Len(t, catalogResponse.Body.Data.EventTypes, len(deployment.Definitions()))

		preferencesRequest := httptest.NewRequest(get.Method(), get.Path(), nil)
		preferencesRequest.Header = headers.Clone()
		preferencesResponse := testutil.CallRaw[getPreferences.Response](h, preferencesRequest)
		require.Equal(t, http.StatusOK, preferencesResponse.Status, preferencesResponse.RawBody)
		require.True(t, preferencesResponse.Body.Data[0].Enabled)
		forwarded := <-service.preferences
		require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
		require.Equal(t, userID, forwarded.GetUserId())
	})

	t.Run("rule target variants are validated on the HTTP route", func(t *testing.T) {
		var target openapi.NotificationTarget
		require.NoError(t, target.FromNotificationAudienceTarget(openapi.NotificationAudienceTarget{
			Type:     openapi.Audience,
			Audience: openapi.NotificationAudienceTargetAudienceWorkspaceAdmins,
		}))
		request := createRule.Request{
			Name:             "Operations",
			NotificationType: deployment.Definitions()[0].Type(),
			Enabled:          true,
			Targets:          []openapi.NotificationTarget{target},
			IdempotencyKey:   uid.New(uid.TestPrefix),
		}
		res := testutil.CallRoute[createRule.Request, createRule.Response](h, createRoutingRule, root, request)
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		forwarded := <-service.rules
		require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
		require.Equal(t, notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS, forwarded.GetTargets()[0].GetAudience())
		require.Equal(t, request.IdempotencyKey, forwarded.GetIdempotencyKey())
		require.NoError(t, target.FromNotificationAudienceTarget(openapi.NotificationAudienceTarget{Type: openapi.Audience, Audience: "billing_contacts"}))
		request.Targets = []openapi.NotificationTarget{target}
		invalid := testutil.CallRoute[createRule.Request, openapi.BadRequestErrorResponse](h, createRoutingRule, root, request)
		require.Equal(t, http.StatusBadRequest, invalid.Status, invalid.RawBody)
		require.Empty(t, service.rules)
	})

	t.Run("root permissions are notification specific and tenant bounded", func(t *testing.T) {
		for _, permission := range []string{
			"unkey:v1:" + workspaceID + ":notifications#read",
			"unkey:v1:" + workspaceID + ":flags#read",
			"unkey:v1:" + workspaceOther.ID + ":notifications#read",
		} {
			headers := testutil.RootKeyHeaders(h.CreateRootKey(workspaceID, permission))
			res := testutil.CallRoute[listDestinations.Request, listDestinations.Response](h, list, headers, listDestinations.Request{})
			if permission == "unkey:v1:"+workspaceID+":notifications#read" {
				require.Equal(t, http.StatusOK, res.Status, res.RawBody)
				require.Equal(t, workspaceID, (<-service.lists).GetWorkspaceId())
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
			}
		}
	})

	t.Run("missing unreadable and cross tenant reads match", func(t *testing.T) {
		missing := testutil.CallRoute[getDestination.Request, openapi.NotFoundErrorResponse](h, destination, root, getDestination.Request{DestinationId: uid.New(uid.TestPrefix)})
		unreadable := testutil.CallRoute[getDestination.Request, openapi.NotFoundErrorResponse](h, destination, testutil.RootKeyHeaders(h.CreateRootKey(workspaceID)), getDestination.Request{DestinationId: service.destinationID})
		other := testutil.CallRoute[getDestination.Request, openapi.NotFoundErrorResponse](h, destination, testutil.RootKeyHeaders(h.CreateRootKey(workspaceOther.ID, "unkey:v1:"+workspaceOther.ID+":notifications#read")), getDestination.Request{DestinationId: service.destinationID})
		require.Equal(t, http.StatusNotFound, missing.Status, missing.RawBody)
		require.Equal(t, openapi.BaseError{
			Title:  "Not Found",
			Type:   codes.Data.Notification.NotFound.DocsURL(),
			Detail: "Notification resource not found.",
			Status: http.StatusNotFound,
		}, missing.Body.Error)
		require.Equal(t, missing.Status, unreadable.Status)
		require.Equal(t, missing.Status, other.Status)
		require.Equal(t, missing.Body.Error, unreadable.Body.Error)
		require.Equal(t, missing.Body.Error, other.Body.Error)
		missingRule := testutil.CallRoute[getRule.Request, openapi.NotFoundErrorResponse](h, rule, root, getRule.Request{RuleId: uid.New(uid.TestPrefix)})
		unreadableRule := testutil.CallRoute[getRule.Request, openapi.NotFoundErrorResponse](h, rule, testutil.RootKeyHeaders(h.CreateRootKey(workspaceID)), getRule.Request{RuleId: uid.New(uid.TestPrefix)})
		require.Equal(t, http.StatusNotFound, missingRule.Status)
		require.Equal(t, missing.Body.Error, missingRule.Body.Error)
		require.Equal(t, missingRule.Status, unreadableRule.Status)
		require.Equal(t, missingRule.Body.Error, unreadableRule.Body.Error)
	})

	t.Run("defaults secrets and actor are forwarded but never returned", func(t *testing.T) {
		request := createDestination.Request{Name: "Operations", Channel: openapi.Webhook, Url: new("https://example.com/SECRET"), SigningSecret: new(strings.Repeat("SECRET", 8)), IdempotencyKey: uid.New(uid.TestPrefix)}
		res := testutil.CallRoute[createDestination.Request, createDestination.Response](h, create, root, request)
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		forwarded := <-service.creates
		require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
		require.True(t, forwarded.GetEnabled())
		require.Equal(t, *request.SigningSecret, forwarded.GetSigningSecret())
		require.Equal(t, request.IdempotencyKey, forwarded.GetIdempotencyKey())
		require.NotEmpty(t, forwarded.GetActor().GetId())
		require.Equal(t, "https://example.com", res.Body.Data.UrlHint)
		require.NotContains(t, string(res.RawBody), "SECRET")
		for _, headers := range []*[]openapi.NotificationHeader{nil, new([]openapi.NotificationHeader{})} {
			updated := testutil.CallRoute[updateDestination.Request, updateDestination.Response](h, edit, root, updateDestination.Request{DestinationId: service.destinationID, Headers: headers})
			require.Equal(t, http.StatusOK, updated.Status, updated.RawBody)
			forwarded := <-service.updates
			require.Equal(t, headers == nil, forwarded.Headers == nil)
		}
	})

	t.Run("test send forwards durable identity and reports provider acceptance", func(t *testing.T) {
		request := testDestination.Request{DestinationId: service.destinationID, IdempotencyKey: uid.New(uid.TestPrefix)}
		res := testutil.CallRoute[testDestination.Request, testDestination.Response](h, test, root, request)
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		require.Equal(t, openapi.NotificationDeliveryAccepted, res.Body.Data.Status)
		forwarded := <-service.tests
		require.Equal(t, workspaceID, forwarded.GetWorkspaceId())
		require.Equal(t, request.DestinationId, forwarded.GetDestinationId())
		require.Equal(t, request.IdempotencyKey, forwarded.GetIdempotencyKey())
		require.NotEmpty(t, forwarded.GetActor().GetId())
	})

	t.Run("test send requires write permission and a saved destination", func(t *testing.T) {
		request := testDestination.Request{DestinationId: service.destinationID, IdempotencyKey: uid.New(uid.TestPrefix)}
		readOnly := testutil.RootKeyHeaders(h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":notifications#read"))
		denied := testutil.CallRoute[testDestination.Request, openapi.ForbiddenErrorResponse](h, test, readOnly, request)
		require.Equal(t, http.StatusForbidden, denied.Status, denied.RawBody)
		type draftRequest struct {
			testDestination.Request
			URL string `json:"url"`
		}
		draft := testutil.CallRoute[draftRequest, openapi.BadRequestErrorResponse](h, test, root, draftRequest{
			Request: request,
			URL:     "https://example.com/SECRET",
		})
		require.Equal(t, http.StatusBadRequest, draft.Status, draft.RawBody)
		require.NotContains(t, draft.RawBody, "SECRET")
		require.Empty(t, service.tests)
	})

	t.Run("all idempotent writes enforce key identity and byte limits", func(t *testing.T) {
		var target openapi.NotificationTarget
		require.NoError(t, target.FromNotificationAudienceTarget(openapi.NotificationAudienceTarget{
			Type:     openapi.Audience,
			Audience: openapi.NotificationAudienceTargetAudienceWorkspaceAdmins,
		}))
		for _, tc := range []struct {
			key    string
			status int
		}{
			{key: " key", status: http.StatusBadRequest},
			{key: "key ", status: http.StatusBadRequest},
			{key: "key\n", status: http.StatusBadRequest},
			{key: "\u00a0key", status: http.StatusBadRequest},
			{key: "key\u2003", status: http.StatusBadRequest},
			{key: "key\x00suffix", status: http.StatusBadRequest},
			{key: strings.Repeat("é", 64) + "a", status: http.StatusBadRequest},
			{key: strings.Repeat("é", 64), status: http.StatusOK},
			{key: "key with spaces", status: http.StatusOK},
			{key: "a", status: http.StatusOK},
		} {
			created := testutil.CallRoute[createDestination.Request, createDestination.Response](h, create, root, createDestination.Request{
				Name:           "Operations",
				Channel:        openapi.Slack,
				Url:            new("https://example.com/SECRET"),
				IdempotencyKey: tc.key,
			})
			require.Equal(t, tc.status, created.Status, created.RawBody)
			createdRule := testutil.CallRoute[createRule.Request, createRule.Response](h, createRoutingRule, root, createRule.Request{
				Name:             "Operations",
				NotificationType: deployment.Definitions()[0].Type(),
				Enabled:          true,
				Targets:          []openapi.NotificationTarget{target},
				IdempotencyKey:   tc.key,
			})
			require.Equal(t, tc.status, createdRule.Status, createdRule.RawBody)
			tested := testutil.CallRoute[testDestination.Request, testDestination.Response](h, test, root, testDestination.Request{
				DestinationId:  service.destinationID,
				IdempotencyKey: tc.key,
			})
			require.Equal(t, tc.status, tested.Status, tested.RawBody)
			if tc.status == http.StatusOK {
				require.Equal(t, tc.key, (<-service.creates).GetIdempotencyKey())
				require.Equal(t, tc.key, (<-service.rules).GetIdempotencyKey())
				require.Equal(t, tc.key, (<-service.tests).GetIdempotencyKey())
			}
			require.Empty(t, service.creates)
			require.Empty(t, service.rules)
			require.Empty(t, service.tests)
		}
	})

	t.Run("real route validation rejects invalid parameters", func(t *testing.T) {
		for _, limit := range []int{0, -1, 101} {
			res := testutil.CallRoute[listDestinations.Request, openapi.BadRequestErrorResponse](h, list, root, listDestinations.Request{Limit: &limit})
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
		}
		for _, request := range []createDestination.Request{
			{Name: "Operations", Channel: "email", Url: new("https://example.com/SECRET"), IdempotencyKey: uid.New(uid.TestPrefix)},
			{Name: "Operations", Channel: openapi.Webhook, Url: new("https://example.com/SECRET"), IdempotencyKey: uid.New(uid.TestPrefix)},
			{Name: "Operations", Channel: openapi.Webhook, Url: new("https://example.com/SECRET"), SigningSecret: new("SECRET"), IdempotencyKey: uid.New(uid.TestPrefix)},
			{Name: "Operations", Channel: openapi.Slack, Url: new("https://example.com/SECRET"), IdempotencyKey: ""},
			{Name: "Operations", Channel: openapi.Slack, Url: new("https://example.com/SECRET"), IdempotencyKey: strings.Repeat("k", 129)},
		} {
			res := testutil.CallRoute[createDestination.Request, openapi.BadRequestErrorResponse](h, create, root, request)
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			require.NotContains(t, string(res.RawBody), "SECRET")
		}
		previewResponse := testutil.CallRoute[previewRecipients.Request, openapi.BadRequestErrorResponse](h, preview, root, previewRecipients.Request{NotificationType: deployment.Definitions()[0].Type()})
		require.Equal(t, http.StatusBadRequest, previewResponse.Status, previewResponse.RawBody)
		testResponse := testutil.CallRoute[testDestination.Request, openapi.BadRequestErrorResponse](h, test, root, testDestination.Request{DestinationId: service.destinationID})
		require.Equal(t, http.StatusBadRequest, testResponse.Status, testResponse.RawBody)
		require.Empty(t, service.lists)
		require.Empty(t, service.creates)
		require.Empty(t, service.tests)
	})
}
