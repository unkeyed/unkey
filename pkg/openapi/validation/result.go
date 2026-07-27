package validation

import (
	"sort"
	"strings"
)

// The shape of what a caller receives, and the shaping applied before they do.

// ValidationError represents a single field-level validation failure.
//
// Location is an instance path into the request, prefixed by the part of the
// request it belongs to: "body.permissions[0].name", "query.limit",
// "header.X-Api-Version", or "request" when nothing narrower applies. Message is
// a predicate about that location. Fix is non-nil when there is something
// actionable to add, which is mostly where the message had to withhold a detail
// that came from the request.
type ValidationError struct {
	Message  string
	Location string
	Fix      *string
}

// String renders the error the way it reads to a developer, joining the location
// and the message into one sentence.
func (e ValidationError) String() string {
	return e.Location + " " + e.Message
}

// Result holds the validation outcome when a request is invalid.
// A nil *Result means the request passed validation.
type Result struct {
	Detail string
	Errors []ValidationError
}

// Summary renders the whole result as one line, for callers that have a single
// message to fill rather than a structured response.
func (r *Result) Summary() string {
	if r == nil {
		return ""
	}
	if len(r.Errors) == 0 {
		return r.Detail
	}

	parts := make([]string, 0, maxSummaryErrors)
	for i, e := range r.Errors {
		if i == maxSummaryErrors {
			break
		}
		parts = append(parts, e.String())
	}
	summary := r.Detail + ": " + strings.Join(parts, "; ")
	if len(r.Errors) > maxSummaryErrors {
		summary += "; …"
	}

	return summary
}

const (
	// maxSummaryErrors bounds how many entries Summary joins into one line.
	maxSummaryErrors = 5

	// maxErrors bounds the errors list. A body can violate a constraint on every
	// one of thousands of properties, and both the response and the request log
	// row would grow with it.
	maxErrors = 25
)

func newError(location, message, fix string) ValidationError {
	e := ValidationError{Location: location, Message: message, Fix: nil}
	if fix != "" {
		e.Fix = &fix
	}

	return e
}

// normalize deduplicates, orders, and bounds the errors.
//
// Ordering matters beyond tidiness: the JSON Schema library validates an object's
// properties by ranging over a map, so without a sort the same request reports
// its problems in a different order on every call.
func normalize(errs []ValidationError) []ValidationError {
	seen := make(map[string]struct{}, len(errs))
	unique := make([]ValidationError, 0, len(errs))
	for _, e := range errs {
		key := e.Location + "\x00" + e.Message
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, e)
	}

	sort.SliceStable(unique, func(i, j int) bool {
		if unique[i].Location != unique[j].Location {
			return unique[i].Location < unique[j].Location
		}

		return unique[i].Message < unique[j].Message
	})

	if len(unique) == 0 {
		return []ValidationError{newError(locationRequest, fallbackMessage, "")}
	}

	if len(unique) > maxErrors {
		truncated := unique[:maxErrors:maxErrors]

		return append(truncated, newError(locationRequest,
			"has further validation errors that were not reported", ""))
	}

	return unique
}
