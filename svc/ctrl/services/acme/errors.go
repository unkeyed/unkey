package acme

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/acme"
)

// ACMEErrorType represents the type of ACME error.
type ACMEErrorType string

const (
	// ACMEErrorRateLimited indicates the request was rate limited by Let's Encrypt.
	ACMEErrorRateLimited ACMEErrorType = "rate_limited"
	// ACMEErrorUnauthorized indicates an authorization/authentication failure.
	ACMEErrorUnauthorized ACMEErrorType = "unauthorized"
	// ACMEErrorBadCredentials indicates AWS/DNS provider credential issues.
	ACMEErrorBadCredentials ACMEErrorType = "bad_credentials"
	// ACMEErrorDNSPropagation indicates DNS propagation timeout.
	ACMEErrorDNSPropagation ACMEErrorType = "dns_propagation"
	// ACMEErrorUnknown indicates an unknown error type.
	ACMEErrorUnknown ACMEErrorType = "unknown"
)

// ParsedACMEError contains parsed information from an ACME error.
type ParsedACMEError struct {
	// Type is the categorized error type.
	Type ACMEErrorType
	// Message is a human-readable error message.
	Message string
	// RetryAfter is when the request can be retried (for rate limits).
	RetryAfter time.Time
	// IsRetryable indicates if the error is transient and can be retried.
	IsRetryable bool
	// OriginalError is the underlying error.
	OriginalError error
}

func (e *ParsedACMEError) Error() string {
	return e.Message
}

// ParseACMEError analyzes an error from ACME operations and returns structured information.
func ParseACMEError(err error) *ParsedACMEError {
	if err == nil {
		return nil
	}

	parsed := &ParsedACMEError{
		Type:          ACMEErrorUnknown,
		Message:       err.Error(),
		IsRetryable:   true,
		OriginalError: err,
	}

	// Check for rate limit error (lego wraps this nicely)
	var rateLimitErr *acme.RateLimitedError
	if errors.As(err, &rateLimitErr) {
		parsed.Type = ACMEErrorRateLimited
		parsed.IsRetryable = false
		parsed.Message = fmt.Sprintf("Let's Encrypt rate limit exceeded: %s", rateLimitErr.Detail)

		if rateLimitErr.RetryAfter != "" {
			if t, parseErr := time.Parse(time.RFC1123, rateLimitErr.RetryAfter); parseErr == nil {
				parsed.RetryAfter = t
				parsed.Message = fmt.Sprintf("Let's Encrypt rate limit exceeded. Retry after %s: %s",
					t.Format(time.RFC3339), rateLimitErr.Detail)
			}
		}
		return parsed
	}

	// Check for general ACME problem details
	var problemErr *acme.ProblemDetails
	if errors.As(err, &problemErr) {
		parsed.Message = fmt.Sprintf("ACME error [%s]: %s", problemErr.Type, problemErr.Detail)

		// Check problem type
		if strings.Contains(problemErr.Type, "rateLimited") {
			parsed.Type = ACMEErrorRateLimited
			parsed.IsRetryable = false
		} else if strings.Contains(problemErr.Type, "unauthorized") {
			parsed.Type = ACMEErrorUnauthorized
			parsed.IsRetryable = false
		}
		return parsed
	}

	// Check error message for common patterns
	errStr := strings.ToLower(err.Error())

	// AWS credential errors
	if strings.Contains(errStr, "signaturadoesnotmatch") ||
		strings.Contains(errStr, "signature") && strings.Contains(errStr, "does not match") ||
		strings.Contains(errStr, "invalidclienttokenid") ||
		strings.Contains(errStr, "invalid security token") ||
		strings.Contains(errStr, "the security token included in the request is invalid") {
		parsed.Type = ACMEErrorBadCredentials
		parsed.IsRetryable = false
		parsed.Message = "AWS credential error: The provided access key or secret key is invalid or malformed. " +
			"Check for trailing whitespace/newlines in the credentials."
		return parsed
	}

	// Rate limit patterns in error string (backup detection)
	if strings.Contains(errStr, "rate") && strings.Contains(errStr, "limit") ||
		strings.Contains(errStr, "too many") ||
		strings.Contains(errStr, "ratelimited") {
		parsed.Type = ACMEErrorRateLimited
		parsed.IsRetryable = false

		// Try to extract retry-after from the error message
		if idx := strings.Index(errStr, "retry after"); idx != -1 {
			timeStr := errStr[idx+12:]
			if endIdx := strings.Index(timeStr, ":"); endIdx != -1 {
				timeStr = strings.TrimSpace(timeStr[:endIdx])
			}
			// Try parsing various time formats
			for _, layout := range []string{
				"2006-01-02 15:04:05 MST",
				time.RFC3339,
				time.RFC1123,
			} {
				if t, parseErr := time.Parse(layout, timeStr); parseErr == nil {
					parsed.RetryAfter = t
					break
				}
			}
		}
		return parsed
	}

	// DNS propagation errors
	if strings.Contains(errStr, "dns") && (strings.Contains(errStr, "propagat") || strings.Contains(errStr, "timeout")) {
		parsed.Type = ACMEErrorDNSPropagation
		parsed.IsRetryable = true
		parsed.Message = "DNS propagation timeout: The DNS challenge record did not propagate in time. This may be transient."
		return parsed
	}

	// Authorization errors
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "accessdenied") || strings.Contains(errStr, "access denied") {
		parsed.Type = ACMEErrorUnauthorized
		parsed.IsRetryable = false
		parsed.Message = fmt.Sprintf("Authorization failed: %s. Check IAM permissions for Route53/DNS access.", err.Error())
		return parsed
	}

	return parsed
}

// ShouldRetry returns true if the error is transient and the operation should be retried.
func ShouldRetry(err error) bool {
	parsed := ParseACMEError(err)
	if parsed == nil {
		return false
	}
	return parsed.IsRetryable
}

// IsRateLimited returns true if the error is a rate limit error.
func IsRateLimited(err error) bool {
	parsed := ParseACMEError(err)
	if parsed == nil {
		return false
	}
	return parsed.Type == ACMEErrorRateLimited
}

// IsCredentialError returns true if the error is related to bad credentials.
func IsCredentialError(err error) bool {
	parsed := ParseACMEError(err)
	if parsed == nil {
		return false
	}
	return parsed.Type == ACMEErrorBadCredentials
}
