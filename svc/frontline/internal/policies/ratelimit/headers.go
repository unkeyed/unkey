package ratelimit

import (
	"math"
	"net/http"
	"strconv"

	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/clock"
)

// writePolicyRateLimitHeaders sets X-RateLimit-* headers on the response.
// Multiple policies (KeyAuth, RateLimit) write to the same standard headers,
// so we only overwrite when the current result is more restrictive.
// Retry-After is only added on denial.
func writePolicyRateLimitHeaders(w http.ResponseWriter, resp rl.RatelimitResponse, clk clock.Clock) {
	h := w.Header()

	if !shouldOverwrite(h, resp) {
		return
	}

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

// shouldOverwrite returns true if resp is more restrictive than existing headers.
func shouldOverwrite(h http.Header, resp rl.RatelimitResponse) bool {
	existing := h.Get("X-RateLimit-Remaining")

	if existing == "" {
		return true
	}

	deniedBefore := h.Get("Retry-After") != ""
	deniedNow := !resp.Success

	if deniedBefore && !deniedNow {
		return false
	}

	if deniedNow && !deniedBefore {
		return true
	}

	existingRemaining, err := strconv.ParseInt(existing, 10, 64)
	if err != nil {
		return true
	}
	if resp.Remaining != existingRemaining {
		return resp.Remaining < existingRemaining
	}

	if deniedNow {
		existingReset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64)
		if err != nil {
			return true
		}
		return resp.Reset.Unix() > existingReset
	}

	return false
}
