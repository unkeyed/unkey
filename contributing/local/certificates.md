---
title: "Custom domains and certificates"
description: "Verify a custom domain and issue its ACME certificate on the local cluster"
notion:
  owners:
    - andreas
  tags:
    - Local
---

The custom domain workflows in `ctrl-worker` talk to the public internet: domain
verification reads public DNS, and certificate issuance asks an ACME CA to fetch
a challenge over HTTP. This page covers what the local cluster needs for both to
run end to end.

> **Warning:** Use a throwaway domain you control. Never use an `unkey.com`
> hostname.

## How Restate runs locally

Tilt runs Restate from `dev/k8s/manifests/restate.yaml` and port-forwards both of
its ports:

| Port | Purpose |
| --- | --- |
| `8080` | Ingress. Invoke handlers here. |
| `9070` | Admin API and UI. |

`ctrl-worker` registers itself on startup. Its config sets
`[restate] register_as = "http://ctrl-worker:9080"`, and `svc/ctrl/worker/run.go`
posts that URL to the admin API, so every restart picks up new handlers without a
manual registration step.

## Enable ACME

The `[acme]` block in `dev/k8s/manifests/ctrl-worker.yaml` ships disabled and
already points at Let's Encrypt staging, so local runs cannot burn production
rate limits:

```toml
[acme]
enabled = true
email_domain = "example.test"
directory_url = "https://acme-staging-v02.api.letsencrypt.org/directory"
```

Set `enabled = true`. The HTTP-01 provider is only created when ACME is enabled,
so with it off, `ProcessChallenge` fails with `HTTP provider required for
certificate`.

Set `email_domain` to something disposable. Every certificate uses one shared
ACME account whose `acme_users.workspace_id` is `acme`, so the registered address
is `acme@<email_domain>`. The default is `unkey.com`.

Wildcard domains use DNS-01, which needs `[acme.route53]` with real AWS
credentials. Without it, wildcard issuance fails, so test with a regular
hostname.

## Make the domain publicly reachable

Neither check can be faked locally:

- **Verification** resolves the domain against `1.1.1.1` (`pkg/dns`). It passes
  when the domain's CNAME equals its `target_cname`, or when
  `_unkey.<domain>` has a TXT record `unkey-domain-verify=<verification_token>`.
- **HTTP-01** has the CA fetch
  `http://<domain>/.well-known/acme-challenge/<token>` on port 80. Frontline
  answers that path, so port 80 on a public address must reach the local
  frontline.

`mise run tunnel` is not enough: it only forwards ports on your machine for
`*.unkey.local`. You need a public tunnel, for example ngrok or cloudflared,
that forwards to frontline's HTTP port. Tilt port-forwards it to
`localhost:9080`; with `mise run tunnel` running it is also on `localhost:80`.
The tunnel must preserve the `Host` header, because frontline looks up the
custom domain by host.

`ctrl-api` builds each domain's `target_cname` as `<random>.<cname_domain>`,
and both manifests set `cname_domain = "unkey.local"`, which no public resolver
can answer. Change `cname_domain` in `dev/k8s/manifests/ctrl-api.yaml` (and
`ctrl-worker.yaml`, which requires the key) to a domain you control, then add a
wildcard record `*.<cname_domain>` that routes to your tunnel.

If that wildcard is itself a CNAME, the CNAME check fails: Go's resolver follows
the chain and returns its final name, not `target_cname`. Add the TXT record
from the domain's `verification_token` instead, which verifies a subdomain on
its own.

## Run the workflows

Add the custom domain through the dashboard or `POST /v2/domains.createDomain`
on the local API. `ctrl-api` starts `VerifyDomain`, which rechecks DNS every
minute for up to 24 hours. On success it inserts an `acme_challenges` row and
sends `ProcessChallenge` itself.

To drive a step by hand, call Restate ingress. Both services are virtual
objects, so the path is `/<service>/<key>/<handler>`, and appending `/send`
returns without waiting for the result.

| Handler | Key | Body |
| --- | --- | --- |
| `hydra.v1.CustomDomainService/VerifyDomain` | domain ID (`custom_domains.id`) | `{}` |
| `hydra.v1.CustomDomainService/RetryVerification` | domain ID | `{}` |
| `hydra.v1.PortalDomainService/VerifyDomain` | portal domain ID (`portal_domains.id`) | `{}` |
| `hydra.v1.PortalDomainService/RetryVerification` | portal domain ID | `{}` |
| `hydra.v1.CertificateService/ProcessChallenge` | domain name | `workspace_id`, `domain` |
| `hydra.v1.CertificateService/RenewExpiringCertificates` | `global` | `{}` |

Restart verification for a domain, for example after fixing DNS. This resets
the row to `pending` and runs `VerifyDomain` again:

```bash
curl -X POST "http://localhost:8080/hydra.v1.CustomDomainService/<domain_id>/RetryVerification/send" \
  -H 'content-type: application/json' \
  -d '{}'
```

Issue a certificate for one domain. The domain is the key and is also the body
field `ProcessChallenge` looks up among verified rows in `custom_domains` and
`portal_domains`:

```bash
curl -X POST "http://localhost:8080/hydra.v1.CertificateService/<domain>/ProcessChallenge/send" \
  -H 'content-type: application/json' \
  -d '{"workspace_id": "<workspace_id>", "domain": "<domain>"}'
```

Run the renewal sweep, which sends `ProcessChallenge` for every challenge that is
`waiting` or expires within 30 days. Tilt runs no cron for it locally:

```bash
curl -X POST "http://localhost:8080/hydra.v1.CertificateService/global/RenewExpiringCertificates/send" \
  -H 'content-type: application/json' \
  -d '{}'
```

The Restate UI on port `9070` shows each invocation's journal, retries, and
errors, and can invoke the same handlers.

## Portal domains

Portal domains live in `portal_domains` and verify under the same DNS rules,
but every verified hostname routes to one shared portal app rather than to the
tenant's own environment. No portal app is seeded locally, so deploy any app to
stand in for it, then find its production environment ID:

```sql
SELECT e.id, p.slug AS project, a.slug AS app
FROM environments e
JOIN apps a ON a.id = e.app_id
JOIN projects p ON p.id = e.project_id
WHERE e.slug = 'production';
```

Set that ID as `[portal] environment_id` in both
`dev/k8s/manifests/ctrl-api.yaml` and `dev/k8s/manifests/ctrl-worker.yaml`.
`ctrl-api` rejects `AddPortalDomain` with `FailedPrecondition` until it and
`portal_cname_domain` are set, and `ctrl-worker` needs it to create the route.

```toml
# ctrl-api.yaml
portal_cname_domain = "portal.unkey.local"

[portal]
environment_id = "<environment_id>"
```

```toml
# ctrl-worker.yaml
[portal]
environment_id = "<environment_id>"
```

Each portal domain's `target_cname` is `<random>.<portal_cname_domain>`. The
default `portal.unkey.local` has the same problem as `cname_domain`: no public
resolver answers it. Point `portal_cname_domain` at a domain you control with a
wildcard `*.<portal_cname_domain>` record to your tunnel, or verify with the TXT
record instead.

Attach a hostname with `POST /v2/portal.createDomain` on the local API, using a
root key that can update the portal. The response carries the `domainId` and
the DNS records to create:

```bash
curl -X POST "http://localhost:7070/v2/portal.createDomain" \
  -H "Authorization: Bearer <root_key>" \
  -H 'content-type: application/json' \
  -d '{"portal": "<portal_id>", "domain": "<domain>"}'
```

`portal.getDomain`, `portal.listDomains`, `portal.verifyDomain`, and
`portal.deleteDomain` cover the rest of the lifecycle. On success the workflow
inserts a `frontline_routes` row on the portal environment with `sticky = 'live'`,
so the hostname follows that app's promotions, and sends `ProcessChallenge`
under the tenant's workspace.

The verification object is `hydra.v1.PortalDomainService`, keyed by portal
domain ID (`portal_domains.id`), with the same `VerifyDomain` and
`RetryVerification` handlers as `CustomDomainService`:

```bash
curl -X POST "http://localhost:8080/hydra.v1.PortalDomainService/<portal_domain_id>/RetryVerification/send" \
  -H 'content-type: application/json' \
  -d '{}'
```

## Staging certificates

Let's Encrypt staging certificates chain to an untrusted root. Browsers and
strict TLS clients reject them, but the issuance flow is the real one.

The ACME account is stored once and reused: on later runs, `GetOrCreateUser` in
`svc/ctrl/services/acme/user.go` loads the stored key and `registration_uri`
instead of registering again. An account belongs to one directory, so after
changing `directory_url`, delete the row before the next issuance:

```sql
DELETE FROM acme_users WHERE workspace_id = 'acme';
```

Local MySQL is on `127.0.0.1:3306`, user `unkey`, password `password`, database
`unkey`.
