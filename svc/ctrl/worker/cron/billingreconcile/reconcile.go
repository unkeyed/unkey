package billingreconcile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	reconcile "github.com/unkeyed/unkey/svc/ctrl/internal/billingreconcile"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

// VO key prefix, keeping this handler off the other period-keyed tasks' object.
const keyPrefix = "deploy-billing-reconcile-"

// Cap parallel reconciles per batch: each makes several Stripe calls, so bound
// the fan-out under Stripe's rate limit.
const batchConcurrency = 16

// Handle reconciles the closed period's invoices against live ClickHouse usage:
// enumerate billable workspaces once, fan out one journaled reconcile each, log
// each verdict, and page Slack on structural findings. A run failure returns a
// terminal error (a failed Restate invocation); no heartbeat (see Handler).
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeployBillingReconcileRequest,
) (*hydrav1.RunDeployBillingReconcileResponse, error) {
	// TrimPrefix is a no-op on a bare key, so an in-flight invocation keyed the
	// old way during a rolling deploy still parses.
	period := strings.TrimPrefix(restate.Key(ctx), keyPrefix)
	logger.Info("running deploy billing reconcile", "billing_period", period)

	if h.engine == nil {
		logger.Info("deploy billing reconcile disabled (no engine configured)",
			"billing_period", period,
		)
		return &hydrav1.RunDeployBillingReconcileResponse{}, nil //nolint:exhaustruct // no-op: zero counts
	}

	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}

	// Only reconcile a closed period. CloseAllowed (like the close) accepts it
	// once wall clock passes the end OR the request carries a Stripe period_end
	// that rolled it (test clocks); the scheduled cron omits period_end.
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	if !p.CloseAllowed(now, req.GetPeriodEnd()) {
		return nil, restate.TerminalError(
			fmt.Errorf("billing period %s has not ended yet (ends %s); reconcile runs after the period closes", period, p.End().Format("2006-01-02T15:04:05Z")),
		)
	}

	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return h.db.ListDeployBillableWorkspaces(rc)
	}, restate.WithName("list deploy billable workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list deploy billable workspaces: %w", err)
	}

	refs, skipped := buildRefs(rows)
	// Stable order: RunAsync steps are journaled by position, so replays must
	// iterate identically.
	sort.Slice(refs, func(i, j int) bool { return refs[i].WorkspaceID < refs[j].WorkspaceID })
	for _, id := range skipped {
		logger.Error("deploy billing reconcile: workspace is billable but has no stripe subscription id; skipping",
			"billing_period", period,
			"workspace_id", id,
		)
	}

	results, runErrs := h.reconcileAll(ctx, p, refs)

	// Emit before the heartbeat decision: verdicts are worth logging even if one
	// workspace's read failed.
	h.emit(ctx, period, results)

	s := summarize(results, len(skipped))
	logger.Info("deploy billing reconcile complete",
		"billing_period", period,
		"workspaces_reconciled", len(results),
		"workspaces_clean", s.clean,
		"workspaces_late_data_underbill", s.lateDataUnderbill,
		"workspaces_overbill", s.overbill,
		"workspaces_structural", s.structural,
		"workspaces_skipped_no_subscription", s.skipped,
		"workspaces_run_failed", len(runErrs),
	)

	// A run failure (a terminal read error) fails the invocation so it shows up
	// in Restate; drift is a finding, not a failure.
	if len(runErrs) > 0 {
		for _, e := range runErrs {
			logger.Error("deploy billing reconcile: workspace reconcile failed",
				"billing_period", period,
				"error", e,
			)
		}
		return nil, restate.TerminalError(
			fmt.Errorf("deploy billing reconcile for %s failed on %d workspace(s)", period, len(runErrs)),
		)
	}

	return &hydrav1.RunDeployBillingReconcileResponse{
		WorkspacesClean:             int32(s.clean),
		WorkspacesLateDataUnderbill: int32(s.lateDataUnderbill),
		WorkspacesOverbill:          int32(s.overbill),
		WorkspacesStructural:        int32(s.structural),
		WorkspacesSkipped:           int32(s.skipped),
	}, nil
}

// reconcileAll fans out one journaled reconcile per workspace in batches.
// RunAsync journals each result (replays skip re-hitting Stripe); a terminal
// error goes to runErrs so the caller can withhold the heartbeat.
func (h *Handler) reconcileAll(
	ctx restate.ObjectContext,
	p billingperiod.Period,
	refs []reconcile.WorkspaceRef,
) (results []reconcile.Result, runErrs []error) {
	for start := 0; start < len(refs); start += batchConcurrency {
		end := min(start+batchConcurrency, len(refs))
		batch := refs[start:end]

		futures := make([]restate.RunAsyncFuture[reconcile.Result], len(batch))
		for i, ref := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (reconcile.Result, error) {
				return h.engine.ReconcileWorkspace(rc, ref, p)
			}, restate.WithName("reconcile "+ref.WorkspaceID))
		}

		for i, fut := range futures {
			result, err := fut.Result()
			if err != nil {
				runErrs = append(runErrs, fmt.Errorf("reconcile %s: %w", batch[i].WorkspaceID, err))
				continue
			}
			results = append(results, result)
		}
	}
	return results, runErrs
}

// emit records the pass's per-workspace verdicts to the structured log and pages
// Slack on each structural finding. Per-workspace detail lives in the log line;
// the caller logs the monthly summary (counts by verdict). The Slack send is
// journaled (restate.RunVoid) so it fires exactly once and is not lost on
// replay, mirroring the quota check's Slack path.
func (h *Handler) emit(ctx restate.ObjectContext, period string, results []reconcile.Result) {
	for _, r := range results {
		h.logResult(period, r)
	}

	for _, r := range structuralResults(results) {
		h.pageStructural(ctx, period, r)
	}
}

// logResult writes the verdict plus one line per finding.
func (h *Handler) logResult(period string, r reconcile.Result) {
	attrs := []any{
		"billing_period", period,
		"workspace_id", r.WorkspaceID,
		"invoice_id", r.InvoiceID,
		"verdict", string(r.Verdict),
		"findings", len(r.Findings),
	}
	logger.Info("deploy billing reconcile verdict", attrs...)

	for _, f := range r.Findings {
		logger.Info("deploy billing reconcile finding",
			"billing_period", period,
			"workspace_id", r.WorkspaceID,
			"invoice_id", r.InvoiceID,
			"check", string(f.Check),
			"class", string(f.Class),
			"meter", string(f.Meter),
			"drift_cents", f.DriftCents,
			"detail", f.Detail,
		)
	}
}

// pageStructural pages engineering: an error log (for log alerting) plus, when
// configured, a journaled Slack post (fires exactly once on replay).
func (h *Handler) pageStructural(ctx restate.ObjectContext, period string, r reconcile.Result) {
	logger.Error("deploy billing reconcile STRUCTURAL finding; billing code or catalog bug",
		"billing_period", period,
		"workspace_id", r.WorkspaceID,
		"invoice_id", r.InvoiceID,
		"findings", len(r.Findings),
	)

	if h.slackWebhookURL == "" {
		return
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return sendSlackStructural(rc, h.slackWebhookURL, period, r)
	}, restate.WithName("page structural finding")); err != nil {
		// A failed page must not fail the pass (the reconcile itself succeeded);
		// the error log above still reaches log alerting.
		logger.Error("deploy billing reconcile: failed to send structural Slack page",
			"billing_period", period,
			"workspace_id", r.WorkspaceID,
			"error", err,
		)
	}
}

// sendSlackStructural posts a structural-finding page to the Slack webhook.
func sendSlackStructural(ctx context.Context, webhookURL, period string, r reconcile.Result) error {
	var findingLines strings.Builder
	for _, f := range r.Findings {
		if f.Class != reconcile.VerdictStructural {
			continue
		}
		meter := string(f.Meter)
		if meter == "" {
			meter = "-"
		}
		fmt.Fprintf(&findingLines, "- `%s`/`%s` meter=%s drift=%d¢: %s\n",
			f.Check, f.Class, meter, f.DriftCents, f.Detail)
	}

	payload := slack.Payload{
		Text: fmt.Sprintf("Deploy billing reconcile: STRUCTURAL finding for workspace %s (%s)", r.WorkspaceID, period),
		Blocks: []slack.Block{
			slack.NewHeaderBlock("Deploy billing reconcile: structural finding"),
			slack.NewSectionBlock(
				slack.NewMarkdownField(fmt.Sprintf("*Billing period:*\n%s", period)),
				slack.NewMarkdownField(fmt.Sprintf("*Workspace ID:*\n`%s`", r.WorkspaceID)),
				slack.NewMarkdownField(fmt.Sprintf("*Invoice ID:*\n`%s`", r.InvoiceID)),
				slack.NewMarkdownField(fmt.Sprintf("*Verdict:*\n%s", r.Verdict)),
			),
			slack.NewSectionBlock(
				slack.NewMarkdownField(fmt.Sprintf("*Findings:*\n%s", findingLines.String())),
			),
		},
	}
	return slack.NewClient().Send(ctx, webhookURL, payload)
}
