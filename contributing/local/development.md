---
title: "Local development"
description: "Set up, run, and test Unkey locally"
notion:
  owners:
    - andreas
  tags:
    - Onboarding
---

## Prerequisites

> **Warning:** We do not support Windows as a development environment. It might work, or it might not.

Unkey installs most tools and dependencies automatically. The only required preinstalled dependencies are:

- docker
- git

All other tools are managed via [mise](https://mise.en.dev/), which you'll install in the next step.

## Bootstrap

Clone the repository and install mise and other tools.

### 1. Clone the repository

```bash
git clone https://github.com/unkeyed/unkey
cd unkey
```

### 2. Install and set up mise

You can set up mise manually or use the install script. It pins mise to a specific version and SHA.

```bash
./dev/install-mise
```

### 3. Bootstrap local configuration

Run the bootstrap task to install the pinned toolchain, create local environment files, and configure the GitHub app.

```bash
mise run bootstrap
```

If GitHub rate limits `mise install`, provide a `GH_TOKEN` when you rerun the task:

```bash
GH_TOKEN=$(gh auth token) mise run bootstrap
```

If you only want to develop on the dashboard, run `mise run dashboard`.
Otherwise continue for a full dev setup.

## Run dev mode

Start the full development setup:

```bash
mise run dev
```

To run one service and its dependencies, pass its Tilt resource name:

```bash
mise run dev -- api
```

This starts the API, its databases, and required setup without compiling the
control plane or starting the dashboard. You can pass multiple resource names.
Selecting `krane` still includes its Cilium policies and storage dependencies.
To change the selection without restarting Tilt:

```bash
mise exec -- tilt args -- api
```

Tilt builds `@unkey/api` before starting the dashboard and rebuilds it when SDK
source files change. The dashboard picks up the compiled output without a
manual SDK build. SDK build failures appear in the `api-sdk` resource.

Minikube waits for the API server, kubelet, and node readiness. Tilt then checks
Cilium, its policy CRDs, and CoreDNS before applying network policies. It does
not wait for old application pods before starting Tilt, so Tilt can repair
failed deployments and reactivate TopoLVM volumes after a restart.

An existing Minikube profile is reused, not reapplied through `ctlptl`.
Changes to `dev/cluster.yaml` apply only when creating a cluster, because
`ctlptl apply` can delete a cluster when its configuration changes. To adopt
cluster configuration changes, explicitly reset it using the command below.

Local deployments use in-cluster BuildKit Jobs by default and don't require Depot credentials. To test the Depot backend, copy `dev/.env.depot.example` to `dev/.env.depot` and set both values to your Depot token before starting the environment.

You get:

- Tilt UI at `http://localhost:10350`
- Various services port-forwarded
- Dashboard at `http://localhost:3000`

## Local HTTPS with Frontline (optional)

Set up local TLS for `*.unkey.local`:

1. Configure local DNS:

```bash
./dev/setup-wildcard-dns.sh
```

2. Start the minikube tunnel in another terminal:

```bash
mise run tunnel
```

3. Open the local domain:

```bash
open https://app.unkey.local
```

Tilt generates trusted TLS certificates using mkcert and Frontline terminates TLS on port 443.

## Stop the development environment

Exit Tilt with Ctrl+C to stop local processes and port forwards, then stop
Minikube:

```bash
mise run down
```

This preserves the cluster, images, and local data. Run `mise run dev` to resume.
Do not use `tilt down` for a routine stop: it deletes Kubernetes resources,
including persistent volume claims.

To reset the cluster and delete its local data, exit Tilt and run:

```bash
mise exec -- minikube delete -p minikube
```

## Environment configuration

Dashboard environment variables live in `web/apps/dashboard/.env`. The
bootstrap task creates the file from `web/apps/dashboard/.env.example`.

### Local authentication

Set local auth mode in `web/apps/dashboard/.env`:

```plaintext
AUTH_PROVIDER="local"
```

### Optional services

WorkOS authentication:

Add WorkOS credentials to `web/apps/dashboard/.env`:

```plaintext
AUTH_PROVIDER="workos"
WORKOS_CLIENT_ID=<your client ID>
WORKOS_API_KEY=<your API key>
WORKOS_COOKIE_PASSWORD=<a unique secret with at least 32 characters>
```

Hosted AuthKit owns sign-in, sessions, and MFA in WorkOS mode. The dashboard
uses WorkOS User Profile and User Security for managed account settings. See
[Dashboard authentication](./dashboard-auth.md) for provider
configuration and release checks.


Stripe billing:

The dashboard subscription flow needs all four variables in
`web/apps/dashboard/.env`. If any is missing the dashboard treats Stripe as
unconfigured and billing calls fail.

The product ID lists come pre-filled in `.env.example` (the shared sandbox
catalog), so you only add two values:

- `STRIPE_SECRET_KEY` - a test-mode key (`sk_test_...`) from the shared sandbox.
- `STRIPE_WEBHOOK_SECRET` - the signing secret for forwarded webhook events.

Stripe webhook forwarding starts automatically when the Stripe CLI can obtain
a valid signing secret. For billing development, authenticate before starting:

```bash
mise exec -- stripe login
mise run dev
```

This runs `stripe listen` against both the dashboard
(`localhost:3000/api/webhooks/stripe`) and ctrl-api
(`localhost:7091/webhooks/stripe`). Tilt writes the shared `STRIPE_WEBHOOK_SECRET`
into both `web/apps/dashboard/.env` and `dev/.env.stripe` before starting its
consumers. A missing CLI, authentication failure, invalid secret, or a check
that takes more than five seconds skips forwarding without blocking startup.
Existing credentials stay unchanged and still load from `dev/.env.stripe`.
After logging in, reload Tilt to retry the check.

To set up a fresh Stripe sandbox (products, meters, prices), follow the catalog
guide in the infra repo: <a href="https://github.com/unkeyed/infra/blob/main/docs/services/stripe-billing.md" target="_blank">Stripe Billing</a>.

Deploy (control plane) billing is separate from the dashboard flow above and is
configured through `dev/` env files that Tilt loads into Kubernetes secrets.
Each is optional: a missing file disables that piece and never breaks startup.

- `dev/.env.stripe` (the `stripe-credentials` secret, read by both ctrl-api and
  the worker). Copy `dev/.env.stripe.example` and set:
  - `STRIPE_SECRET_KEY` - test-mode key for the hourly usage push and the
    month-end invoice finalize.
  - `STRIPE_WEBHOOK_SECRET` - the close webhook's signing secret, written
    automatically when the Stripe CLI is available and authenticated (as above).
  - `STRIPE_DEPLOY_*_LOOKUP_KEY` - the price lookup_keys `CancelDeploy` uses to
    find a subscription's Deploy items. Same handles the dashboard uses; empty
    disables cancel.
- The spend-cap budget alert emails need `dev/.env.workos` (`WORKOS_API_KEY`, to
  resolve an org's admin recipients) and `dev/.env.resend` (`RESEND_API_KEY`, to
  send). Without them the alerts only log; the suspend/resume enforcement is
  unaffected.

See [Deploy Billing](../../docs/engineering/architecture/services/control-plane/worker/workflows/deploy-billing.mdx) and [Deploy Spend Cap](../../docs/engineering/architecture/services/control-plane/worker/workflows/deploy-spend-cap.mdx).

### Feature flags

You don't need Vercel Flags setup to run dashboard code that imports `@/lib/flags`. When `FLAGS` is missing, the dashboard uses the noop adapter and resolves each flag to its declared `defaultValue`.

If you're adding flags, testing remote targeting rules, or using Vercel Toolbar overrides, ask Andreas for the dev values of `FLAGS` and `FLAGS_SECRET`. Add them to `web/apps/dashboard/.env`. They're stable, so you set them once and forget. See [Feature flags](../tooling/feature-flags.md) for the rest of the workflow.

## Seed local data

```bash
mise run unkey -- dev seed local
```

To fill the Limits and Usage settings pages, seed workspace usage after `seed local`. The command reserves compute, adds custom domains and log drains, and writes billable operations, active keys, and hourly compute usage for the current and previous month. `--fill` sets the used share of each limit. Use a value above 1 to see the over-limit state. A second run replaces the rows of the first run.

```bash
mise run unkey -- dev seed workspace-usage --fill 0.8
```

## Test locally

Run Go tests with Rask:

```bash
mise run test
```

Run a single Go test:

```bash
mise exec -- go test -run TestCacheName ./pkg/cache
```

Run TypeScript tests with pnpm:

```bash
mise exec -- pnpm --dir=web test
```

## Code quality

```bash
mise run fmt
mise run build
```

## Troubleshooting

### Failure: resource\_exhausted: too many requests

If you receive an error message similar to the example below, authenticate your terminal with buf. You can sign up for a free account at [buf.build](https://buf.build/home).

```bash
Failure: resource_exhausted: too many requestssh

Please see https://buf.build/docs/bsr/rate-limits/ for details about BSR rate limiting.
svc/frontline/proto/generate.go:4: running "go": exit status 1
Failure: resource_exhausted: too many requests

Please see https://buf.build/docs/bsr/rate-limits/ for details about BSR rate limiting.
svc/vault/proto/generate.go:3: running "go": exit status 1
mise run generate: command failed
```
