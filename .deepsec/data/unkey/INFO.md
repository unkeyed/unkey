# unkey

## What this codebase does

Unkey is a multi-service API platform for API keys, RBAC, rate limiting, deployments, gateways, analytics, audit logs, billing, and customer portals.
Its production code spans Go services plus Next.js and TanStack web applications.

- `svc/api` exposes the public v2/v3 management API, portal endpoints, liveness, OpenAPI, and reference routes.
  Most operations use root-key or dashboard JWT authentication.
- `svc/frontline` is the internet-facing gateway and reverse proxy.
  Per-deployment policies optionally enforce key authentication, rate limits, firewall rules, and OpenAPI validation.
- `web/apps/dashboard` provides dashboard pages, tRPC procedures, Server Actions, auth callbacks, an API proxy, and provider webhooks.
  `web/apps/portal` exposes cookie-backed server functions for customer end users.
- `svc/ctrl` and `svc/vault` expose internal Connect RPC services.
  They manage deployments, infrastructure state, domains, applications, projects, and encryption.
- Restate provides durable workflows, event consumers, and cron execution under `svc/ctrl/worker`.
  The repository also ships an authenticated management CLI; no MCP or agent-tool server was found.

## Auth shape

Authentication and authorization are deliberately separate, with different primitives at each boundary.
Review registration and call chains before treating a handler as unauthenticated.

- The public API builds an ordered `auth.New` resolver chain for root keys and JWTs.
  `middleware.WithAuthentication` stores a normalized principal in `zen.Session`.
- API handlers retrieve that principal with `GetPrincipal`, scope data through `AuthorizedWorkspaceID`, and call `principal.Authorize` with RBAC or URN permission queries.
  Legacy root-key ownership and authorized workspaces can differ.
- Dashboard sessions come from WorkOS AuthKit or local auth, then become tRPC context.
  `protectedProcedure`, `workspaceProcedure`, role middleware, and explicit Server Action checks provide authorization.
- Portal code exchange is intentionally public and single-use.
  Subsequent portal operations use a separate hashed `portal_session` cookie resolver with session-derived identity and keyspace scopes.
- Ctrl and Vault RPC methods perform per-method constant-time shared-bearer checks; provider webhooks instead verify raw-body signatures.
  Restate handlers rely on Restate ingress visibility, auth keys, and internal network placement.

## Threat model

The highest-value assets are tenant data, API-key material, deployment secrets, portal tokens, billing state, and control-plane authority.
The main trust boundaries cross public HTTP, provider callbacks, proxy forwarding, and durable internal execution.

- Tenant isolation depends on organization-to-workspace mapping, `AuthorizedWorkspaceID`, workspace-bound database predicates, and resource-specific permission checks.
- Root keys, JWT signing material, portal cookies, environment ciphertext, Vault master keys, and one-time plaintext key responses require strict redaction and non-persistence.
- Frontline accepts arbitrary internet traffic and forwards it to customer workloads.
  Reserved headers, signed peer metadata, policy order, request replay, body capture, and custom key-location redaction are critical.
- GitHub, GitHub secret-scanning, Stripe, and WorkOS webhooks are public.
  Signature verification authenticates the provider, while idempotency and database binding establish replay safety and tenant ownership.
- Restate, cron, CDC, and internal RPC paths can perform infrastructure or billing mutations.
  Direct reachability, ingress-private bindings, retry policy, replay-safe I/O, and cluster network controls are security boundaries.

## Project-specific patterns to flag

- Trace route registration before judging authentication: Zen middleware arrays, tRPC procedure bases, Next proxy rules, and Server Action bodies determine whether a handler is public.
- Flag caller-supplied workspace scope, missing `principal.Authorize`, weak dashboard role middleware, or authorization errors that reveal whether another tenant's resource exists.
- Flag changes that trust client `X-Unkey-*` headers or trailers, weaken peer-metadata verification, reorder Frontline policies, broaden proxy retries, or log unredacted key locations and bodies.
- For webhooks and dashboard/API proxies, preserve raw-body signature checks, issuer/audience and organization claims, redirect/origin validation, tenant lookup, and provider-event idempotency.
- Restate external I/O must be journaled without putting Restate client/state calls inside `Run` closures.
  Audit cron bindings for bounded retries ending in `KillOnMaxAttempts`; key-refill and idle-preview handlers currently lack explicit binding policies.

## Known false-positives

- Liveness, health, OpenAPI/reference, ACME challenge, portal code exchange, and selected capability routes are intentionally unauthenticated.
- Frontline's catch-all can intentionally proxy without KeyAuth when a customer's deployment policy does not require authentication.
- API resolver chaining may accept a credential after an earlier compatible resolver rejects it; the later resolver must still fully verify it.
  Authorization failures may intentionally be masked as 404 to prevent enumeration.
- Webhook routes intentionally bypass user sessions and root keys.
  Stripe's `IgnoreAPIVersionMismatch` does not disable signature verification, and ignored event types deliberately return 200.
- Restate worker ports lack local bearer middleware because Restate and network placement are the intended boundary.
  Workspace and dashboard rate limiters also deliberately fail open on backend errors, which is an abuse-resilience concern rather than an auth bypass.
