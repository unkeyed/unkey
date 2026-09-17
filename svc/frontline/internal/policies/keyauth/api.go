package keyauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

// APIExecutor inherits the public API's authorization and IP-allowlist checks.
// For IP-restricted keys, the API sees the gateway's outbound IP.
type APIExecutor struct {
	endpoint string
	rootKey  string
	client   *http.Client
	clock    clock.Clock
}

type APIConfig struct {
	BaseURL string
	RootKey string
	Clock   clock.Clock
}

func NewAPI(cfg APIConfig) (*APIExecutor, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Hostname() == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fault.New("verification API requires an HTTP(S) base URL without credentials, query, or fragment")
	}
	if strings.TrimSpace(cfg.RootKey) == "" || strings.ContainsAny(cfg.RootKey, "\r\n") {
		return nil, fault.New("verification API requires a valid root key")
	}
	if err := assert.NotNilAndNotZero(cfg.Clock, "clock is required"); err != nil {
		return nil, err
	}
	return &APIExecutor{
		endpoint: base.JoinPath("v2/keys.verifyKey").String(),
		rootKey:  cfg.RootKey,
		clock:    cfg.Clock,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// A redirect can leak the root key or repeat quota consumption.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (e *APIExecutor) Execute(ctx context.Context, sess *zen.Session, req *http.Request, _ string, cfg *frontlinev1.KeyAuth) (*principal.Principal, error) {
	rawKey := extractKey(req, cfg.GetLocations())
	if rawKey == "" {
		return nil, fault.New("missing API key",
			fault.Code(codes.Frontline.Auth.MissingCredentials.URN()),
			fault.Public("Authentication required. Please provide a valid API key."))
	}
	body, err := newAPIVerificationRequest(rawKey, cfg)
	if err != nil {
		return nil, err
	}
	data, err := e.verify(ctx, body)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Public("An internal error occurred during authentication."),
		)
	}
	var mostRestrictive *ratelimit.RatelimitResponse
	for _, limit := range data.Ratelimits {
		result := &ratelimit.RatelimitResponse{
			Limit: limit.Limit, Remaining: limit.Remaining, Reset: time.UnixMilli(limit.Reset),
			Success: !limit.Exceeded, Current: 0,
		}
		if mostRestrictive == nil || moreRestrictive(result, mostRestrictive) {
			mostRestrictive = result
		}
	}
	writeRateLimitResponseHeaders(sess.ResponseWriter(), mostRestrictive, e.clock)
	switch data.Code {
	case openapi.VALID:
		return keyPrincipalFromAPI(data), nil
	case openapi.INSUFFICIENTPERMISSIONS:
		return nil, fault.New("insufficient permissions",
			fault.Code(codes.Frontline.Auth.InsufficientPermissions.URN()),
			fault.Public("Access denied. The API key does not have the required permissions."))
	case openapi.RATELIMITED:
		return nil, fault.New("rate limited",
			fault.Code(codes.Frontline.Auth.RateLimited.URN()),
			fault.Public("Rate limit exceeded. Please try again later."))
	case openapi.USAGEEXCEEDED:
		return nil, fault.New("usage exceeded",
			fault.Code(codes.Frontline.Auth.UsageExceeded.URN()),
			fault.Public("Usage limit exceeded. This API key has no remaining credits."))
	case openapi.NOTFOUND, openapi.DISABLED, openapi.EXPIRED, openapi.FORBIDDEN:
		return nil, fault.New("key verification failed",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Public("Authentication failed."))
	default:
		return nil, fault.New("unknown verification code",
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Public("An internal error occurred during authentication."))
	}
}

func newAPIVerificationRequest(rawKey string, cfg *frontlinev1.KeyAuth) (openapi.V2KeysVerifyKeyRequestBody, error) {
	body := openapi.V2KeysVerifyKeyRequestBody{
		Key:         rawKey,
		Keyspaces:   ptr.P(append([]string{}, cfg.GetKeySpaceIds()...)),
		Credits:     &openapi.KeysVerifyKeyCredits{Cost: ptr.SafeDeref(cfg.Credits, 1)},
		Permissions: nil,
		Ratelimits:  nil,
		MigrationId: nil,
		Tags:        nil,
	}
	if body.Credits.Cost < 0 {
		return body, fault.New("negative credits cost in keyauth policy",
			fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
			fault.Public("Service configuration error."))
	}
	if query := cfg.GetPermissionQuery(); query != "" {
		if _, err := rbac.ParseQuery(query); err != nil {
			return body, fault.Wrap(err,
				fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
				fault.Public("Service configuration error."))
		}
		body.Permissions = ptr.P(query)
	}
	if len(cfg.GetRatelimits()) > 0 {
		body.Ratelimits = ptr.P(toVerifyRatelimits(cfg.GetRatelimits()))
	}
	return body, nil
}

func (e *APIExecutor) verify(ctx context.Context, body openapi.V2KeysVerifyKeyRequestBody) (openapi.V2KeysVerifyKeyResponseData, error) {
	var result openapi.V2KeysVerifyKeyResponseBody
	encoded, err := json.Marshal(body)
	if err != nil {
		return result.Data, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return result.Data, err
	}
	req.Header.Set("Authorization", "Bearer "+e.rootKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := e.client.Do(req)
	if err != nil {
		return result.Data, err
	}
	const responseSizeMax = 1 << 20
	response, err := io.ReadAll(io.LimitReader(res.Body, responseSizeMax+1))
	closeErr := res.Body.Close()
	if err != nil {
		return result.Data, err
	}
	if closeErr != nil {
		return result.Data, closeErr
	}
	if res.StatusCode != http.StatusOK {
		return result.Data, fmt.Errorf("verification API returned HTTP %d", res.StatusCode)
	}
	if len(response) > responseSizeMax {
		return result.Data, fmt.Errorf("verification API response exceeds %d bytes", responseSizeMax)
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return result.Data, err
	}
	data := result.Data
	if data.Valid != (data.Code == openapi.VALID) {
		return data, fault.New("inconsistent verification result")
	}
	if data.Valid && (data.KeyId == "" || data.KeyspaceId == "" || !slices.Contains(ptr.SafeDeref(body.Keyspaces), data.KeyspaceId)) {
		return data, fault.New("verification returned a missing key ID or an unexpected keyspace")
	}
	if data.Valid && data.Identity != nil && data.Identity.ExternalId == "" {
		return data, fault.New("verification returned an identity without a subject")
	}
	return data, nil
}

func keyPrincipalFromAPI(data openapi.V2KeysVerifyKeyResponseData) *principal.Principal {
	key := &principal.KeySource{
		KeyID: data.KeyId, KeySpaceID: data.KeyspaceId, Credits: data.Credits,
		Meta: data.Meta, Roles: data.Roles, Permissions: data.Permissions,
		Name: nil, ExpiresAt: nil,
	}
	if key.Meta == nil {
		key.Meta = map[string]any{}
	}
	if data.Name != "" {
		key.Name = ptr.P(data.Name)
	}
	if data.Expires != 0 {
		key.ExpiresAt = ptr.P(data.Expires)
	}
	p := &principal.Principal{
		Version: principal.PrincipalVersion, Type: principal.PrincipalTypeAPIKey,
		Subject: data.KeyId, Identity: nil,
		Source: principal.Source{Key: key, JWT: nil},
	}
	if data.Identity != nil {
		p.Subject = data.Identity.ExternalId
		p.Identity = &principal.Identity{ExternalID: data.Identity.ExternalId, Meta: data.Identity.Meta}
		if p.Identity.Meta == nil {
			p.Identity.Meta = map[string]any{}
		}
	}
	return p
}
