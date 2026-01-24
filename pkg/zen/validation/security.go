package validation

import (
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/svc/api/openapi"
)

// SecuritySchemeType represents the type of security scheme
type SecuritySchemeType string

const (
	SecurityTypeHTTP          SecuritySchemeType = "http"
	SecurityTypeAPIKey        SecuritySchemeType = "apiKey"
	SecurityTypeOAuth2        SecuritySchemeType = "oauth2"
	SecurityTypeOpenIDConnect SecuritySchemeType = "openIdConnect"
)

// SecurityScheme represents an OpenAPI security scheme definition
type SecurityScheme struct {
	Type   SecuritySchemeType
	Scheme string            // "bearer", "basic" for HTTP
	Name   string            // header/query/cookie name for apiKey
	In     ParameterLocation // where apiKey is located
}

// SecurityRequirement represents a security requirement for an operation
// Each key is a scheme name, and the value is a list of required scopes
type SecurityRequirement struct {
	Schemes map[string][]string
}

// ValidateSecurity validates the security requirements for a request
// Returns nil if security is satisfied, or an error response if not
func ValidateSecurity(r *http.Request, requirements []SecurityRequirement, schemes map[string]SecurityScheme, requestID string) *UnauthorizedError {
	// If no security requirements, the operation is public
	if len(requirements) == 0 {
		return nil
	}

	// OR logic: any requirement can satisfy the security
	for _, req := range requirements {
		if satisfiesRequirement(r, req, schemes) {
			return nil
		}
	}

	// None of the requirements were satisfied
	// Return an error based on what kind of auth is expected
	return buildSecurityError(requirements, schemes, requestID)
}

// satisfiesRequirement checks if a request satisfies a security requirement
// AND logic: all schemes in the requirement must be satisfied
func satisfiesRequirement(r *http.Request, req SecurityRequirement, schemes map[string]SecurityScheme) bool {
	for schemeName := range req.Schemes {
		scheme, exists := schemes[schemeName]
		if !exists {
			// Unknown scheme, can't validate
			return false
		}
		if !validateScheme(r, scheme) {
			return false
		}
	}
	return true
}

// validateScheme checks if a request satisfies a single security scheme
func validateScheme(r *http.Request, scheme SecurityScheme) bool {
	switch scheme.Type {
	case SecurityTypeHTTP:
		return validateHTTPScheme(r, scheme)
	case SecurityTypeAPIKey:
		return validateAPIKeyScheme(r, scheme)
	case SecurityTypeOAuth2, SecurityTypeOpenIDConnect:
		// For OAuth2 and OpenIDConnect, we only do presence validation
		// The actual token validation should be done by the authorization middleware
		return validateOAuth2Scheme(r)
	default:
		return false
	}
}

// validateOAuth2Scheme validates OAuth2/OpenIDConnect authentication
// This only checks for presence of the Authorization header
// Actual token validation is expected to be done by separate middleware
func validateOAuth2Scheme(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	return authHeader != ""
}

// validateHTTPScheme validates HTTP authentication schemes (bearer, basic)
func validateHTTPScheme(r *http.Request, scheme SecurityScheme) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	switch strings.ToLower(scheme.Scheme) {
	case "bearer":
		prefix := "bearer "
		if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
			return false
		}
		token := strings.TrimSpace(authHeader[len(prefix):])
		return token != ""

	case "basic":
		prefix := "basic "
		if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
			return false
		}
		credentials := strings.TrimSpace(authHeader[len(prefix):])
		return credentials != ""

	default:
		return false
	}
}

// validateAPIKeyScheme validates API key authentication
func validateAPIKeyScheme(r *http.Request, scheme SecurityScheme) bool {
	switch scheme.In {
	case LocationHeader:
		return r.Header.Get(scheme.Name) != ""
	case LocationQuery:
		return r.URL.Query().Get(scheme.Name) != ""
	case LocationCookie:
		cookie, err := r.Cookie(scheme.Name)
		return err == nil && cookie.Value != ""
	case LocationPath:
		// API keys in path are not supported - path parameters are validated elsewhere
		return false
	default:
		return false
	}
}

// buildSecurityError creates an appropriate error response based on the expected security
func buildSecurityError(requirements []SecurityRequirement, schemes map[string]SecurityScheme, requestID string) *UnauthorizedError {
	// Find the first HTTP bearer scheme to provide a helpful error message
	for _, req := range requirements {
		for schemeName := range req.Schemes {
			scheme, exists := schemes[schemeName]
			if !exists {
				continue
			}
			if scheme.Type == SecurityTypeHTTP && strings.ToLower(scheme.Scheme) == "bearer" {
				return &UnauthorizedError{
					UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
						Meta: openapi.Meta{
							RequestId: requestID,
						},
						Error: openapi.BaseError{
							Title:  "Unauthorized",
							Detail: "Authorization header is required but was not provided",
							Status: http.StatusUnauthorized,
							Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
						},
					},
				}
			}
		}
	}

	// Generic auth error for other scheme types
	return &UnauthorizedError{
		UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BaseError{
				Title:  "Unauthorized",
				Detail: "Authentication is required",
				Status: http.StatusUnauthorized,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
			},
		},
	}
}

// ValidateBearerAuth validates that a request has a valid Bearer token in the Authorization header
// This is used for detailed error messages when we know bearer auth is required
func ValidateBearerAuth(r *http.Request, requestID string) *UnauthorizedError {
	authHeader := r.Header.Get("Authorization")

	// Check if Authorization header is present
	if authHeader == "" {
		return &UnauthorizedError{
			UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BaseError{
					Title:  "Unauthorized",
					Detail: "Authorization header is required but was not provided",
					Status: http.StatusUnauthorized,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				},
			},
		}
	}

	// Check for Bearer scheme (case-insensitive)
	const bearerPrefix = "bearer "
	if len(authHeader) < len(bearerPrefix) || !strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
		return &UnauthorizedError{
			UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BaseError{
					Title:  "Unauthorized",
					Detail: "Authorization header must use Bearer scheme",
					Status: http.StatusUnauthorized,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
				},
			},
		}
	}

	// Check that there's actually a token after "Bearer "
	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		return &UnauthorizedError{
			UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BaseError{
					Title:  "Unauthorized",
					Detail: "Bearer token is empty",
					Status: http.StatusUnauthorized,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
				},
			},
		}
	}

	return nil
}
