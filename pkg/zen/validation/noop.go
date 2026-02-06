package validation

import (
	"context"
	"net/http"
)

// NoopValidator implements OpenAPIValidator without validating or redacting anything.
// Use this for services that don't have an OpenAPI spec yet but want to participate
// in the WithValidation middleware pattern (infra header filtering, JSON compaction).
type NoopValidator struct{}

func (n *NoopValidator) Validate(_ context.Context, _ *http.Request) (ValidationErrorResponse, bool) {
	return nil, true
}

func (n *NoopValidator) SanitizeRequest(_ *http.Request, body []byte, headers http.Header) (string, []string) {
	return string(CompactJSON(body)), SanitizeHeaders(headers, nil)
}

func (n *NoopValidator) SanitizeResponse(_ *http.Request, body []byte, headers http.Header) (string, []string) {
	return string(CompactJSON(body)), SanitizeHeaders(headers, nil)
}
