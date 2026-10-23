// Package vercel authenticates Vercel OIDC federation tokens.
package vercel

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/unkeyed/unkey/pkg/auth/jwt"
	oidcdb "github.com/unkeyed/unkey/pkg/auth/oidc/vercel/db"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
)

const (
	globalIssuer      = "https://oidc.vercel.com"
	maxTokenSizeBytes = 16 * 1024
)

var (
	// ErrIntegrationNotFound lets lookup implementations distinguish a missing
	// integration from a database outage.
	ErrIntegrationNotFound = errors.New("Vercel OIDC integration not found")
	validTeamSlug          = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)
)

// Integration is the trust policy and permission set for one Vercel deployment identity.
type Integration struct {
	ID                string
	WorkspaceID       string
	Name              string
	Audience          string
	VercelTeamSlug    string
	VercelProjectName string
	VercelEnvironment string
	Enabled           bool
	WorkspaceEnabled  bool
	Permissions       []string
}

// Database provides the read replica used to load integration trust policies and grants.
type Database interface {
	RO() *db.Replica
}

type claims struct {
	tokenjwt.RegisteredClaims
	Owner       string `json:"owner"`
	OwnerID     string `json:"owner_id"`
	Project     string `json:"project"`
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
}

type Resolver struct {
	db               Database
	integrationCache cache.Cache[string, Integration]
	jwksURLForIssuer func(string) string
	verifiersMu      sync.Mutex
	verifiers        map[string]*jwt.JWKSVerifier[claims]
}

// NewResolver creates a Vercel OIDC resolver using Vercel's fixed JWKS endpoints.
func NewResolver(database Database, integrationCache cache.Cache[string, Integration]) *Resolver {
	return newResolver(database, integrationCache, func(issuer string) string {
		return issuer + "/.well-known/jwks"
	})
}

func newResolver(database Database, integrationCache cache.Cache[string, Integration], jwksURLForIssuer func(string) string) *Resolver {
	return &Resolver{
		db:               database,
		integrationCache: integrationCache,
		jwksURLForIssuer: jwksURLForIssuer,
		verifiersMu:      sync.Mutex{},
		verifiers:        map[string]*jwt.JWKSVerifier[claims]{},
	}
}

// Resolve verifies a Vercel token whose audience selects an Unkey integration.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	token, err := zen.Bearer(sess)
	if err != nil || strings.Count(token, ".") != 2 {
		return nil, nil
	}

	audience, unverifiedClaims, ok := integrationFromToken(token)
	if !ok {
		return nil, nil
	}
	if audience == "" {
		return nil, malformed("invalid Vercel OIDC audience")
	}

	integration, err := r.findIntegration(ctx, audience)
	if err != nil {
		if errors.Is(err, ErrIntegrationNotFound) {
			return nil, malformed("Vercel OIDC integration does not exist")
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load Vercel OIDC integration"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}
	if !integration.Enabled || !integration.WorkspaceEnabled {
		return nil, forbidden("Vercel OIDC integration or workspace is disabled")
	}
	if !validTeamSlug.MatchString(integration.VercelTeamSlug) {
		return nil, fault.New("invalid Vercel OIDC integration",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("stored Vercel team slug is invalid"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}

	expectedTeamIssuer := globalIssuer + "/" + integration.VercelTeamSlug
	if unverifiedClaims.Issuer != globalIssuer && unverifiedClaims.Issuer != expectedTeamIssuer {
		return nil, malformed("unexpected Vercel OIDC issuer")
	}

	verifier, err := r.verifier(unverifiedClaims.Issuer)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to configure Vercel OIDC verifier"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}
	verified, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	expectedSubject := fmt.Sprintf(
		"owner:%s:project:%s:environment:%s",
		integration.VercelTeamSlug,
		integration.VercelProjectName,
		integration.VercelEnvironment,
	)
	if len(verified.Audience) != 1 || verified.Audience[0] != integration.Audience {
		return nil, malformed("unexpected Vercel OIDC audience")
	}
	if verified.ExpiresAt == 0 || verified.NotBefore == 0 || verified.IssuedAt == 0 {
		return nil, malformed("Vercel OIDC temporal claims are required")
	}
	if verified.Owner != integration.VercelTeamSlug ||
		verified.Project != integration.VercelProjectName ||
		verified.Environment != integration.VercelEnvironment ||
		verified.Subject != expectedSubject {
		return nil, forbidden("Vercel OIDC claims do not match the integration")
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   integration.ID,
			Name: integration.Name,
			Type: principal.SubjectTypeOIDCVercel,
		},
		Type: principal.TypeJWT,
		Source: principal.OIDCVercelSource{
			IntegrationID: integration.ID,
			Owner: principal.OIDCVercelOwner{
				ID:   verified.OwnerID,
				Slug: integration.VercelTeamSlug,
			},
			Project: principal.OIDCVercelProject{
				ID:   verified.ProjectID,
				Name: integration.VercelProjectName,
			},
			Environment: integration.VercelEnvironment,
		},
		AuthorizedWorkspaceID: integration.WorkspaceID,
		Permissions:           integration.Permissions,
	}, nil
}

func (r *Resolver) findIntegration(ctx context.Context, audience string) (Integration, error) {
	integration, hit, err := r.integrationCache.SWR(ctx, audience, func(ctx context.Context) (Integration, error) {
		rows, err := db.WithRetryContext(ctx, func() ([]oidcdb.FindOIDCVercelIntegrationByAudienceRow, error) {
			return oidcdb.Query.FindOIDCVercelIntegrationByAudience(ctx, r.db.RO(), audience)
		})
		if err != nil {
			return Integration{}, err
		}
		if len(rows) == 0 {
			return Integration{}, sql.ErrNoRows
		}
		first := rows[0]
		found := Integration{
			ID:                first.ID,
			WorkspaceID:       first.WorkspaceID,
			Name:              first.Name,
			Audience:          first.Audience,
			VercelTeamSlug:    first.VercelTeamSlug,
			VercelProjectName: first.VercelProjectName,
			VercelEnvironment: first.VercelEnvironment,
			Enabled:           first.Enabled,
			WorkspaceEnabled:  first.WorkspaceEnabled,
			Permissions:       make([]string, 0, len(rows)),
		}
		for _, row := range rows {
			if row.PermissionSlug.Valid {
				found.Permissions = append(found.Permissions, row.PermissionSlug.String)
			}
		}
		return found, nil
	}, func(err error) cache.Op {
		if db.IsNotFound(err) {
			return cache.WriteNull
		}
		if err == nil {
			return cache.WriteValue
		}
		return cache.Noop
	})
	if err != nil {
		if db.IsNotFound(err) {
			return Integration{}, ErrIntegrationNotFound
		}
		return Integration{}, err
	}
	if hit == cache.Null {
		return Integration{}, ErrIntegrationNotFound
	}
	return integration, nil
}

func (r *Resolver) verifier(issuer string) (*jwt.JWKSVerifier[claims], error) {
	r.verifiersMu.Lock()
	defer r.verifiersMu.Unlock()
	if verifier, ok := r.verifiers[issuer]; ok {
		return verifier, nil
	}
	verifier, err := jwt.NewJWKSVerifier[claims](issuer, "", r.jwksURLForIssuer(issuer))
	if err != nil {
		return nil, err
	}
	r.verifiers[issuer] = verifier
	return verifier, nil
}

func integrationFromToken(token string) (string, tokenjwt.RegisteredClaims, bool) {
	var unverifiedClaims tokenjwt.RegisteredClaims
	if len(token) > maxTokenSizeBytes {
		return "", unverifiedClaims, false
	}
	segments := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil || json.Unmarshal(payload, &unverifiedClaims) != nil {
		return "", unverifiedClaims, false
	}

	if unverifiedClaims.Issuer != globalIssuer {
		teamSlug, found := strings.CutPrefix(unverifiedClaims.Issuer, globalIssuer+"/")
		if !found || !validTeamSlug.MatchString(teamSlug) {
			return "", unverifiedClaims, false
		}
	}
	if len(unverifiedClaims.Audience) != 1 || unverifiedClaims.Audience[0] == "" {
		return "", unverifiedClaims, true
	}
	return unverifiedClaims.Audience[0], unverifiedClaims, true
}

func malformed(internal string) error {
	return fault.New("invalid Vercel OIDC token",
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Internal(internal),
		fault.Public("Invalid bearer token."),
	)
}

func forbidden(internal string) error {
	return fault.New("Vercel OIDC access denied",
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Internal(internal),
		fault.Public("This Vercel deployment is not allowed to access the workspace."),
	)
}
