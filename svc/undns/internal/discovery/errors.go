package discovery

import (
	"errors"
	"fmt"
)

// Reason names why discovery cannot identify a caller or resolve a name. Its
// values are used as metric labels and connection states, so they must stay
// stable and low-cardinality.
type Reason string

const (
	ReasonUnknownCaller        Reason = "unknown_caller"
	ReasonAmbiguousCaller      Reason = "ambiguous_caller"
	ReasonIneligibleCaller     Reason = "ineligible_caller"
	ReasonConnectionUnresolved Reason = "connection_unresolved"
	ReasonConnectionAmbiguous  Reason = "connection_ambiguous"
	ReasonConnectionInvalid    Reason = "connection_invalid"
	ReasonServiceMissing       Reason = "service_missing"
	ReasonServiceRejected      Reason = "service_rejected"
	ReasonServiceRetired       Reason = "service_retired"
	ReasonNoReadyEndpoints     Reason = "no_ready_endpoints"
	ReasonLookupError          Reason = "lookup_error"
)

// Error is a discovery failure together with its [Reason]. Err carries the
// detail, such as the failed assertion, and stays reachable with errors.Is
// and errors.As.
type Error struct {
	Reason Reason
	Err    error
}

// Error returns the detail message, or the reason if there is no detail.
func (e *Error) Error() string {
	if e.Err == nil {
		return string(e.Reason)
	}
	return e.Err.Error()
}

// Unwrap returns the detail error.
func (e *Error) Unwrap() error { return e.Err }

// ReasonOf returns the reason of the first [Error] in err's chain, or
// [ReasonLookupError] if there is none, such as for a cache index failure.
func ReasonOf(err error) Reason {
	if failure, ok := errors.AsType[*Error](err); ok {
		return failure.Reason
	}
	return ReasonLookupError
}

func fail(reason Reason, format string, args ...any) error {
	return &Error{Reason: reason, Err: fmt.Errorf(format, args...)}
}
