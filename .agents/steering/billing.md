# Billing System Guide

Unkey uses a two-product billing model via Stripe: Auth (API key verification quotas) and Deploy (compute usage). This doc covers how the pieces connect across TypeScript (dashboard) and Go (control plane).

## Architecture Overview

```
Stripe ←→ Dashboard (webhooks + tRPC mutations) ←→ MySQL (deploy_plan column)
Stripe ←  Ctrl Worker (hourly meter push)        ←  ClickHouse (Heimdall instance meters)
Stripe ←  tools/pricing (catalog reconciliation)
```

### Data Flow

1. **Subscription lifecycle** (dashboard tRPC): subscribeDeploy / changeDeployPlan / cancelDeploy mutate Stripe subscriptions
2. **Plan sync** (Stripe webhook → dashboard): `customer.subscription.*` events detect the deploy plan-fee price and write `workspaces.deploy_plan`
3. **Usage metering** (ctrl worker cron): hourly job reads month-to-date from ClickHouse, pushes to Stripe Billing Meters
4. **Credit grants** (Stripe webhook → dashboard): `invoice.paid` webhook sums net deploy fee lines and grants usage credits
5. **Deploy gate** (ctrl-api): checks `deploy_plan` / `deploy_plan_override` before CreateProject

## Stripe Object Model

| Concept | Stripe Object | Identity |
|---------|--------------|----------|
| Deploy plan fee | Price with `plan=<tier>` metadata | `lookup_key` (e.g., `plan.starter`) |
| Usage meter | Billing Meter | event_name (e.g., `deploy.cpu_seconds`) |
| Metered price | Price on usage meter | `lookup_key` (e.g., `usage.cpu_seconds`) |
| API product | Product with quota metadata | `managed_by=unkey-pricing` + `pricing_key` |
| Webhook endpoint | Webhook Endpoint | URL match |

The `tools/pricing` tool manages the catalog. It never keys off Stripe IDs, only `lookup_key` and metadata.

## Dashboard (TypeScript) Side

### tRPC Routers (`lib/trpc/routers/stripe/`)

- **subscribeDeploy.ts**: Creates or appends deploy plan-fee + metered prices to subscription. Uses `billing_mode: "classic"` for Stripe creates.
- **changeDeployPlan.ts**: Reprices the deploy plan-fee item. Upgrades use `always_invoice` (immediate charge triggers credit top-up). Downgrades use `proration_behavior: "none"` (keeps current period credits).
- **cancelDeploy.ts**: Removes deploy items from subscription, clears `deploy_plan`.
- **getDeploySubscription.ts**: Reads `deploy_plan` from workspace (local signal, no Stripe call).
- **getDeployEntitlement.ts**: Reads `deploy_plan` + `deploy_plan_override` for gate UI.
- **getUpcomingInvoice.ts**: Previews next invoice from Stripe (cached 30s client-side).
- **subscriptionGuards.ts**: Shared helpers like `findApiItem` and `findDeployItem` that locate items by product/price metadata on mixed subscriptions.

### Stripe Webhooks (`app/api/webhooks/stripe/route.ts`)

Two key handlers:

1. **`customer.subscription.*`**: Detects deploy plan-fee price via `plan` metadata, writes `workspaces.deploy_plan` (starter|pro|business). Clears when no deploy item present or subscription deleted.
2. **`invoice.paid`**: Computes net deploy fee from invoice lines, grants Stripe credit balance for the billing period.

### Deploy Plan Detection (`lib/stripe/deployPlan.ts`)

```typescript
// Reads price metadata to find the deploy plan tier
function detectDeployPlan(subscription): "starter" | "pro" | "business" | null
```

### Deploy Billing Utilities (`lib/stripe/deployBilling.ts`)

- `netDeployFee(invoice)`: Pure function summing deploy plan-fee lines on an invoice. Handles subscribe, renewal, upgrade netting, and downgrade netting.
- `DEPLOY_PLANS`: Ordered list (lowest to highest) used to derive upgrade vs downgrade direction.

### ClickHouse Query (`web/internal/clickhouse/src/deploy_billing.ts`)

Mirrors the Go instance meter query. Must stay in sync with `pkg/clickhouse/instance_meter.go`. Returns per-workspace month-to-date usage for dashboard display.

## Go (Control Plane) Side

### Hourly Billing Push (`svc/ctrl/worker/cron/deploybilling/`)

- Restate cron handler keyed by `"YYYY-MM"` (billing period)
- Reads billable workspaces from MySQL (`workspace_deploy_billing` query)
- Reads month-to-date usage from ClickHouse instance meter
- Pushes absolute totals to Stripe via `billingmeter.Pusher` (set, not increment)
- Idempotent: retries, overlapping ticks, and replays all converge
- No-op when ClickHouse or Stripe is not configured

### Billing Meter Interface (`svc/ctrl/internal/billingmeter/`)

```go
type Pusher interface {
    Push(ctx context.Context, req PushRequest) (int, error)
}
```

- `NewStripe(secretKey)`: Real implementation using Stripe Billing Meter Events API
- `NewNoop()`: No-op for environments without billing

### Deploy Gate (`svc/ctrl/services/project/deploy_gate.go`)

```go
func deployEntitled(plan, override sql.NullString) bool {
    return isSet(plan) || isSet(override)
}
```

Checked in `CreateProject`. Enforcement is gated by config flag (`DeployGate.Enforce`). In observe mode, logs `deploy_gate.would_block` without blocking.

### Billing Period (`pkg/billingperiod/`)

Shared "YYYY-MM" parser used across cron handlers. Enforces strict format (4-digit year, 2-digit month).

## tools/pricing

Standalone Go tool managing the Stripe catalog. Commands:

- `plan`: Show diff (read-only)
- `apply --env <env>`: Reconcile Stripe to match catalog
- `verify --env <env>`: Exit non-zero if drift detected
- `export --env <env>`: Print dashboard env block

Catalog declared in `catalog.go`. Price changes create new immutable prices and transfer `lookup_key`. Run via `mise run pricing`.

## Key Conventions

1. **Local signal over Stripe calls**: Dashboard reads `workspaces.deploy_plan` (synced by webhook) instead of calling Stripe for plan status in hot paths.
2. **Absolute totals, not deltas**: Billing push sends month-to-date absolute values. Stripe `last` aggregation means the newest event wins.
3. **Mixed subscriptions**: A single Stripe subscription can have both Auth and Deploy items. Always use `findApiItem`/`findDeployItem` helpers, never `items[0]`.
4. **Immutable prices**: Never edit a price. Create new, transfer `lookup_key`, archive old.
5. **Upgrade = immediate charge, downgrade = keep period**: Upgrade triggers proration invoice (grants top-up credits). Downgrade waits until renewal.
6. **TS and Go must stay in sync**: `web/internal/clickhouse/src/deploy_billing.ts` mirrors `pkg/clickhouse/instance_meter.go`.

## Adding a New Meter

1. Add the meter definition to `tools/pricing/catalog.go`
2. Add the metered price with `lookup_key` = `usage.<name>`
3. Run `mise run pricing apply --env sandbox` to create in Stripe
4. Add the field to `billingmeter.MeterValues` and `PushRequest`
5. Update the ClickHouse query in both Go and TS
6. Update `netDeployFee` if the new meter affects credit grants
