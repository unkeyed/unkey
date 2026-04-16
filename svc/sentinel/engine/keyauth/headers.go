package keyauth

import (
	"math"
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/clock"
)

// writeRateLimitHeaders sets standard rate limit headers on the response.
// When multiple rate limits exist, it picks the most restrictive one.
// Retry-After is only added on denial.
func writeRateLimitHeaders(w http.ResponseWriter, results map[string]keys.RatelimitConfigAndResult, clk clock.Clock) {
	if len(results) == 0 {
		return
	}

	mostRestrictive := findMostRestrictive(results)
	if mostRestrictive == nil {
		return
	}

	resp := mostRestrictive.Response
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
	h.Set("X-RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(resp.Reset.Unix(), 10))

	if !resp.Success {
		retryAfter := math.Ceil(resp.Reset.Sub(clk.Now()).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		h.Set("Retry-After", strconv.FormatInt(int64(retryAfter), 10))
	}
}

// findMostRestrictive returns the rate limit result that should be shown to the client.
// A denial always takes precedence, otherwise the one with the lowest remaining wins.
func findMostRestrictive(results map[string]keys.RatelimitConfigAndResult) *keys.RatelimitConfigAndResult {
	var best *keys.RatelimitConfigAndResult

	for _, r := range results {
		if r.Response == nil {
			continue
		}

		if best == nil {
			rCopy := r
			best = &rCopy
			continue
		}

		deniedBefore := !best.Response.Success
		deniedNow := !r.Response.Success

		// Already rate limited, don't overwrite.
		if deniedBefore && !deniedNow {
			continue
		}

		// Newly rate limited, overwrite.
		if deniedNow && !deniedBefore {
			rCopy := r
			best = &rCopy
			if r.Response.Remaining == 0 {
				return best
			}
			continue
		}

		// Lower remaining = more restrictive.
		if r.Response.Remaining < best.Response.Remaining {
			rCopy := r
			best = &rCopy
			if deniedNow && r.Response.Remaining == 0 {
				return best
			}
		}
	}

	return best
}
