package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/gcra"
)

// gcraName names one rate limit. Each part carries its length, so no choice of
// identifier or namespace can collide with another limit. The limit is not part
// of the name, so changing it keeps what was spent. A name longer than MySQL
// stores is replaced by its SHA-256 under a different prefix, so it cannot
// collide with a readable name either.
func gcraName(req RatelimitRequest) string {
	name := fmt.Sprintf("rl:%d:%s:%d:%s:%d:%s:%d",
		len(req.WorkspaceID), req.WorkspaceID,
		len(req.Namespace), req.Namespace,
		len(req.Identifier), req.Identifier,
		req.Duration.Milliseconds())
	if len(name) <= gcra.NameMaxLength {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	return "rlh:" + hex.EncodeToString(sum[:])
}

// gcraEnabled reports whether the GCRA limiter decides req. A request it
// decides never uses the regional counters, even when the limiter fails.
func (s *service) gcraEnabled(req RatelimitRequest) bool {
	return s.gcra != nil && req.UseGCRA
}

func gcraRequest(req RatelimitRequest) gcra.Request {
	return gcra.Request{
		Name: gcraName(req),
		Rate: gcra.Rate{Limit: req.Limit, Duration: req.Duration},
		Cost: req.Cost,
		Time: req.Time,
	}
}

// ratelimitManyGCRA decides reqs with the GCRA limiter, all or nothing. The
// API rejects repeated checks; a call that names the same limit twice still
// returns an error here.
func (s *service) ratelimitManyGCRA(ctx context.Context, reqs []RatelimitRequest) ([]RatelimitResponse, error) {
	gcraReqs := make([]gcra.Request, len(reqs))
	for i, req := range reqs {
		gcraReqs[i] = gcraRequest(req)
	}
	results, err := s.gcra.AllowMany(ctx, gcraReqs)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("decide with the gcra limiter"))
	}
	responses := make([]RatelimitResponse, len(reqs))
	for i, req := range reqs {
		outcome := "denied"
		if results[i].Allowed {
			outcome = "passed"
		}
		metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, "gcra", outcome).Inc()
		responses[i] = RatelimitResponse{
			Success:   results[i].Allowed,
			Remaining: results[i].Remaining,
			Reset:     results[i].ResetAt,
			Limit:     req.Limit,
			Current:   results[i].Current,
		}
	}
	return responses, nil
}
