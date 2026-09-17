// Package buildlimitsync implements the CronService.RunBuildLimitSync
// handler. The handler keeps the build concurrency rules in Restate in step
// with limits.builds_concurrent_max.
//
// Restate consults its concurrency rules, written through the admin API,
// before it dispatches an invocation. Each rule matches a pattern of
// <scope>/<limit key> and caps how many matching invocations run at once; a
// pattern no rule matches is unlimited. This handler writes "builds/*" for
// every workspace and one "builds/<workspace_id>" rule for each workspace
// whose limit is above the default, and deletes such a rule once the limit is
// back at the default
package buildlimitsync

import (
	"context"
	"fmt"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// rulePatternPrefix starts every build concurrency rule pattern
const rulePatternPrefix = restateadmin.BuildConcurrencyScope + "/"

// defaultRulePattern caps every workspace that has no rule of its own
const defaultRulePattern = rulePatternPrefix + "*"

// defaultConcurrency is the build concurrency every plan tier grants today. A
// limits row at or below it needs no rule of its own
const defaultConcurrency = 1

// Rule descriptions must stay constant across releases. A rule's version
// advances only when a write changes its limits or its disabled flag, so a
// repeated identical write stays a no-op and queued builds are not woken for
// nothing
const (
	defaultRuleDescription   = "default per-workspace build concurrency"
	workspaceRuleDescription = "per-workspace build concurrency from limits"
)

// RestateRules is the part of the Restate admin API this handler needs. It is
// an interface because the integration harness builds the cron service before
// the Restate container it talks to exists
type RestateRules interface {
	ListRules(ctx context.Context) ([]restateadmin.Rule, error)
	UpsertRules(ctx context.Context, rules []restateadmin.RuleUpsert) error
	DeleteRules(ctx context.Context, patterns []string) error
}

// Config holds the handler's dependencies
type Config struct {
	// DB reads limits.builds_concurrent_max. Must not be nil
	DB db.Database
	// RestateRules reads and writes Restate's concurrency rules. Must not be nil
	RestateRules RestateRules
	// Heartbeat is pinged after a successful sync. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunBuildLimitSync
type Handler struct {
	db        db.Database
	rules     RestateRules
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.RestateRules, "RestateRules must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, rules: cfg.RestateRules, heartbeat: cfg.Heartbeat}, nil
}

// Handle writes the default rule and one rule per workspace whose limit is
// above the default, deletes every other "builds/<workspace_id>" rule, then
// pings the heartbeat, so a green heartbeat means the rules match the
// database. The upsert and the delete are separate Runs, so a crash between
// them leaves a stale workspace rule until the next tick. Ticks share the
// fixed key "build-limit-sync", so a stuck invocation blocks the following
// ticks; the retry policy kills it instead of pausing for that reason
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunBuildLimitSyncRequest,
) (*hydrav1.RunBuildLimitSyncResponse, error) {
	desired, err := restate.Run(ctx, func(rc restate.RunContext) ([]restateadmin.RuleUpsert, error) {
		rows, err := h.db.ListWorkspaceBuildConcurrencyAbove(rc, defaultConcurrency)
		if err != nil {
			return nil, err
		}
		rules := make([]restateadmin.RuleUpsert, 0, len(rows)+1)
		rules = append(rules, restateadmin.RuleUpsert{
			Pattern:     defaultRulePattern,
			Concurrency: defaultConcurrency,
			Description: defaultRuleDescription,
		})
		for _, row := range rows {
			rules = append(rules, restateadmin.RuleUpsert{
				Pattern:     rulePatternPrefix + row.WorkspaceID,
				Concurrency: uint32(row.BuildsConcurrentMax),
				Description: workspaceRuleDescription,
			})
		}
		return rules, nil
	}, restate.WithName("list workspace build limits"))
	if err != nil {
		return nil, fmt.Errorf("list workspace build limits: %w", err)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.rules.UpsertRules(rc, desired)
	}, restate.WithName("upsert rules")); err != nil {
		return nil, fmt.Errorf("upsert %d rules: %w", len(desired), err)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		existing, err := h.rules.ListRules(rc)
		if err != nil {
			return err
		}
		wanted := make(map[string]struct{}, len(desired))
		for _, rule := range desired {
			wanted[rule.Pattern] = struct{}{}
		}
		stale := make([]string, 0)
		for _, rule := range existing {
			if _, ok := wanted[rule.Pattern]; ok || !strings.HasPrefix(rule.Pattern, rulePatternPrefix) {
				continue
			}
			stale = append(stale, rule.Pattern)
		}
		if len(stale) == 0 {
			return nil
		}
		return h.rules.DeleteRules(rc, stale)
	}, restate.WithName("delete stale workspace rules")); err != nil {
		return nil, fmt.Errorf("delete stale workspace rules: %w", err)
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunBuildLimitSyncResponse{}, nil
}
