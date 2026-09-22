// Package buildlimitsync implements the CronService.RunBuildLimitSync
// handler. The handler keeps the build concurrency rules in Restate.
//
// Restate consults its concurrency rules, written through the admin API,
// before it dispatches an invocation. Each rule matches a pattern of
// <scope>/<limit key> and caps how many matching invocations run at once; a
// pattern no rule matches is unlimited. This handler writes "builds/*", which
// caps a workspace's concurrent builds
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

// RestateRules is the part of the Restate admin API this handler needs. It is
// an interface because the integration harness builds the cron service before
// the Restate container it talks to exists
type RestateRules interface {
	UpsertRules(ctx context.Context, rules []restateadmin.RuleUpsert) error
}

// Config holds the handler's dependencies
type Config struct {
	// RestateRules reads and writes Restate's concurrency rules. Must not be nil
	RestateRules RestateRules
	// Heartbeat is pinged after a successful sync. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunBuildLimitSync
type Handler struct {
	rules     RestateRules
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.RestateRules, "RestateRules must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{rules: cfg.RestateRules, heartbeat: cfg.Heartbeat}, nil
}

// Handle writes the default build concurrency rule on every tick and pings
// the heartbeat once Restate has accepted it, so a green heartbeat means the
// rule is in place. Ticks share the fixed key "build-limit-sync", so a stuck
// invocation blocks the following ticks; the retry policy kills it instead of
// pausing for that reason
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunBuildLimitSyncRequest,
) (*hydrav1.RunBuildLimitSyncResponse, error) {
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.rules.UpsertRules(rc, []restateadmin.RuleUpsert{{
			Pattern:     defaultRulePattern,
			Concurrency: defaultConcurrency,
			Description: defaultRuleDescription,
		}})
	}, restate.WithName("upsert default rule")); err != nil {
		return nil, fmt.Errorf("upsert rule %s: %w", defaultRulePattern, err)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunBuildLimitSyncResponse{}, nil
}
