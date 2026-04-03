package sessionauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/jwks"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

// JWKSConfig configures the JWKS-based session auth service.
type JWKSConfig struct {
	// KeySet is the JWKS key set used to resolve signing keys.
	KeySet jwks.KeySet

	// Issuer is the expected JWT issuer (e.g. "https://api.workos.com").
	// If empty, issuer validation is skipped.
	Issuer string

	// DB is the database used to resolve org_id to workspace_id.
	DB db.Database
}

type jwksService struct {
	verifier *jwks.Verifier[SessionClaims]
	db       db.Database
}

// NewJWKS creates a session auth service that validates JWTs using a JWKS endpoint
// and resolves the org_id claim to a workspace ID via the database.
func NewJWKS(cfg JWKSConfig) Service {
	opts := []jwks.VerifierOption{}
	if cfg.Issuer != "" {
		opts = append(opts, jwks.WithIssuer(cfg.Issuer))
	}

	return &jwksService{
		verifier: jwks.NewVerifier[SessionClaims](cfg.KeySet, opts...),
		db:       cfg.DB,
	}
}

// CanHandle returns true if the token looks like a JWT (three dot-separated
// base64url parts with a JSON header containing an "alg" field).
func (s *jwksService) CanHandle(token string) bool {
	parts := strings.SplitN(token, ".", 4)
	if len(parts) != 3 {
		return false
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}
	return header.Alg != ""
}

func (s *jwksService) Authenticate(ctx context.Context, token string) (*SessionResult, error) {
	ctx, span := tracing.Start(ctx, "sessionauth.JWKS.Authenticate")
	defer span.End()

	claims, err := s.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	if claims.OrgID == "" {
		return nil, fmt.Errorf("token missing org_id claim")
	}

	workspace, err := db.Query.FindWorkspaceByOrgID(ctx, s.db.RO(), claims.OrgID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("no workspace found for org %q", claims.OrgID)
		}
		return nil, fmt.Errorf("looking up workspace: %w", err)
	}

	return &SessionResult{
		WorkspaceID: workspace.ID,
		UserID:      claims.Subject,
		OrgID:       claims.OrgID,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	}, nil
}
