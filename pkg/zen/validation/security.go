package validation

import (
	"net/http"
	"sort"
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

// SecurityError represents a security validation error
// It returns 401 Unauthorized for missing/malformed auth
type SecurityError struct {
	openapi.UnauthorizedErrorResponse
}

func (e *SecurityError) GetStatus() int {
	return e.Error.Status
}

func (e *SecurityError) SetRequestID(requestID string) {
	e.Meta.RequestId = requestID
}

func newSecurityError(requestID, detail, errType string) *SecurityError {
	return &SecurityError{
		UnauthorizedErrorResponse: openapi.UnauthorizedErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BaseError{
				Title:  "Unauthorized",
				Detail: detail,
				Status: http.StatusUnauthorized,
				Type:   errType,
			},
		},
	}
}

// ValidateSecurity validates the security requirements for a request
// Returns nil if security is satisfied, or an error response if not
// Returns 400 Bad Request for missing/malformed auth headers (format issues)
func ValidateSecurity(r *http.Request, requirements []SecurityRequirement, schemes map[string]SecurityScheme, requestID string) *SecurityError {
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
	// Return a detailed error based on what kind of auth is expected and what's wrong
	return buildSecurityError(r, requirements, schemes, requestID)
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
// It inspects the request to provide detailed error messages about what's wrong
// Returns 400 Bad Request for format issues (missing/malformed auth headers)
func buildSecurityError(r *http.Request, requirements []SecurityRequirement, schemes map[string]SecurityScheme, requestID string) *SecurityError {
	for _, req := range requirements {
		schemeNames := make([]string, 0, len(req.Schemes))
		for name := range req.Schemes {
			schemeNames = append(schemeNames, name)
		}
		sort.Strings(schemeNames)

		for _, schemeName := range schemeNames {
			scheme, exists := schemes[schemeName]
			if !exists {
				continue
			}
			if scheme.Type == SecurityTypeHTTP {
				return buildHTTPAuthError(r, requestID, scheme.Scheme)
			}
			if scheme.Type == SecurityTypeAPIKey {
				return buildAPIKeyError(scheme, requestID)
			}
		}
	}

	// Generic auth error for other scheme types
	return newSecurityError(requestID,
		"Authentication is required",
		"https://unkey.com/docs/errors/unkey/authentication/missing",
	)
}

// buildHTTPAuthError creates a detailed error for HTTP auth scheme format failures (Bearer, Basic, etc.)
// Returns 400 Bad Request for missing/malformed Authorization header
func buildHTTPAuthError(r *http.Request, requestID string, scheme string) *SecurityError {
	// Title-case the scheme for user-facing messages (e.g. "bearer" -> "Bearer")
	displayScheme := strings.ToUpper(scheme[:1]) + strings.ToLower(scheme[1:])

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return newSecurityError(requestID,
			"Authorization header is required but was not provided",
			"https://unkey.com/docs/errors/unkey/authentication/missing",
		)
	}

	prefix := strings.ToLower(scheme) + " "
	if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return newSecurityError(requestID,
			"Authorization header must use "+displayScheme+" scheme",
			"https://unkey.com/docs/errors/unkey/authentication/malformed",
		)
	}

	token := strings.TrimSpace(authHeader[len(prefix):])
	if token == "" {
		return newSecurityError(requestID,
			displayScheme+" credentials are empty",
			"https://unkey.com/docs/errors/unkey/authentication/malformed",
		)
	}

	// Unreachable if validateHTTPScheme is consistent, but satisfies the compiler
	return newSecurityError(requestID,
		displayScheme+" authentication failed",
		"https://unkey.com/docs/errors/unkey/authentication/missing",
	)
}

// buildAPIKeyError creates a detailed error for API key auth format failures
func buildAPIKeyError(scheme SecurityScheme, requestID string) *SecurityError {
	var location string
	switch scheme.In {
	case LocationHeader:
		location = "header"
	case LocationQuery:
		location = "query parameter"
	case LocationCookie:
		location = "cookie"
	case LocationPath:
		location = "path"
	default:
		location = "request"
	}

	return newSecurityError(requestID,
		"API key is required in "+location+" '"+scheme.Name+"'",
		"https://unkey.com/docs/errors/unkey/authentication/missing",
	)
}
