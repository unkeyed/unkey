// Package oauth implements the WorkOS AuthKit OAuth 2.0 Device Authorization
// Grant (RFC 8628) used by the Unkey CLI to obtain user-scoped access tokens.
//
// It is deliberately dependency-light (plain net/http, no WorkOS SDK), mirroring
// the existing WorkOS callers in svc/ctrl/internal/workos and
// tools/upsert-workos-permissions. Both cmd/auth (login/logout) and cmd/api/util
// (refresh-on-call) import it, so it lives under cmd/internal rather than being
// scoped to a single command.
package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultBaseURL is the WorkOS API host. Overridable via WithBaseURL for tests.
const defaultBaseURL = "https://api.workos.com"

const deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// Sentinel errors let callers message users precisely and drive the polling
// state machine.
var (
	// ErrAccessDenied means the user rejected the device authorization request.
	ErrAccessDenied = errors.New("device authorization was denied")
	// ErrExpiredToken means the device code expired before the user approved it.
	ErrExpiredToken = errors.New("device code expired before authorization completed")
	// ErrRevocationUnsupported means server-side revocation could not be
	// performed from this (public) client. Logout treats it as non-fatal.
	ErrRevocationUnsupported = errors.New("server-side session revocation is not available from the CLI")

	// errAuthorizationPending and errSlowDown are internal polling signals.
	errAuthorizationPending = errors.New("authorization_pending")
	errSlowDown             = errors.New("slow_down")
)

// slowDownIncrement is added to the poll interval when WorkOS returns slow_down,
// per RFC 8628 section 3.5.
const slowDownIncrement = 5 * time.Second

// DeviceAuth is the response to a device authorization request.
type DeviceAuth struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// TokenResponse is a successful authentication or refresh result.
type TokenResponse struct {
	AccessToken    string
	RefreshToken   string
	Email          string
	OrganizationID string
	// ExpiresAt is derived from the access token's exp claim.
	ExpiresAt time.Time
}

// Client talks to the WorkOS User Management API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	slowDown   time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the WorkOS API host (used in tests).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithSlowDownIncrement overrides the interval added on a slow_down response.
// Used by tests to avoid a real multi-second wait.
func WithSlowDownIncrement(d time.Duration) Option {
	return func(c *Client) { c.slowDown = d }
}

// New returns a Client with sensible defaults.
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		slowDown:   slowDownIncrement,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RequestDeviceAuthorization starts the device flow, returning the user code and
// verification URI the caller displays plus the device code used for polling.
func (c *Client) RequestDeviceAuthorization(ctx context.Context, clientID string) (DeviceAuth, error) {
	form := url.Values{"client_id": {clientID}}

	var body struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := c.postForm(ctx, "/user_management/authorize/device", form, &body); err != nil {
		return DeviceAuth{}, err
	}

	interval := body.Interval
	if interval <= 0 {
		interval = 5 // RFC 8628 default
	}

	return DeviceAuth{
		DeviceCode:              body.DeviceCode,
		UserCode:                body.UserCode,
		VerificationURI:         body.VerificationURI,
		VerificationURIComplete: body.VerificationURIComplete,
		ExpiresIn:               time.Duration(body.ExpiresIn) * time.Second,
		Interval:                time.Duration(interval) * time.Second,
	}, nil
}

// PollForToken polls the token endpoint until the user approves, the code
// expires, or ctx is cancelled. It honors authorization_pending (keep polling)
// and slow_down (increase interval) per RFC 8628.
func (c *Client) PollForToken(ctx context.Context, clientID, deviceCode string, interval, expiresIn time.Duration) (TokenResponse, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(expiresIn)

	for {
		tok, err := c.authenticate(ctx, url.Values{
			"client_id":   {clientID},
			"grant_type":  {deviceGrantType},
			"device_code": {deviceCode},
		})
		switch {
		case err == nil:
			return tok, nil
		case errors.Is(err, errAuthorizationPending):
			// keep polling
		case errors.Is(err, errSlowDown):
			interval += c.slowDown
		default:
			return TokenResponse{}, err
		}

		if expiresIn > 0 && !time.Now().Before(deadline) {
			return TokenResponse{}, ErrExpiredToken
		}

		select {
		case <-ctx.Done():
			return TokenResponse{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// Refresh exchanges a refresh token for a fresh access/refresh token pair. When
// orgID is non-empty the resulting token is scoped to that organization.
func (c *Client) Refresh(ctx context.Context, clientID, refreshToken, orgID string) (TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if orgID != "" {
		form.Set("organization_id", orgID)
	}
	return c.authenticate(ctx, form)
}

// Revoke attempts to revoke the WorkOS session server-side so a leaked refresh
// token cannot keep minting access tokens after logout.
//
// NOTE: WorkOS session revocation is an authenticated management operation that
// requires the environment's secret API key, which a public device-flow client
// does not possess. Until an Unkey API endpoint proxies revocation server-side,
// this returns ErrRevocationUnsupported. Callers (logout) treat that as
// non-fatal and fall back to clearing the local credential plus surfacing that
// the token expires naturally.
func (c *Client) Revoke(ctx context.Context, clientID, refreshToken string) error {
	return ErrRevocationUnsupported
}

// authenticate posts to the WorkOS authenticate endpoint and maps the response
// to a TokenResponse or a device-flow error.
func (c *Client) authenticate(ctx context.Context, form url.Values) (TokenResponse, error) {
	var body struct {
		AccessToken    string `json:"access_token"`
		RefreshToken   string `json:"refresh_token"`
		OrganizationID string `json:"organization_id"`
		User           struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := c.postForm(ctx, "/user_management/authenticate", form, &body); err != nil {
		return TokenResponse{}, err
	}

	expiresAt, err := accessTokenExpiry(body.AccessToken)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("could not read access token expiry: %w", err)
	}

	return TokenResponse{
		AccessToken:    body.AccessToken,
		RefreshToken:   body.RefreshToken,
		Email:          body.User.Email,
		OrganizationID: body.OrganizationID,
		ExpiresAt:      expiresAt,
	}, nil
}

// postForm sends an application/x-www-form-urlencoded POST and decodes a 2xx
// JSON body into out. Non-2xx responses are mapped to device-flow sentinel
// errors when recognized, else a descriptive error.
func (c *Client) postForm(ctx context.Context, path string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading response from %s failed: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapError(resp.StatusCode, data)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding response from %s failed: %w", path, err)
	}
	return nil
}

// mapError translates a WorkOS error body into a sentinel or descriptive error.
func mapError(status int, data []byte) error {
	var body struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(data, &body)

	switch body.Error {
	case "authorization_pending":
		return errAuthorizationPending
	case "slow_down":
		return errSlowDown
	case "access_denied":
		return ErrAccessDenied
	case "expired_token":
		return ErrExpiredToken
	}

	if body.Error != "" {
		desc := body.ErrorDescription
		if desc == "" {
			desc = body.Error
		}
		return fmt.Errorf("workos error (%s): %s", body.Error, desc)
	}
	return fmt.Errorf("workos request failed with status %d", status)
}

// accessTokenExpiry decodes (without verifying) the access token's exp claim.
// The CLI does not verify the token — svc/api does that server-side — it only
// needs the expiry to decide when to refresh.
func accessTokenExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("access token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("access token payload is not valid base64url: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("access token payload is not valid JSON: %w", err)
	}
	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("access token has no exp claim")
	}
	return time.Unix(claims.Exp, 0).UTC(), nil
}
