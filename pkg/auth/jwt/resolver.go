package jwt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
)

const (
	// Issuer is the expected JWT issuer for API bearer JWTs.
	Issuer = "app.unkey.com"

	// Audience is the expected JWT audience for API bearer JWTs.
	Audience = "api.unkey.com"
)

// Claims is the JWT payload accepted by the API auth resolver.
type Claims struct {
	tokenjwt.RegisteredClaims

	// WorkspaceID scopes all API work performed with this token.
	WorkspaceID string `json:"wid"`

	// Name is optional display text for audit logs. Subject is used when empty.
	Name string `json:"name"`

	// Permissions is the flat RBAC permission set granted to the token.
	Permissions []string `json:"perms"`
}

// Resolver authenticates bearer JWTs into auth principals.
type Resolver struct {
	verifier tokenjwt.Verifier[Claims]
}

// NewResolver creates a resolver that verifies HS256 JWTs with the shared secret.
func NewResolver(secret []byte) (*Resolver, error) {
	verifier, err := tokenjwt.NewHS256Verifier[Claims](
		secret,
		tokenjwt.WithIssuer(Issuer),
		tokenjwt.WithAudience(Audience),
	)
	if err != nil {
		return nil, err
	}
	return &Resolver{verifier: verifier}, nil
}

// Resolve claims JWT-shaped bearer tokens and leaves other credentials to later resolvers.
func (r *Resolver) Resolve(_ context.Context, sess *zen.Session) (*principal.Principal, error) {
	token, err := zen.Bearer(sess)
	if err != nil {
		return nil, nil
	}
	if strings.Count(token, ".") != 2 {
		return nil, nil
	}
	segments := strings.Split(token, ".")

	claims, err := r.verifier.Verify(token)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to verify JWT"),
			fault.Public("Invalid bearer token."),
		)
	}
	if claims.Subject == "" || claims.WorkspaceID == "" || claims.ExpiresAt == 0 || claims.NotBefore == 0 || claims.IssuedAt == 0 {
		return nil, fault.New("invalid JWT claims",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT subject, workspace id, exp, nbf, and iat are required"),
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

	name := claims.Name
	if name == "" {
		name = claims.Subject
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   claims.Subject,
			Name: name,
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.Source{
			Key: nil,
			JWT: &principal.JWTSource{
				Header:    header,
				Payload:   payload,
				Signature: segments[2],
			},
			PortalSession: nil,
		},
		WorkspaceID: claims.WorkspaceID,
		Permissions: claims.Permissions,
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
