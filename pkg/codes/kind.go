package codes

// Kind classifies a Code by its HTTP semantics so that codes which
// share a wire-level meaning (e.g. "this resource doesn't exist",
// "the client tried to create a duplicate") can map to the same
// HTTP status without per-URN overrides.
//
// Kind is intentionally narrow: only add a new value when several
// existing codes would benefit from it AND when the HTTP mapping is
// unambiguous. For codes whose status is fully implied by their
// Category (e.g. CategoryUnauthorized → 401) leave Kind as
// KindUnknown — Category.HTTPStatus() already does the right thing.
type Kind string

const (
	// KindUnknown means the code's HTTP status falls back to its
	// Category default (or to a per-URN override). Most codes use
	// this; setting Kind is opt-in and reserved for cross-cutting
	// semantics that cut across categories.
	KindUnknown Kind = ""

	// KindNotFound covers every "X does not exist" lookup failure.
	// Maps to 404 regardless of which data subsystem produced it.
	KindNotFound Kind = "not_found"

	// KindDuplicate covers attempts to create a resource that already
	// exists (unique constraint violations surfaced to the client).
	// Maps to 409 Conflict.
	KindDuplicate Kind = "duplicate"

	// KindGone covers resources that existed but have been
	// permanently removed (soft-deleted, retired). Maps to 410.
	KindGone Kind = "gone"

	// KindInvalidInput covers requests whose input was syntactically
	// parseable but semantically wrong. Maps to 400.
	KindInvalidInput Kind = "invalid_input"

	// KindPreconditionFailed covers requests that depend on a system
	// state which is not satisfied (feature not configured, resource
	// is protected, etc.). Maps to 412.
	KindPreconditionFailed Kind = "precondition_failed"

	// KindRequestTimeout covers cases where the server gave up waiting
	// for the request to complete on its side. Maps to 408.
	KindRequestTimeout Kind = "request_timeout"

	// KindClientClosedRequest covers cases where the client
	// disconnected before we could write a response. Maps to 499
	// (nginx-style, non-stdlib).
	KindClientClosedRequest Kind = "client_closed_request"

	// KindRequestEntityTooLarge covers payloads that exceed our
	// configured maximum body size. Maps to 413.
	KindRequestEntityTooLarge Kind = "request_entity_too_large"

	// KindServiceUnavailable covers transient downstream failures
	// where retrying may succeed (lost DB connection, dependency
	// unreachable). Maps to 503.
	KindServiceUnavailable Kind = "service_unavailable"
)

// HTTPStatus returns the canonical HTTP status for this kind, or 0
// if the kind has no HTTP mapping (in which case the caller should
// fall back to Category.HTTPStatus()).
func (k Kind) HTTPStatus() HTTPStatus {
	switch k {
	case KindUnknown:
		return 0
	case KindNotFound:
		return StatusNotFound
	case KindDuplicate:
		return StatusConflict
	case KindGone:
		return StatusGone
	case KindInvalidInput:
		return StatusBadRequest
	case KindPreconditionFailed:
		return StatusPreconditionFailed
	case KindRequestTimeout:
		return StatusRequestTimeout
	case KindClientClosedRequest:
		return StatusClientClosedRequest
	case KindRequestEntityTooLarge:
		return StatusRequestEntityTooLarge
	case KindServiceUnavailable:
		return StatusServiceUnavailable
	}
	return 0
}
