package engine

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

var sentinelEngineUnavailableTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "sentinel_engine_unavailable_total",
		Help: "Total number of requests rejected because the middleware engine is unavailable.",
	},
)

// ResilientEvaluator wraps an Evaluator and fails closed (503) when the inner
// engine is not yet available. Use SetEngine to atomically swap in a working
// evaluator once it becomes ready.
type ResilientEvaluator struct {
	inner atomic.Pointer[Evaluator]
}

var _ Evaluator = (*ResilientEvaluator)(nil)

// NewResilientEvaluator creates a ResilientEvaluator. If eng is non-nil it is
// set as the initial inner evaluator; otherwise Evaluate will return 503 until
// SetEngine is called.
func NewResilientEvaluator(eng Evaluator) *ResilientEvaluator {
	r := &ResilientEvaluator{}
	if eng != nil {
		r.inner.Store(&eng)
	}
	return r
}

// SetEngine atomically swaps in a working evaluator.
func (r *ResilientEvaluator) SetEngine(eng Evaluator) {
	r.inner.Store(&eng)
}

// Ready reports whether the inner engine is available.
func (r *ResilientEvaluator) Ready() bool {
	return r.inner.Load() != nil
}

// Evaluate delegates to the inner evaluator if available, otherwise returns a
// 503 fault to fail closed.
func (r *ResilientEvaluator) Evaluate(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	policies []*sentinelv1.Policy,
) (Result, error) {
	ptr := r.inner.Load()
	if ptr == nil {
		sentinelEngineUnavailableTotal.Inc()
		return Result{}, fault.New("middleware engine unavailable",
			fault.Code(codes.Sentinel.Internal.EngineUnavailable.URN()),
			fault.Internal("redis-backed middleware engine has not connected yet"),
			fault.Public("The middleware engine is temporarily unavailable. Please retry."),
		)
	}
	return (*ptr).Evaluate(ctx, sess, req, policies)
}
