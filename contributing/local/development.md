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

Independent updates run in parallel, with up to 16 active Tilt updates.
`seed-compile` builds the host CLI while MySQL starts. `seed` then writes the
fixtures without blocking unrelated updates. Both steps have separate timings
in Tilt.

MySQL and ClickHouse pull their pinned upstream images directly into the
cluster. Their SQL and initialization scripts mount from ConfigMaps, so Tilt
does not rebuild or push database images. A fresh node still downloads the
upstream images. Image pulls run in parallel.

Tilt manages Minikube's storage provisioner instead of the built-in addon.
The same provisioner uses the node's API endpoint directly, with its service
account and TLS verification. This avoids a 30-second startup timeout when
service networking is not ready. Existing PVCs and storage paths are unchanged.
Keep the built-in `storage-provisioner` addon disabled while using Tilt.

Minikube waits for the API server and kubelet before starting Tilt. Tilt installs
Cilium through Helm, then checks the Cilium agent, policy CRDs, node readiness,
and CoreDNS before applying network policies. Node readiness cannot gate Helm
installation because the node needs Cilium to become ready. Hubble readiness
does not block application startup. Orbs use bridge networking and wait for
node readiness during Minikube startup instead.

Private networking uses the cluster CoreDNS in both modes. The
`coredns-private-network` resource appends the `unkey.internal` server block
from `dev/k8s/coredns/private-network.Corefile` to the `coredns` ConfigMap.
It replaces only that marked block and leaves the default server block
unchanged. CoreDNS restarts only when the Corefile changes. The block resolves
only `unkey-connection-<32 hex>.<namespace>.unkey.internal` to the Service of
that name in that namespace, with a 5 second TTL. Other names in the zone
return NXDOMAIN. Tilt sets Krane's `private_network.resolver_ip` to the
`kube-system/kube-dns` ClusterIP. Customer pods use that nameserver with
`ndots:1` and no search domains.

Minikube 1.39.0 requires `--force` when CNI installation is disabled with the
containerd runtime. The Helm-managed Cilium path uses this flag so Tilt can
install Cilium. It also bypasses Minikube's driver and resource safety checks.
The startup script still rejects incompatible Kubernetes versions and CNI modes.

Unkey uses the Minikube profile `unkey` and registry `unkey-registry`.
The previous generic `minikube` profile and `ctlptl-registry` are left alone.
Their data is not moved into the `unkey` cluster.

An existing `unkey` profile is reused, not reapplied through `ctlptl`.
Changes to `dev/cluster.yaml` apply only when creating a cluster, because
`ctlptl apply` can delete a cluster when its configuration changes.
Kubernetes version or CNI changes require an explicit reset.

The toolchain pins Minikube 1.39.0, Kubernetes/kubectl 1.37.0, and Cilium 1.20.2.
Kubernetes 1.37.0 is built into Minikube's version list, avoiding a remote
version lookup during startup. Tilt manages actual Helm releases for Cilium
(including Hubble) and TopoLVM 15.5.4. Their configuration lives in
`dev/k8s/cilium/values.yaml` and `dev/k8s/topolvm/values.yaml`. Helm adopts the
previously applied local resources without deleting volumes. TopoLVM's chart
also owns the `topolvm-ssd` StorageClass. Orbs keep the matching
CiliumNetworkPolicy CRD without policy enforcement or a Cilium release.

Inspect the installed releases with:

```bash
mise exec -- helm list --all-namespaces
```

Startup rejects profiles with a different Kubernetes version or CNI mode.
The task does not request a Kubernetes upgrade or cluster deletion. Minikube
can reset an unhealthy control plane during recovery. Back up local data
before recovering an unhealthy cluster or explicitly resetting it.

Run `mise install --locked --yes` after pulling toolchain changes. The web
dependency task installs pinned Node and pnpm if needed and invokes pnpm
directly, without falling back to Corepack. Installation failures stop the task
without weakening lockfile or provenance checks. pnpm 11.28.5 publishes no
Intel macOS standalone binary; this pin supports Apple Silicon Macs, not Intel
Macs. Don't regenerate a lock entry pointing to `source-maps.tgz`.

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

To delete Unkey's local data and builds, exit Tilt and stop tests. Preview the
targets, then run the cleanup and type `unkey` to confirm:

```bash
mise run nuke --dry-run
mise run nuke
```

This removes the `unkey` cluster, owned dev and test containers, volumes,
networks, images, and the dedicated `unkey` Docker builder and its cache.
It also removes this checkout's build outputs and Next.js/Turbo caches.
Environment files and dependencies remain.

Containers, volumes, and networks use `dev.unkey.project=unkey`. Built images
use `dev.unkey.image=true`, so containers do not inherit resource ownership
from an image. Minikube and Buildx use dedicated names and owned state volumes.
Older Unkey Compose project labels and image source labels are also recognized.
Resources still used by other containers are kept.

Shared upstream images, Docker's shared build cache, other Minikube profiles,
and the legacy `minikube`/`ctlptl-registry` resources are not removed.

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

Set `UNKEY_JWT_AUDIENCE=api.unkey.local` in the dashboard `.env`. This overrides
the audience of fallback tokens used by `AUTH_PROVIDER=local`. Keep
`UNKEY_JWT_SECRET` matched to the local API's `auth.secrets` entry.

WorkOS sessions forward their access token unchanged. Both local API configs
verify these tokens with WorkOS JWKS and audience `api.unkey.local`. Configure
the development WorkOS environment's JWT template as follows:

```json
{
  "org": { "id": {{organization.id}} },
  "aud": ["api.unkey.local"]
}
```

The dashboard audience override does not change WorkOS tokens. Don't change
the production WorkOS template. Sign in again after updating the development
template to get a token with the new claims.

`mise run dashboard` reads `WORKOS_CLIENT_ID` from the dashboard `.env` and
passes it to the Docker API. For Kubernetes, also set `WORKOS_CLIENT_ID` in
`dev/.env.workos`; Tilt loads it into the `workos-credentials` secret. Restart
the local API after changing its config or environment.

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
