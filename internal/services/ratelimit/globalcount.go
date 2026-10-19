package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/globalcounter"
)

// globalCounterNameMax is the longest counter name MySQL stores.
const globalCounterNameMax = 512

// window is the sliding-window position of a request.
type window struct {
	sequence int64
	elapsed  float64
	reset    time.Time
}

func windowOf(req RatelimitRequest) window {
	durationMs := req.Duration.Milliseconds()
	sequence := calculateSequence(req.Time, req.Duration)
	startMs := sequence * durationMs
	return window{
		sequence: sequence,
		elapsed:  float64(req.Time.UnixMilli()-startMs) / float64(durationMs),
		reset:    time.UnixMilli(startMs).Add(req.Duration),
	}
}

// globalCounterName names one window cell. Each part carries its length, so no
// choice of identifier or namespace can collide with another cell. A name longer
// than MySQL stores is replaced by its SHA-256 under a different prefix, so it
// cannot collide with a readable name either.
func globalCounterName(req RatelimitRequest, sequence int64) string {
	name := fmt.Sprintf("rl:%d:%s:%d:%s:%d:%s:%d:%d",
		len(req.WorkspaceID), req.WorkspaceID,
		len(req.Namespace), req.Namespace,
		len(req.Identifier), req.Identifier,
		req.Duration.Milliseconds(), sequence)
	if len(name) <= globalCounterNameMax {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	return "rlh:" + hex.EncodeToString(sum[:])
}

// counterEnabled reports whether the global counter decides for req. A request
// it decides never uses the regional counters, even when the counter fails.
func (s *service) counterEnabled(req RatelimitRequest) bool {
	return s.globalCounter != nil && req.UseGlobalCounter
}

// previousWindowShare is what the previous window still contributes to the
// sliding window. It is never negative, so deductions in the previous window
// cannot grant credit in this one.
func previousWindowShare(previousTotal int64, elapsed float64) int64 {
	return max(0, int64(float64(previousTotal)*(1.0-elapsed)))
}

// ratelimitCounted decides req with the global counter. It reads the previous
// window, then checks and adds to the current window in one ConditionalAdd.
func (s *service) ratelimitCounted(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	w := windowOf(req)
	previousTotal, err := s.globalCounter.Get(ctx, globalcounter.GetParams{Name: globalCounterName(req, w.sequence-1)})
	if err != nil {
		return RatelimitResponse{}, fault.Wrap(err, fault.Internal("read the previous window from the global counter"))
	}
	previousShare := previousWindowShare(previousTotal, w.elapsed)

	var effective int64
	passed := false
	_, err = s.globalCounter.ConditionalAdd(ctx, globalcounter.ConditionalAddParams{
		Name:        globalCounterName(req, w.sequence),
		DeleteAfter: time.UnixMilli((w.sequence + 2) * req.Duration.Milliseconds()),
		Delta: func(current int64) int64 {
			effective = current + previousShare + req.Cost
			if effective > req.Limit {
				return 0
			}
			passed = true
			return req.Cost
		},
	})
	if err != nil {
		return RatelimitResponse{}, fault.Wrap(err, fault.Internal("add to the global counter"))
	}

	outcome := "denied"
	remaining := int64(0)
	if passed {
		outcome = "passed"
		remaining = max(0, req.Limit-effective)
	}
	metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, "global_counter", outcome).Inc()
	return RatelimitResponse{
		Success:   passed,
		Remaining: remaining,
		Reset:     w.reset,
		Limit:     req.Limit,
		Current:   effective,
	}, nil
}

// ratelimitManyCounted decides reqs with the global counter, all or nothing.
// Every limit is checked against its own cell and, only if all of them pass, all
// costs are added together. A call that lists the same window cell twice, or
// reads a previous window that another check of the call adds to, returns an
// error, because the counter cannot add those atomically.
func (s *service) ratelimitManyCounted(ctx context.Context, reqs []RatelimitRequest) ([]RatelimitResponse, error) {
	windows := make([]window, len(reqs))
	currentNames := make([]string, len(reqs))
	previousNames := make([]string, len(reqs))
	previousShares := make([]int64, len(reqs))
	seen := make(map[string]struct{}, len(reqs))
	for i, req := range reqs {
		windows[i] = windowOf(req)
		currentNames[i] = globalCounterName(req, windows[i].sequence)
		previousNames[i] = globalCounterName(req, windows[i].sequence-1)
		if _, duplicate := seen[currentNames[i]]; duplicate {
			return nil, fault.New("ratelimit checks share a window", fault.Internal("duplicate global counter cell "+currentNames[i]))
		}
		seen[currentNames[i]] = struct{}{}
	}
	for _, previousName := range previousNames {
		if _, overlaps := seen[previousName]; overlaps {
			return nil, fault.New("ratelimit checks share a window", fault.Internal("previous window is another check's current window "+previousName))
		}
	}
	for i := range reqs {
		previousTotal, err := s.globalCounter.Get(ctx, globalcounter.GetParams{Name: previousNames[i]})
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("read the previous window from the global counter"))
		}
		previousShares[i] = previousWindowShare(previousTotal, windows[i].elapsed)
	}

	effective := make([]int64, len(reqs))
	passed := make([]bool, len(reqs))
	checks := make([]globalcounter.ConditionalAddCheck, len(reqs))
	for i, req := range reqs {
		checks[i] = globalcounter.ConditionalAddCheck{
			Name:        currentNames[i],
			DeleteAfter: time.UnixMilli((windows[i].sequence + 2) * req.Duration.Milliseconds()),
			Delta: func(current int64) (int64, bool) {
				effective[i] = current + previousShares[i] + req.Cost
				passed[i] = effective[i] <= req.Limit
				return req.Cost, !passed[i]
			},
		}
	}
	if _, err := s.globalCounter.ConditionalAddMany(ctx, globalcounter.ConditionalAddManyParams{Checks: checks}); err != nil {
		return nil, fault.Wrap(err, fault.Internal("add to the global counter"))
	}

	allPassed := true
	for _, ok := range passed {
		allPassed = allPassed && ok
	}
	responses := make([]RatelimitResponse, len(reqs))
	for i, req := range reqs {
		remaining := req.Limit - effective[i]
		if !allPassed && passed[i] {
			// Nothing was consumed, so give this entry's own cost back.
			remaining += req.Cost
		}
		outcome := "denied"
		if passed[i] {
			outcome = "passed"
		}
		metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, "global_counter", outcome).Inc()
		responses[i] = RatelimitResponse{
			Success:   passed[i],
			Remaining: max(0, remaining),
			Reset:     windows[i].reset,
			Limit:     req.Limit,
			Current:   effective[i],
		}
	}
	return responses, nil
}
