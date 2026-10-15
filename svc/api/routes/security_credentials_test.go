package routes_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	portalservice "github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/auth"
	authjwt "github.com/unkeyed/unkey/pkg/auth/jwt"
	rootkey "github.com/unkeyed/unkey/pkg/auth/root_key"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes"
)

func TestProductionSecurityCredentials(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	foreignWorkspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	foreignAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: foreignWorkspace.ID})
	externalID := uid.New(uid.TestPrefix)
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalID,
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  &identity.ID,
	})

	otherIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  uid.New(uid.TestPrefix),
	})
	h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  &otherIdentity.ID,
	})

	foreignIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: foreignWorkspace.ID,
		ExternalID:  externalID,
	})
	h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: foreignWorkspace.ID,
		KeySpaceID:  foreignAPI.KeyAuthID.String,
		IdentityID:  &foreignIdentity.ID,
	})

	otherAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  otherAPI.KeyAuthID.String,
		IdentityID:  &identity.ID,
	})

	root := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":projects/"+api.ProjectID+"/keyspaces/"+api.KeyAuthID.String+"#read",
		"api."+api.ID+".update_key",
	)
	foreignRoot := h.CreateRootKey(foreignWorkspace.ID, "api.*.update_key")
	portalHeaders := h.CreatePortalSession(
		workspace.ID, externalID,
		[]string{api.KeyAuthID.String}, []string{"keys:read"},
	)

	secret := []byte("production-router-test-signing-secret")
	const issuer = "https://router-security.test"
	jwtResolver, err := authjwt.NewResolver(authjwt.WorkspaceLookupFunc(func(ctx context.Context, orgID string) (string, error) {
		ws, err := db.Query.FindWorkspaceByOrgID(ctx, h.DB.RO(), orgID)
		if db.IsNotFound(err) {
			return "", authjwt.ErrWorkspaceNotFound
		}
		if err != nil {
			return "", err
		}
		if !ws.Enabled {
			return "", authjwt.ErrWorkspaceDisabled
		}
		return ws.ID, nil
	}), issuer, authjwt.Audience, secret)
	require.NoError(t, err)

	sign := func(secret []byte, orgID string) string {
		t.Helper()
		signer, err := tokenjwt.NewHS256Signer[authjwt.Claims](secret)
		require.NoError(t, err)

		now := time.Now()
		token, err := signer.Sign(authjwt.Claims{
			RegisteredClaims: tokenjwt.RegisteredClaims{
				Issuer:    issuer,
				Subject:   uid.New(uid.TestPrefix),
				Audience:  []string{authjwt.Audience},
				IssuedAt:  now.Unix(),
				NotBefore: now.Add(-time.Minute).Unix(),
				ExpiresAt: now.Add(time.Hour).Unix(),
			},
			Org:   authjwt.OrganizationClaims{ID: orgID},
			Roles: []string{"admin"},
		})
		require.NoError(t, err)
		return token
	}

	jwt := sign(secret, workspace.OrgID)
	forgedJWT := sign([]byte("different-router-test-signing-secret"), workspace.OrgID)
	foreignJWT := sign(secret, foreignWorkspace.OrgID)

	validator, err := validation.New()
	require.NoError(t, err)
	srv, err := zen.New(zen.Config{})
	require.NoError(t, err)
	requests := batch.NewNoop[schema.ApiRequest]()
	t.Cleanup(requests.Close)
	redactedPaths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)

	routes.Register(srv, &routes.Services{
		Database:         h.DB,
		Keys:             h.Keys,
		Auth:             auth.New(workos.NewRoleMappingResolver(jwtResolver), rootkey.NewResolver(h.Keys)),
		PortalAuth:       h.PortalAuth,
		Validator:        validator,
		Caches:           h.Caches,
		Clock:            h.Clock,
		ApiRequests:      requests,
		Redactor:         redaction.New(redactedPaths),
		KeyVerifications: h.KeyVerifications,
		Ratelimit:        h.Ratelimit,
		Auditlogs:        h.Auditlogs,
		UsageLimiter:     h.UsageLimiter,
		Vault:            h.Vault,
	}, zen.InstanceInfo{})

	unknownRoot := openapi.BaseError{
		Status: 401,
		Title:  "Unauthorized",
		Type:   "https://unkey.com/docs/errors/unkey/authentication/key_not_found",
		Detail: "The provided root key is invalid. We could not find the requested key.",
	}
	keyNotFound := openapi.BaseError{
		Status: 404,
		Title:  "Not Found",
		Type:   "https://unkey.com/docs/errors/unkey/data/key_not_found",
		Detail: "The specified key was not found.",
	}
	invalidJWT := openapi.BaseError{
		Status: 400,
		Title:  "Bad Request",
		Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
		Detail: "Invalid bearer token.",
	}

	unknownHeaders := testutil.Bearer("unkey_" + uid.New(uid.TestPrefix))
	mixedHeaders := portalHeaders.Clone()
	mixedHeaders.Set("Authorization", unknownHeaders.Get("Authorization"))
	update := openapi.V2KeysUpdateKeyRequestBody{
		KeyId:   key.KeyID,
		Enabled: new(false),
	}

	t.Run("missing header is rejected by production validation", func(t *testing.T) {
		res := securityRequest[openapi.V2KeysUpdateKeyRequestBody, openapi.BadRequestErrorResponse](
			t, srv, http.MethodPost, "/v2/keys.updateKey", nil, update,
		)
		require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, openapi.BadRequestErrorDetails{
			Status: 400,
			Title:  "Bad Request",
			Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			Detail: "Authorization header for 'bearer' scheme",
			Errors: []openapi.ValidationError{{
				Location: "security",
				Message:  "Authorization header was not found",
				Fix:      new("Add an 'Authorization' header to this request"),
			}},
		}, res.Body.Error)

		row, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.True(t, row.KeyEnabled)
	})

	for _, tt := range []struct {
		name    string
		headers http.Header
		want    openapi.BaseError
	}{
		{"unknown root", unknownHeaders, unknownRoot},
		{"portal cookie cannot rescue unknown bearer", mixedHeaders, unknownRoot},
		{"ordinary API key is not a root credential", testutil.Bearer(key.Key), unknownRoot},
		{"foreign root cannot mutate key", testutil.Bearer(foreignRoot), keyNotFound},
		{"forged admin JWT", testutil.Bearer(forgedJWT), invalidJWT},
		{"foreign admin JWT cannot mutate key", testutil.Bearer(foreignJWT), keyNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := securityRequest[openapi.V2KeysUpdateKeyRequestBody, openapi.ForbiddenErrorResponse](
				t, srv, http.MethodPost, "/v2/keys.updateKey", tt.headers, update,
			)
			require.Equal(t, tt.want.Status, res.Status, "%s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, tt.want, res.Body.Error)

			row, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), key.KeyID)
			require.NoError(t, err)
			require.True(t, row.KeyEnabled)
		})
	}

	t.Run("root mutation positive control", func(t *testing.T) {
		res := securityRequest[openapi.V2KeysUpdateKeyRequestBody, openapi.V2KeysUpdateKeyResponseBody](
			t, srv, http.MethodPost, "/v2/keys.updateKey", testutil.Bearer(root), update,
		)
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, openapi.EmptyResponse{}, res.Body.Data)

		row, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.False(t, row.KeyEnabled)
	})

	t.Run("root and verified JWT read positive controls", func(t *testing.T) {
		for _, token := range []string{root, jwt} {
			res := securityRequest[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
				t, srv, http.MethodPost, "/v2/apis.getApi",
				testutil.Bearer(token), openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
			)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, openapi.V2ApisGetApiResponseData{
				Id:   api.ID,
				Name: api.Name,
			}, res.Body.Data)
		}
	})

	t.Run("portal credential separation", func(t *testing.T) {
		bearerAndCookie := portalHeaders.Clone()
		bearerAndCookie.Set("Authorization", "Bearer "+root)

		for _, tt := range []struct {
			name    string
			headers http.Header
			want    openapi.BaseError
		}{
			{
				"unknown session",
				http.Header{"Cookie": {"portal_session=pat_" + uid.New(uid.TestPrefix)}},
				openapi.BaseError{
					Status: 401,
					Title:  "Unauthorized",
					Type:   "https://unkey.com/docs/errors/unkey/authentication/portal_session_not_found",
					Detail: "The portal session is invalid or has expired.",
				},
			},
			{
				"root key in cookie is not a session",
				http.Header{"Cookie": {"portal_session=" + root}},
				openapi.BaseError{
					Status: 401,
					Title:  "Unauthorized",
					Type:   "https://unkey.com/docs/errors/unkey/authentication/portal_session_not_found",
					Detail: "The portal session is invalid or has expired.",
				},
			},
			{
				"bearer cannot authenticate portal route even with valid cookie",
				bearerAndCookie,
				openapi.BaseError{
					Status: 400,
					Title:  "Bad Request",
					Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
					Detail: "You must provide a bearer credential in the Authorization header.",
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				res := securityRequest[openapi.V2PortalListKeysRequestBody, openapi.ForbiddenErrorResponse](
					t, srv, http.MethodPost, "/v2/portal.listKeys",
					tt.headers, openapi.V2PortalListKeysRequestBody{},
				)
				require.Equal(t, tt.want.Status, res.Status, "%s", res.RawBody)
				require.NotEmpty(t, res.Body.Meta.RequestId)
				require.Equal(t, tt.want, res.Body.Error)
			})
		}

		res := securityRequest[openapi.V2PortalListKeysRequestBody, openapi.V2PortalListKeysResponseBody](
			t, srv, http.MethodPost, "/v2/portal.listKeys",
			portalHeaders, openapi.V2PortalListKeysRequestBody{},
		)
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		row, err := db.Query.FindLiveKeyByID(t.Context(), h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Equal(t, []openapi.KeyResponseData{{
			KeyId:   key.KeyID,
			Start:   key.Key[:4],
			Name:    "test-key",
			Enabled: false,
			Identity: &openapi.Identity{
				Id:         identity.ID,
				ExternalId: externalID,
			},
			CreatedAt: row.KeyCreatedAtM,
			UpdatedAt: row.KeyUpdatedAtM.Int64,
		}}, res.Body.Data)
		require.Equal(t, openapi.Pagination{
			Cursor:  nil,
			HasMore: false,
		}, res.Body.Pagination)
	})

	t.Run("public liveness ignores invalid bearer", func(t *testing.T) {
		res := securityRequest[struct{}, openapi.V2LivenessResponseBody](
			t, srv, http.MethodGet, "/v2/liveness", unknownHeaders, struct{}{},
		)
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, "we're cooking", res.Body.Data.Message)
	})

	t.Run("public exchange authenticates code without bearer", func(t *testing.T) {
		code := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
		grant, err := json.Marshal(portalservice.Grant{
			KeyspaceIDs: []string{api.KeyAuthID.String},
			Scopes:      []string{"keys:read"},
		})
		require.NoError(t, err)

		now := time.Now()
		require.NoError(t, db.Query.InsertPortalSession(t.Context(), h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    uid.New(uid.PortalSessionPrefix),
			WorkspaceID:           workspace.ID,
			PortalID:              uid.New(uid.PortalPrefix),
			ExternalID:            externalID,
			Scopes:                grant,
			ExchangeCodeHash:      hash.Sha256(code),
			ExchangeCodeExpiresAt: now.Add(time.Minute).UnixMilli(),
			CreatedAt:             now.UnixMilli(),
		}))

		res := securityRequest[openapi.V2PortalExchangeCodeRequestBody, openapi.V2PortalExchangeCodeResponseBody](
			t, srv, http.MethodPost, "/v2/portal.exchangeCode",
			nil, openapi.V2PortalExchangeCodeRequestBody{Code: code},
		)
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Data.AccessToken)
		require.GreaterOrEqual(t, res.Body.Data.ExpiresAt, now.Add(24*time.Hour).UnixMilli())
		require.LessOrEqual(t, res.Body.Data.ExpiresAt, time.Now().Add(24*time.Hour).UnixMilli())

		stored, err := db.Query.FindPortalSessionByAccessTokenHash(t.Context(), h.DB.RO(), sql.NullString{
			String: hash.Sha256(res.Body.Data.AccessToken),
			Valid:  true,
		})
		require.NoError(t, err)
		require.Equal(t, workspace.ID, stored.WorkspaceID)
		require.Equal(t, externalID, stored.ExternalID)
		require.Equal(t, res.Body.Data.ExpiresAt, stored.AccessTokenExpiresAt.Int64)
		require.Equal(t, "no-store", res.Headers.Get("Cache-Control"))

		for _, rejectedCode := range []string{code, string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()} {
			denied := securityRequest[openapi.V2PortalExchangeCodeRequestBody, openapi.UnauthorizedErrorResponse](
				t, srv, http.MethodPost, "/v2/portal.exchangeCode",
				nil, openapi.V2PortalExchangeCodeRequestBody{Code: rejectedCode},
			)
			require.Equal(t, http.StatusUnauthorized, denied.Status, "%s", denied.RawBody)
			require.NotEmpty(t, denied.Body.Meta.RequestId)
			require.Equal(t, openapi.BaseError{
				Status: 401,
				Title:  "Unauthorized",
				Type:   "https://unkey.com/docs/errors/unkey/authentication/portal_session_not_found",
				Detail: "Session is invalid, expired, or has already been used.",
			}, denied.Body.Error)
		}
	})
}

func securityRequest[Req, Res any](t *testing.T, srv *zen.Server, method, path string, headers http.Header, body Req) testutil.TestResponse[Res] {
	t.Helper()
	var encoded bytes.Buffer
	require.NoError(t, json.NewEncoder(&encoded).Encode(body))

	req := httptest.NewRequest(method, path, &encoded)
	if headers != nil {
		req.Header = headers.Clone()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	recorder := httptest.NewRecorder()
	srv.Mux().ServeHTTP(recorder, req)

	var response Res
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response), "%s", recorder.Body.String())
	return testutil.TestResponse[Res]{
		Status:  recorder.Code,
		Headers: recorder.Header(),
		Body:    &response,
		RawBody: recorder.Body.String(),
	}
}
