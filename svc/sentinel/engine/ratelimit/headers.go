package ratelimit

import (
	"math"
	"net/http"
	"strconv"

	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/clock"
)

// writePolicyRateLimitHeaders sets rate limit headers on the response using the
// X-Policy-RateLimit-* prefix to avoid collision with KeyAuth's X-RateLimit-* headers.
func writePolicyRateLimitHeaders(w http.ResponseWriter, resp rl.RatelimitResponse, clk clock.Clock) {
	h := w.Header()
	h.Set("X-Policy-RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
	h.Set("X-Policy-RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
	h.Set("X-Policy-RateLimit-Reset", strconv.FormatInt(resp.Reset.Unix(), 10))

	if !resp.Success {
		retryAfter := math.Ceil(resp.Reset.Sub(clk.Now()).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		h.Set("Retry-After", strconv.FormatInt(int64(retryAfter), 10))
	}
}
