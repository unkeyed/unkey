// Package observability provides Prometheus instrumentation helpers for
// Restate workflow steps and top-level handlers.
//
// The helpers wrap restate.Run / restate.RunVoid so every step emits:
//   - a duration histogram
//   - an outcome counter (success/failed/cancelled) labeled with an error category
//
// Error categorization (infra / provider / app / user / cancelled) is derived
// from fault-wrapped error codes via the pkg/codes Workflow domain. Call sites
// that produce categorizable errors should wrap them with fault.Code(...) so the
// resulting metrics carry useful blame information.
package observability

import (
	"context"
	"errors"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Outcome labels for the workflow_step_total / workflow_run_total counters.
const (
	OutcomeSuccess   = "success"
	OutcomeFailed    = "failed"
	OutcomeCancelled = "cancelled"
)

// Category labels for the error_category dimension. The set is intentionally
// small to keep Prometheus cardinality bounded.
const (
	CategoryNone      = "none"      // err == nil
	CategoryCancelled = "cancelled" // context.Canceled / restate cancellation
	CategoryUser      = "user"      // user input / validation
	CategoryApp       = "app"       // user's application (Dockerfile broken, healthcheck failing)
	CategoryProvider  = "provider"  // external provider (Depot, ACME, GitHub, AWS)
	CategoryInfra     = "infra"     // our own infrastructure / control plane
)

// Classify maps an error to (outcome, category) for use as Prometheus labels.
//
// Outcome answers "did this attempt succeed". Category answers "who is to
// blame". Unclassified errors default to infra — until proven otherwise, an
// unknown failure is our problem.
func Classify(err error) (outcome string, category string) {
	if err == nil {
		return OutcomeSuccess, CategoryNone
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return OutcomeCancelled, CategoryCancelled
	}

	urn, ok := fault.GetCode(err)
	if !ok {
		return OutcomeFailed, CategoryInfra
	}

	code, parseErr := codes.ParseURN(urn)
	if parseErr != nil {
		return OutcomeFailed, CategoryInfra
	}

	// Workflow categories take precedence — they are explicitly set by call
	// sites to convey blame regardless of the underlying System.
	switch code.Category {
	case codes.CategoryWorkflowApp:
		return OutcomeFailed, CategoryApp
	case codes.CategoryWorkflowProvider:
		return OutcomeFailed, CategoryProvider
	case codes.CategoryWorkflowInfra:
		return OutcomeFailed, CategoryInfra
	}

	// Fall back to System-based mapping for codes that pre-date the workflow
	// observability layer.
	switch code.System {
	case codes.SystemUser:
		return OutcomeFailed, CategoryUser
	case codes.SystemGitHub, codes.SystemAws, codes.SystemDepot, codes.SystemAcme:
		return OutcomeFailed, CategoryProvider
	case codes.SystemUnkey, codes.SystemSentinel:
		return OutcomeFailed, CategoryInfra
	}

	return OutcomeFailed, CategoryInfra
}
