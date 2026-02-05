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

// SecurityError represents a security validation error
// It returns 400 Bad Request for format issues (missing/malformed headers)
type SecurityError struct {
	openapi.BadRequestErrorResponse
}

func (e *SecurityError) GetStatus() int {
	return e.Error.Status
}

func (e *SecurityError) SetRequestID(requestID string) {
	e.Meta.RequestId = requestID
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
	// Check for bearer auth schemes and provide detailed errors
	for _, req := range requirements {
		for schemeName := range req.Schemes {
			scheme, exists := schemes[schemeName]
			if !exists {
				continue
			}
			if scheme.Type == SecurityTypeHTTP && strings.ToLower(scheme.Scheme) == "bearer" {
				return buildBearerAuthError(r, requestID)
			}
			if scheme.Type == SecurityTypeHTTP && strings.ToLower(scheme.Scheme) == "basic" {
				return buildBasicAuthError(r, requestID)
			}
			if scheme.Type == SecurityTypeAPIKey {
				return buildAPIKeyError(scheme, requestID)
			}
		}
	}

	// Generic auth error for other scheme types
	return &SecurityError{
		BadRequestErrorResponse: openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Authentication is required",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				Errors: []openapi.ValidationError{},
				Schema: nil,
			},
		},
	}
}

// buildBearerAuthError creates a detailed error for bearer auth format failures
// Returns 400 Bad Request for missing/malformed Authorization header
func buildBearerAuthError(r *http.Request, requestID string) *SecurityError {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Authorization header is required but was not provided",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	const bearerPrefix = "bearer "
	if len(authHeader) < len(bearerPrefix) || !strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Authorization header must use Bearer scheme",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Bearer token is empty",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	// Fallback - shouldn't reach here if validateHTTPScheme works correctly
	return &SecurityError{
		BadRequestErrorResponse: openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Bearer authentication failed",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				Errors: []openapi.ValidationError{},
				Schema: nil,
			},
		},
	}
}

// buildBasicAuthError creates a detailed error for basic auth format failures
// Returns 400 Bad Request for missing/malformed Authorization header
func buildBasicAuthError(r *http.Request, requestID string) *SecurityError {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Authorization header is required but was not provided",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	const basicPrefix = "basic "
	if len(authHeader) < len(basicPrefix) || !strings.EqualFold(authHeader[:len(basicPrefix)], basicPrefix) {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Authorization header must use Basic scheme",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	credentials := strings.TrimSpace(authHeader[len(basicPrefix):])
	if credentials == "" {
		return &SecurityError{
			BadRequestErrorResponse: openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{
					RequestId: requestID,
				},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Detail: "Basic credentials are empty",
					Status: http.StatusBadRequest,
					Type:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
					Errors: []openapi.ValidationError{},
					Schema: nil,
				},
			},
		}
	}

	return &SecurityError{
		BadRequestErrorResponse: openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "Basic authentication failed",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				Errors: []openapi.ValidationError{},
				Schema: nil,
			},
		},
	}
}

// buildAPIKeyError creates a detailed error for API key auth format failures
// Returns 400 Bad Request for missing API key
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

	return &SecurityError{
		BadRequestErrorResponse: openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "API key is required in " + location + " '" + scheme.Name + "'",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/authentication/missing",
				Errors: []openapi.ValidationError{},
				Schema: nil,
			},
		},
	}
}
