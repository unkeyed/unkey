// Package buildlimitsync implements the CronService.RunBuildLimitSync
// handler. The handler keeps the build concurrency rules in Restate's rule
// book.
//
// A rule book is the cluster-wide table Restate consults before it dispatches
// an invocation. Each rule matches a pattern of <scope>/<limit key> and caps
// how many matching invocations run at once; a pattern no rule matches is
// unlimited. This handler writes "builds/*", which caps a workspace's
// concurrent builds
package buildlimitsync

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
)

// defaultRulePattern caps every workspace that has no rule of its own
const defaultRulePattern = restateadmin.BuildConcurrencyScope + "/*"

// defaultConcurrency is the build concurrency every plan tier grants today
const defaultConcurrency = 1

// defaultRuleDescription must stay constant across releases. A rule's version
// advances only when a write changes its limits or its disabled flag, so a
// repeated identical write stays a no-op and queued builds are not woken for
// nothing
const defaultRuleDescription = "default per-workspace build concurrency"

// RuleBook is the part of the Restate admin API this handler needs. It is an
// interface because the integration harness builds the cron service before
// the Restate container it talks to exists
type RuleBook interface {
	UpsertRules(ctx context.Context, rules []restateadmin.RuleUpsert) ([]restateadmin.Rule, error)
}

// Config holds the handler's dependencies
type Config struct {
	// RuleBook reads and writes Restate's rule book. Must not be nil
	RuleBook RuleBook
	// Heartbeat is pinged after a successful sync. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunBuildLimitSync
type Handler struct {
	ruleBook  RuleBook
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.RuleBook, "RuleBook must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{ruleBook: cfg.RuleBook, heartbeat: cfg.Heartbeat}, nil
}

// Handle writes the default build concurrency rule and fails if the rule book
// does not report it back enabled at that concurrency. The heartbeat is sent
// last, so a green heartbeat means the write was accepted.
//
// The write is unconditional because Restate ignores one that changes nothing:
// an identical upsert advances neither the rule's version nor its last-modified
// time, so reading the book first would only add a step that can fail. The rule
// book is the only place the limit lives and nothing is cached in the worker,
// so a failed run leaves nothing half applied and the next tick converges.
//
// Stateless: the VO key is fixed at "build-limit-sync" so a wedged invocation
// cannot block other cron handlers, and the CronJob sets concurrencyPolicy
// Forbid so ticks never overlap. A wedged invocation does block later ticks of
// this handler, which is why the registered retry policy kills rather than
// pauses on exhaustion
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunBuildLimitSyncRequest,
) (*hydrav1.RunBuildLimitSyncResponse, error) {
	written, err := restate.Run(ctx, func(rc restate.RunContext) ([]restateadmin.Rule, error) {
		return h.ruleBook.UpsertRules(rc, []restateadmin.RuleUpsert{{
			Pattern:     defaultRulePattern,
			Concurrency: defaultConcurrency,
			Description: defaultRuleDescription,
		}})
	}, restate.WithName("upsert default rule"))
	if err != nil {
		return nil, fmt.Errorf("upsert rule %s: %w", defaultRulePattern, err)
	}
	if len(written) != 1 || written[0].Pattern != defaultRulePattern {
		return nil, fmt.Errorf("wrote rule %s, rule book returned %d rules", defaultRulePattern, len(written))
	}
	if written[0].Concurrency != defaultConcurrency || written[0].Disabled {
		return nil, fmt.Errorf("rule %s: wrote concurrency %d enabled, rule book returned concurrency %d disabled=%t",
			defaultRulePattern, defaultConcurrency, written[0].Concurrency, written[0].Disabled)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunBuildLimitSyncResponse{}, nil
}
