package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Audience is the default JWT audience for dashboard-minted API bearer JWTs.
const Audience = "api.unkey.com"

// Claims is the JWT payload accepted by the API auth resolver.
type Claims struct {
	tokenjwt.RegisteredClaims

	// Org scopes locally minted dashboard fallback tokens.
	Org OrganizationClaims `json:"org"`

	// WorkOSOrgID is the built-in organization claim in WorkOS access tokens.
	WorkOSOrgID string `json:"org_id"`

	// User supports providers that put user identity in a nested object.
	User UserClaims `json:"user"`

	// Name is optional display text for audit logs. Subject is used when empty.
	Name string `json:"name"`

	// Permissions is the RBAC permission set in locally minted dashboard fallback tokens.
	Permissions []string `json:"perms"`

	// WorkOSPermissions is the built-in permission claim in WorkOS access tokens.
	WorkOSPermissions []string `json:"permissions"`
}

// OrganizationClaims contains the organization identifier from a JWT.
type OrganizationClaims struct {
	ID string `json:"id"`
}

// UserClaims contains the user identifier from a JWT.
type UserClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (c Claims) organizationID() string {
	if c.Org.ID != "" {
		return c.Org.ID
	}
	return c.WorkOSOrgID
}

func (c Claims) subjectID() string {
	if c.Subject != "" {
		return c.Subject
	}
	return c.User.ID
}

func (c Claims) subjectName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.User.Email != "" {
		return c.User.Email
	}
	return c.subjectID()
}

func (c Claims) permissions() []string {
	if len(c.Permissions) > 0 {
		return c.Permissions
	}
	return c.WorkOSPermissions
}

// WorkspaceLookup resolves JWT organization IDs to workspace IDs.
type WorkspaceLookup interface {
	FindWorkspaceIDByOrgID(ctx context.Context, orgID string) (string, error)
}

// WorkspaceLookupFunc adapts a function into a WorkspaceLookup.
type WorkspaceLookupFunc func(ctx context.Context, orgID string) (string, error)

// FindWorkspaceIDByOrgID resolves an organization ID to a workspace ID.
func (f WorkspaceLookupFunc) FindWorkspaceIDByOrgID(ctx context.Context, orgID string) (string, error) {
	return f(ctx, orgID)
}

// Resolver authenticates bearer JWTs into auth principals.
type Resolver struct {
	verifiers       []tokenjwt.Verifier[Claims]
	workspaceLookup WorkspaceLookup
}

const jwksFetchTimeout = 10 * time.Second

// NewResolver creates a resolver that verifies HS256 JWTs with any configured secret.
//
// Secrets are tried in order. Callers should put the active signing secret first
// and keep recently retired secrets later in the list until every token signed
// with those secrets has expired.
func NewResolver(workspaceLookup WorkspaceLookup, issuer string, audience string, secrets ...[]byte) (*Resolver, error) {
	if len(secrets) == 0 {
		return nil, errors.New("at least one JWT secret is required")
	}

	if workspaceLookup == nil {
		return nil, errors.New("workspace lookup is required")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT issuer is required")
	}

	verifiers := make([]tokenjwt.Verifier[Claims], 0, len(secrets))
	for i, secret := range secrets {
		verifier, err := tokenjwt.NewHS256Verifier[Claims](secret, verifierOptions(issuer, audience)...)
		if err != nil {
			return nil, fmt.Errorf("create verifier for jwt secret %d: %w", i, err)
		}
		verifiers = append(verifiers, verifier)
	}

	return &Resolver{
		verifiers:       verifiers,
		workspaceLookup: workspaceLookup,
	}, nil
}

// NewResolverWithJWKSURL creates a resolver that verifies RS256 JWTs signed by
// keys from the configured JWKS endpoint.
func NewResolverWithJWKSURL(ctx context.Context, workspaceLookup WorkspaceLookup, issuer string, audience string, jwksURL string) (*Resolver, error) {
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT issuer is required")
	}

	ctx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()

	jwksVerifiers, err := fetchJWKSVerifiers(ctx, issuer, audience, jwksURL)
	if err != nil {
		return nil, err
	}

	if workspaceLookup == nil {
		return nil, errors.New("workspace lookup is required")
	}

	return &Resolver{
		verifiers:       jwksVerifiers,
		workspaceLookup: workspaceLookup,
	}, nil
}

// Resolve claims JWT-shaped bearer tokens and leaves other credentials to later resolvers.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	token, err := zen.Bearer(sess)
	if err != nil {
		return nil, nil
	}
	if strings.Count(token, ".") != 2 {
		return nil, nil
	}
	segments := strings.Split(token, ".")

	claims, err := r.verify(token)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to verify JWT"),
			fault.Public("Invalid bearer token."),
		)
	}
	orgID := claims.organizationID()
	subjectID := claims.subjectID()
	if subjectID == "" || orgID == "" || claims.ExpiresAt == 0 || claims.IssuedAt == 0 {
		return nil, fault.New("invalid JWT claims",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT subject, organization id, exp, and iat are required"),
			fault.Public("Invalid bearer token."),
		)
	}
	workspaceID, err := r.workspaceLookup.FindWorkspaceIDByOrgID(ctx, orgID)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to resolve JWT organization to workspace"),
			fault.Public("Invalid bearer token."),
		)
	}
	if workspaceID == "" {
		return nil, fault.New("invalid JWT organization",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT organization resolved to empty workspace id"),
			fault.Public("Invalid bearer token."),
		)
	}
	header, err := decodeSegment(segments[0])
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to decode JWT header"),
			fault.Public("Invalid bearer token."),
		)
	}
	payload, err := decodeSegment(segments[1])
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to decode JWT payload"),
			fault.Public("Invalid bearer token."),
		)
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   subjectID,
			Name: claims.subjectName(),
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.JWTSource{
			Header:    header,
			Payload:   payload,
			Signature: segments[2],
		},
		WorkspaceID: workspaceID,
		Permissions: claims.permissions(),
	}, nil
}

func decodeSegment(segment string) (map[string]any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

func (r *Resolver) verify(token string) (Claims, error) {
	var lastErr error
	for _, verifier := range r.verifiers {
		claims, err := verifier.Verify(token)
		if err == nil {
			return claims, nil
		}
		lastErr = err
	}

	var zero Claims
	if lastErr == nil {
		return zero, errors.New("no JWT verifiers configured")
	}
	return zero, lastErr
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Algorithm string `json:"alg"`
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
	KeyID     string `json:"kid"`
}

func fetchJWKSVerifiers(ctx context.Context, issuer string, audience string, jwksURL string) ([]tokenjwt.Verifier[Claims], error) {
	if strings.TrimSpace(jwksURL) == "" {
		return nil, errors.New("JWKS URL must not be empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch JWKS: unexpected status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("JWKS contains no keys")
	}

	verifiers := make([]tokenjwt.Verifier[Claims], 0, len(jwks.Keys))
	for _, key := range jwks.Keys {
		verifier, err := verifierFromJWK(key, issuer, audience)
		if err != nil {
			return nil, err
		}
		if verifier == nil {
			continue
		}
		verifiers = append(verifiers, verifier)
	}
	if len(verifiers) == 0 {
		return nil, errors.New("JWKS contains no usable RS256 signing keys")
	}

	return verifiers, nil
}

func verifierFromJWK(key jwkKey, issuer string, audience string) (tokenjwt.Verifier[Claims], error) {
	if key.KeyType != "RSA" {
		return nil, nil
	}
	if key.Use != "" && key.Use != "sig" {
		return nil, nil
	}
	if key.Algorithm != "" && key.Algorithm != "RS256" {
		return nil, nil
	}

	modulus, err := base64.RawURLEncoding.DecodeString(key.Modulus)
	if err != nil {
		return nil, fmt.Errorf("decode JWKS key %q modulus: %w", key.KeyID, err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.Exponent)
	if err != nil {
		return nil, fmt.Errorf("decode JWKS key %q exponent: %w", key.KeyID, err)
	}
	exponent := new(big.Int).SetBytes(exponentBytes)
	if !exponent.IsInt64() || exponent.Sign() <= 0 {
		return nil, fmt.Errorf("JWKS key %q has invalid exponent", key.KeyID)
	}

	publicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(exponent.Int64()),
	}
	if publicKey.N.Sign() <= 0 {
		return nil, fmt.Errorf("JWKS key %q has invalid modulus", key.KeyID)
	}

	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("encode JWKS key %q public key: %w", key.KeyID, err)
	}
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   der,
	}))

	return tokenjwt.NewRS256Verifier[Claims](publicKeyPEM, verifierOptions(issuer, audience)...)
}

func verifierOptions(issuer string, audience string) []tokenjwt.VerifyOption {
	options := []tokenjwt.VerifyOption{tokenjwt.WithIssuer(issuer)}
	if strings.TrimSpace(audience) != "" {
		options = append(options, tokenjwt.WithAudience(audience))
	}
	return options
}
