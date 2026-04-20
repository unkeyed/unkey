package observability

import (
	"time"
)

// Run instruments a top-level workflow handler or single-shot cron job.
//
// Unlike Step, Run does NOT wrap a restate.Run journaled block — it just times
// the function call and records outcome. Suitable for:
//   - top-level handler bodies (the outermost timing of a workflow)
//   - scheduled cron-style handlers that have a single logical "run" with no
//     internal step breakdown worth instrumenting separately
//
// Note: if the worker crashes mid-handler and Restate replays from the journal,
// the replayed run records its own duration. This produces some noise for
// crash-and-restart cases but is acceptable — most workflows complete on the
// first invocation. The Restate UI is the source of truth for end-to-end
// timing of a specific invocation.
func Run(workflow string, fn func() error) error {
	start := time.Now()
	err := fn()
	recordRun(workflow, time.Since(start), err)
	return err
}

// RunTimer is the defer-friendly variant of Run for top-level handlers that
// already have a named return error. Usage:
//
//	func (w *Workflow) Deploy(...) (_ *Response, retErr error) {
//	    defer observability.RunTimer("deploy", &retErr)()
//	    ...
//	}
//
// The returned function records duration and outcome when invoked. The errPtr
// must point to the handler's named return so the deferred call observes the
// final error.
func RunTimer(workflow string, errPtr *error) func() {
	start := time.Now()
	return func() {
		var err error
		if errPtr != nil {
			err = *errPtr
		}
		recordRun(workflow, time.Since(start), err)
	}
}

func recordRun(workflow string, d time.Duration, err error) {
	outcome, category := Classify(err)
	runDuration.WithLabelValues(workflow, outcome).Observe(d.Seconds())
	runTotal.WithLabelValues(workflow, outcome, category).Inc()
}
