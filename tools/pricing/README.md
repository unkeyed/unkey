# pricing

Reconciles the Unkey Stripe billing catalog against a Stripe account. The catalog
is declared as typed Go in [`catalog.go`](./catalog.go), with the schema behind it
in [`pricing.go`](./pricing.go). There is no state file: Stripe is the only state,
and the tool diffs the catalog against it.

Stripe billing objects are immutable and append-only. You never edit a price, you
publish a new one and move its `lookup_key`. The tool follows that model. It
reconciles toward the catalog and never deletes: a rate change creates a new
price, transfers the `lookup_key` onto it, and archives the old one.

## What it manages

- **Plans**: flat monthly Deploy subscription fees (`plan.<key>`). Each plan-fee
  price carries `plan=<tier>` metadata, which the dashboard's Stripe webhook reads
  to detect the Deploy plan (`detectDeployPlan` in the dashboard); a reprice carries
  it onto the new price. A plan-fee price without it reads as "no Deploy plan", so
  it must always be set.
- **Meters and metered prices**: usage billing (`usage.<key>`), priced in sub-cent
  `unit_amount_decimal`.
- **API products**: the legacy licensed API tiers and add-ons, with quota
  metadata.
- **Webhook endpoints**: per environment, declared in [`webhooks.go`](./webhooks.go).

Identity is the `lookup_key` for prices, and `managed_by=unkey-pricing` plus a
`pricing_key` for products. The tool never keys off the Stripe-generated id.

## Usage

Run it through mise from anywhere in the repo:

```bash
mise run pricing plan                    # show the diff, write nothing (sandbox)
mise run pricing apply --env production    # make Stripe match the catalog
mise run pricing verify --env canary       # exit non-zero if Stripe drifts
mise run pricing export --env production    # print the dashboard env block
```

`--env` is `sandbox` (the default), `canary`, or `production`. `apply` shows the
plan first and does nothing if there are no changes; against `production` it asks
you to type the environment name to confirm. Pass `--yes` to skip the prompt in
automation.

For the development loop, run the tests directly from this directory:

```bash
cd tools/pricing
go test ./...
```

## Credentials

For sandbox and local dev, export your own key and you're done:

```bash
export STRIPE_SECRET_KEY="rk_test_..."
```

Without `STRIPE_SECRET_KEY`, the tool reads the key from AWS Secrets Manager
(the `api_key` field of the `unkey/stripe` secret) through the `aws` CLI. It picks
the per-account profile from `AWS_PROFILE`, or from a per-environment
`AWS_PROFILE_<ENV>` such as `AWS_PROFILE_PRODUCTION` (with our generated AWS config
these are named `unkey-<account>-<role>`). If the SSO session is stale it runs
`aws sso login --sso-session unkey` (the SSO session, not the per-account
profile), overridable with `AWS_SSO_SESSION`. `AWS_REGION` (default `us-east-1`)
and `STRIPE_SECRET_ID` (default `unkey/stripe`) cover accounts that differ.

Profiles are deployment-specific, so they live in a local, untracked `.env`
instead of in source. Copy [`.env.example`](./.env.example) to `.env` and fill it
in; the `mise run pricing` task loads it automatically, so `mise run pricing plan
--env production` works once it's set.

The tool refuses to run a live key against a non-production environment, or a test
key against production, so you can't point the wrong account at the wrong catalog.

To create the restricted Stripe key with least-privilege permissions, see
[`docs/stripe-api-key.md`](./docs/stripe-api-key.md).

## Changing a price

Edit the amount in [`catalog.go`](./catalog.go) and update the matching pin in
[`pricing_test.go`](./pricing_test.go) in the same change. That test is a
decimal-shift guard and runs with no network, so a fat-fingered amount fails
before it ever reaches Stripe. Then `mise run pricing apply`: the tool creates a
new immutable price, transfers the `lookup_key` onto it, and archives the old one.

New checkouts use the new price immediately. Existing subscriptions keep their old
price until the app repoints them, which is application work the tool does not do.

## Orphans

An orphan is an object in our namespace that the catalog no longer declares: a
`plan.`/`usage.` price, or a product tagged `managed_by=unkey-pricing`. The tool
reports orphans and never touches them. `plan` lists them, `verify` fails on them,
and an operator decides what to do. Hand-made and legacy objects outside our
namespace are ignored.

## Caveats

Stripe only returns a webhook signing secret when the endpoint is created. `apply`
prints it once at that moment; `verify` and `export` cannot read it back. The
existing hand-made production endpoint is adopted by its URL without recovering the
secret, so the live `STRIPE_WEBHOOK_SECRET` keeps working untouched.

Stripe cannot delete a meter, only deactivate it, so meter cleanup is not
automated. A leftover meter shows up as an orphaned `usage.<key>` price, which is
the signal to clean it up by hand.
