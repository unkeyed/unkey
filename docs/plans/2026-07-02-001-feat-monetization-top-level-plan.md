---
type: feat
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: ce-plan-bootstrap
created: 2026-07-02
title: Promote end-user billing to a top-level "Monetization" surface with per-keyspace/namespace enablement
---

# Monetization: top-level surface + per-resource enablement

## Goal Capsule

End-user billing (a customer billing their own users from Unkey usage) currently
lives buried in `Settings > Billing` alongside Unkey's *own* billing, and it
meters **all** of an identity's usage indiscriminately. Promote it to a
first-class top-level nav item and let the customer choose exactly which
keyspaces (APIs) and ratelimit namespaces are monetized. Usage from
non-enabled resources is excluded from invoices. Pricing stays one rate card
per identity.

## Product Contract

### Requirements

- **R1** End-user billing is a top-level nav item ("Monetization"), separate
  from `Settings > Billing` (which remains Unkey's own billing). The existing
  Stripe Connect + rate-card configuration moves there.
- **R2** The customer selects which keyspaces (APIs) and which ratelimit
  namespaces are enabled for billing. Selection is **opt-in**: with nothing
  selected, nothing is billed.
- **R3** Only usage attributed to enabled keyspaces/namespaces contributes to an
  identity's billable quantities. Usage on non-enabled resources is excluded
  from the invoice draft and the period-close push.
- **R4** Pricing is unchanged: an identity resolves to one rate card
  (selection -> assignment -> workspace default) that prices its combined usage
  across all enabled resources (R19 parity preserved).
- **R5** The surface shows how billing attaches to an end-user: the identity ->
  rate card resolution and the identity -> provider (Stripe) customer mapping.
- **R6** All mutations are workspace-admin gated (consistent with the existing
  rate-card / Stripe-connect mutations).

### Key Technical Decisions

- **KTD1 — enablement storage.** New MySQL table
  `billing_billable_resources(workspace_id, resource_type enum('keyspace','namespace'),
  resource_id, created_at, updated_at)` with `UNIQUE(workspace_id, resource_type,
  resource_id)`. A row's presence = enabled. Chosen over boolean columns on
  `key_auth`/`ratelimit_namespaces` to keep billing config out of resource
  tables and to mirror the `workspace_billing_settings` pattern.
- **KTD2 — usage scoping in ClickHouse (RECOMMENDED: dimensioned rollups).**
  The per-identity rollups (`041/042/044`) group `key_space_id`/`namespace_id`
  away, so they cannot scope. Add the resource dimension to persisted rollups:
  new `*_v2` per-identity rollups keyed additionally by `key_space_id`
  (verifications, credits) and `namespace_id` (ratelimits), fed by new MVs.
  `GetBillableUsagePerIdentity` takes the enabled keyspace/namespace ID sets and
  sums only matching rows per identity.
  - *Why not query the source tables directly (lighter, no new MVs):* the
    ratelimits source `ratelimits_raw_v3` has a 1-month TTL, so a delayed
    period-close (see the known cron-outage limitation) could find the raw data
    gone. Persisted dimensioned rollups survive past the raw TTL, matching the
    existing architecture and the backfill mechanism. Fallback to source-query
    only if the added rollup cardinality proves problematic.
  - `041/042/044` (added on this branch, not on `main`, consumed only by
    `GetBillableUsagePerIdentity`) are superseded by the `_v2` dimensioned
    rollups and removed in the same change to avoid dead MVs.
- **KTD3 — empty-set semantics.** A workspace with zero enabled resources bills
  nothing; the period-close skips it before querying ClickHouse.
- **KTD4 — nav naming.** Top-level item is "Monetization" (not "Billing") to
  avoid collision with the existing Unkey-billing `Settings > Billing`.

## Implementation Units

### U1. Enablement schema + queries
- **Files:** `web/internal/db/src/schema/billing.ts` (new `billingBillableResources`
  table), regenerate `pkg/mysql/schema/billing_billable_resources.sql` via
  `mise run generate-sql`; new sqlc queries under `pkg/db/queries/`
  (`billing_billable_resources_list.sql`, `_upsert.sql`, `_delete.sql`),
  regenerate with `go tool sqlc generate` (+ remove the `delete_me.go` placeholder).
- **Approach:** presence-row model per KTD1. List returns all rows for a
  workspace; upsert = `INSERT IGNORE`; delete by (workspace, type, resource_id).
- **Cites:** R2, KTD1.
- **Test:** sqlc round-trip covered indirectly by U3/U4 tests.

### U2. ClickHouse dimensioned rollups + scoped query
- **Files:** new `pkg/clickhouse/schema/045_*_v2.sql`, `046_*_v2.sql`,
  `047_*_v2.sql` (per-identity rollups carrying `key_space_id`/`namespace_id` +
  MVs); remove `041/042/044`; `pkg/clickhouse/billable_identity.go`
  (`GetBillableUsagePerIdentity` gains `enabledKeyspaces []string`,
  `enabledNamespaces []string` and filters/sums); extend
  `pkg/clickhouse/billable_identity_backfill.go` for the new grain.
- **Approach:** MVs mirror `041/042/044` but retain the resource id in the
  ORDER BY and GROUP BY. Billing query filters `key_space_id IN (...)` for
  verifications/credits and `namespace_id IN (...)` for ratelimits, then
  aggregates to one row per identity.
- **Cites:** R3, R4, KTD2.
- **Test scenarios:** `pkg/clickhouse/billable_identity_test.go` — (a) usage on
  an enabled keyspace counts, usage on a non-enabled keyspace is excluded; (b)
  same for namespaces; (c) an identity spanning two keyspaces, only one enabled,
  bills the enabled subset; (d) backfill idempotency + reconstruction at the new
  grain; (e) empty enabled-set yields zero rows.

### U3. Backend wiring (period-close + invoice draft)
- **Files:** `svc/ctrl/worker/cron/enduserbillingpush/periodclose.go`,
  `svc/ctrl/worker/cron/enduserbillingpush/vault.go` (or a new loader),
  `svc/api/routes/v2_billing_get_invoice_draft/handler.go`; a shared loader for
  the enabled resource sets from `billing_billable_resources`.
- **Approach:** per workspace, load enabled keyspace/namespace ID sets; if both
  empty, skip (KTD3); else pass to `GetBillableUsagePerIdentity`. Applies to
  both the money path and the preview so R19 parity holds.
- **Cites:** R3, R4, KTD3.
- **Test scenarios:** `periodclose_test.go` — a workspace with one enabled
  keyspace bills only that keyspace's usage; a workspace with nothing enabled
  pushes nothing. Invoice-draft test mirrors it.

### U4. tRPC: resources + enablement
- **Files:** `web/apps/dashboard/lib/trpc/routers/billing/monetization/`
  (`list-billable-resources.ts` — lists the workspace's APIs (named, ->
  key_auth_id) and ratelimit namespaces with an enabled flag; `set-billable-resource.ts`
  — toggle enable/disable, `requireWorkspaceAdmin`); register in the billing router.
- **Approach:** join `apis`/`key_auth` and `ratelimit_namespaces` against
  `billing_billable_resources`. Audit-log toggles like the rate-card mutations.
- **Cites:** R2, R6.
- **Test:** existing tRPC test patterns; assert admin gating.

### U5. Top-level nav + route + move existing config
- **Files:** `web/apps/dashboard/lib/navigation/leaves.ts` (new `monetization`
  leaf, icon e.g. `Coins`/`ReceiptText`), `lib/navigation/routes/` (new
  `monetization.ts` route group + index wiring),
  new `app/(app)/[workspaceSlug]/monetization/page.tsx`; move
  `settings/billing/components/stripe-connect-card.tsx` and the rate-card client
  out of `settings/billing/client.tsx` into the new page; drop them from the
  settings page.
- **Approach:** relocate the already-built Stripe Connect + rate-card UI; leave
  Unkey's own billing (deploy billing, plans, subscription) in `Settings > Billing`.
- **Cites:** R1, KTD4.
- **Test:** route table tests under `lib/navigation/routes/` (mirror existing
  `*.test.ts`); `no-handrolled-routes.test.ts` stays green.

### U6. Monetization config UI
- **Files:** new components under `app/(app)/[workspaceSlug]/monetization/`.
- **Approach:** sections — (1) Stripe Connect status (moved), (2) Rate cards
  (moved; create/select-default/selectable), (3) **Billable resources**: two
  multiselect/toggle lists (keyspaces via APIs, ratelimit namespaces) backed by
  U4, (4) **Attach billing to a user**: surface an identity's resolved rate card
  and its provider-customer mapping (read the existing identity billing binding).
- **Cites:** R1, R2, R5.
- **Test:** component render + the tRPC hooks; manual walkthrough.

## Sequencing & Dependencies

U1 -> U2 -> U3 (backend spine, gate the money path first). U4 depends on U1.
U5 then U6 (UI) depend on U4. U2 requires `mise run generate` for CH schema and a
backfill run for historical periods (per the existing runbook, now at the new grain).

## Risks

- **Rollup cardinality** grows by (identity x keyspace) and (identity x
  namespace); acceptable for typical workspaces, watched via KTD2 fallback.
- **Migration ordering:** the new `_v2` MVs only capture new inserts; historical
  periods need the extended backfill before a scoped invoice is trusted (same
  constraint and runbook as the current rollups).
- **Parity:** U3 must apply the identical enabled-set filter to both the
  invoice-draft preview and the period-close push, or R19 breaks.

## Definition of Done

R1-R6 met; scoped usage proven by U2/U3 tests; nav item live and the old
`Settings > Billing` no longer shows end-user billing; admin gating on all
mutations; `mise run bazel` + affected `bazel test` + dashboard `tsc` green.
