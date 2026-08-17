// Package legacybilling implements the manually invoked Restate workflow for
// legacy API billing.
//
// One invocation prepares one standalone Stripe draft invoice for a workspace
// and completed UTC calendar month. The request identifies the workspace,
// year, and month.
//
// The workflow reads current pricing from workspaces.subscriptions, rejects
// deleted or subscription-billed workspaces, and reads exact-month billable
// verification and ratelimit usage from ClickHouse. Fixed plan and support
// charges are billed in full. Usage follows the inclusive legacy tiers. No
// charge is prorated.
//
// Stripe writes are idempotent and resumable. The workflow creates or resumes
// a standalone draft with auto-advance disabled, excludes unrelated pending
// invoice items, and reconciles every line before succeeding. It never
// finalizes, sends, pays, or attaches the invoice to a subscription.
package legacybilling
