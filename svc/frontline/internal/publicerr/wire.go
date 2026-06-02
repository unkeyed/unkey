// This file maps the public catalog codes to their gRPC and Connect
// wire-protocol equivalents. The catalog itself stays protocol-agnostic;
// these mappings are read-only views over Problem so the wire writers
// in middleware/ don't have to know about Code strings.
//
// One table (wireCodes) drives both protocols so the gRPC int and
// Connect string can't drift apart for a given public code. Connect's
// HTTP status mapping is the standard one defined in the Connect spec
// (https://connectrpc.com/docs/protocol#error-codes), keyed off the
// Connect code string.

package publicerr

import "github.com/unkeyed/unkey/pkg/codes"

// wireCode is the gRPC and Connect representation of a public code.
// grpc is the numeric google.rpc.Code; connect is the lowercase
// Connect error code string.
type wireCode struct {
	grpc    int
	connect string
}

// gRPC status numeric constants from google.rpc.Code. Reproduced here
// so we don't pull in grpc-go just for the ints.
const (
	grpcOK                = 0
	grpcCanceled          = 1
	grpcInvalidArgument   = 3
	grpcDeadlineExceeded  = 4
	grpcNotFound          = 5
	grpcPermissionDenied  = 7
	grpcResourceExhausted = 8
	grpcInternal          = 13
	grpcUnavailable       = 14
	grpcUnauthenticated   = 16
)

// wireCodes maps each public catalog code to its gRPC/Connect pair.
// Entries must exist for every key in catalog; TestWireCodes_Complete
// fails CI on omission.
var wireCodes = map[string]wireCode{
	// ── Actionable 4xx ────────────────────────────────────────────
	"missing_credentials":      {grpcUnauthenticated, "unauthenticated"},
	"invalid_key":              {grpcUnauthenticated, "unauthenticated"},
	"insufficient_permissions": {grpcPermissionDenied, "permission_denied"},
	"rate_limited":             {grpcResourceExhausted, "resource_exhausted"},
	"firewall_denied":          {grpcPermissionDenied, "permission_denied"},
	"openapi_invalid_request":  {grpcInvalidArgument, "invalid_argument"},
	// gRPC has no "payload too large"; resource_exhausted is the
	// canonical bucket for "client exceeded a server limit".
	"request_body_too_large": {grpcResourceExhausted, "resource_exhausted"},
	"client_closed_request":  {grpcCanceled, "canceled"},

	// ── Generic 4xx classes ───────────────────────────────────────
	"bad_request":       {grpcInvalidArgument, "invalid_argument"},
	"unauthorized":      {grpcUnauthenticated, "unauthenticated"},
	"forbidden":         {grpcPermissionDenied, "permission_denied"},
	"not_found":         {grpcNotFound, "not_found"},
	"too_many_requests": {grpcResourceExhausted, "resource_exhausted"},

	// ── 5xx classes ───────────────────────────────────────────────
	"internal_server_error": {grpcInternal, "internal"},
	"bad_gateway":           {grpcUnavailable, "unavailable"},
	"service_unavailable":   {grpcUnavailable, "unavailable"},
	"gateway_timeout":       {grpcDeadlineExceeded, "deadline_exceeded"},
}

// connectHTTPStatus is the Connect spec's mapping from Connect code
// string → HTTP status for unary responses. Used only by Connect-unary;
// gRPC trailers-only responses are always HTTP 200, and Connect-stream
// is also always HTTP 200 (errors ride the end-stream envelope).
var connectHTTPStatus = map[string]codes.HTTPStatus{
	"canceled":            codes.StatusRequestTimeout,
	"unknown":             codes.StatusInternalServerError,
	"invalid_argument":    codes.StatusBadRequest,
	"deadline_exceeded":   codes.StatusRequestTimeout,
	"not_found":           codes.StatusNotFound,
	"already_exists":      codes.StatusConflict,
	"permission_denied":   codes.StatusForbidden,
	"resource_exhausted":  codes.StatusTooManyRequests,
	"failed_precondition": codes.StatusPreconditionFailed,
	"aborted":             codes.StatusConflict,
	"out_of_range":        codes.StatusBadRequest,
	"unimplemented":       codes.StatusNotFound,
	"internal":            codes.StatusInternalServerError,
	"unavailable":         codes.StatusServiceUnavailable,
	"data_loss":           codes.StatusInternalServerError,
	"unauthenticated":     codes.StatusUnauthorized,
}

// GRPCStatus returns the numeric gRPC status code for this Problem.
// Unknown public codes default to INTERNAL (13) so a missing entry
// surfaces as a generic gRPC error rather than as OK.
func (p Problem) GRPCStatus() int {
	if c, ok := wireCodes[p.Code]; ok {
		return c.grpc
	}
	return grpcInternal
}

// ConnectCode returns the Connect error code string for this Problem.
// Unknown public codes default to "internal".
func (p Problem) ConnectCode() string {
	if c, ok := wireCodes[p.Code]; ok {
		return c.connect
	}
	return "internal"
}

// ConnectHTTPStatus returns the HTTP status that a Connect-unary
// response carries for this Problem. Derived from ConnectCode via the
// Connect spec's standard mapping — not from Problem.Status, because
// Connect defines its own code→status table that differs from ours
// (e.g. canceled → 408, not 499).
func (p Problem) ConnectHTTPStatus() codes.HTTPStatus {
	if s, ok := connectHTTPStatus[p.ConnectCode()]; ok {
		return s
	}
	return codes.StatusInternalServerError
}
