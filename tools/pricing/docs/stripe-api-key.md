# Stripe API key setup

The `pricing` tool needs one Stripe **restricted key** per environment (sandbox,
canary, production). Restricted keys are least-privilege: this one can only edit
the billing catalog — it can never touch customers, subscriptions, invoices, or
money, so even a leaked key cannot grant credits or issue refunds.

## Permissions

In the Stripe Dashboard: **Developers → API keys → Create restricted key**.

Set exactly these four to **Write**, and leave everything else at **None**:

| Section in the picker | Resource | Permission |
| --------------------- | -------- | ---------- |
| Core                  | **Products** | Write |
| Billing               | **Prices** | Write |
| Billing               | **Billing Meters** | Write |
| Webhook Endpoints     | **Webhook Endpoints, Event Destinations** | Write |

`Write` includes read, so the same key serves `plan`, `verify`, and `export`.

### Do not enable these look-alikes

They sound related but the tool never uses them — leave at **None**:

- **Billing Meter Events** / **Billing Meter Event Adjustments** — these *report*
  usage. The tool only manages meter *definitions*.
- **Usage Records** — usage reporting, not catalog.
- **Events** (Core) — the event log, unrelated to webhook endpoints.
- **Customers, Subscriptions, Invoices, Charges and Refunds, Coupons,
  Credit Grants** — never touched. Keeping these at None is what bounds the blast
  radius.

## One key per environment

Create the key while signed into the Stripe account for that environment:

- **sandbox** / **canary** — a **test-mode** key (`rk_test_…`).
- **production** — a **live-mode** key (`rk_live_…`).

The tool refuses to run a live key against a non-production environment, or a
test key against production, so a misfiled key fails closed rather than touching
the wrong account.

## Where the key goes

Two ways to supply it, in priority order:

1. **`STRIPE_SECRET_KEY` environment variable** — the simple path for sandbox/dev
   where you bring your own key:

   ```bash
   export STRIPE_SECRET_KEY="rk_test_..."
   mise run pricing plan --env sandbox
   ```

2. **AWS Secrets Manager** — for canary/production, store the key in the
   `unkey/stripe` secret in that account (`us-east-1`) as JSON:

   ```json
   { "api_key": "rk_live_..." }
   ```

   The tool reads it through the per-account AWS profile you configure for that
   environment (via `AWS_PROFILE` or `AWS_PROFILE_<ENV>`, see
   [`.env.example`](../.env.example)). If the SSO session is stale it runs
   `aws sso login --sso-session unkey` (the SSO session, override with
   `AWS_SSO_SESSION`), not the per-account profile. Set the secret once with your
   own profile:

   ```bash
   aws secretsmanager put-secret-value \
     --profile "$AWS_PROFILE_PRODUCTION" \
     --region us-east-1 \
     --secret-id unkey/stripe \
     --secret-string '{"api_key":"rk_live_..."}'
   ```

   The secret name and region default to `unkey/stripe` / `us-east-1`; override
   with `STRIPE_SECRET_ID` and `AWS_REGION` if your account differs.

## Rotation

Restricted keys can be rolled in the Dashboard (**API keys → ⋯ → Roll key**).
After rolling, update the value (`STRIPE_SECRET_KEY` or the `unkey/stripe`
secret) — nothing else in the tool changes, since it never persists the key.

## Read-only variant (optional)

If you want CI or a drift check to never hold a write key, create a second key
with the same four resources set to **Read** instead of Write, and use it for
`plan`, `verify`, and `export`. Reserve the Write key for `apply`.
