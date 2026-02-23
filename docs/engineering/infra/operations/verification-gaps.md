# Verification gaps

This file tracks documentation statements that could not be verified against code or infrastructure configuration.

## Vault

- `docs/engineering/architecture/services/vault/api.mdx`: TODO to document request and response fields with examples.
- `docs/engineering/architecture/services/vault/key-lifecycle.mdx`: TODO to document explicit DEK rotation triggers and manual procedure.
- `docs/engineering/architecture/services/vault/storage.mdx`: TODOs for object payload example, HA constraints, and S3 health checks.
- `docs/engineering/architecture/services/vault/configuration.mdx`: TODO to document the re-encryption procedure.
- `docs/engineering/architecture/services/vault/get-vault.mdx`: TODO to document local config location and secret setup.
- `docs/engineering/architecture/services/vault/plugins.mdx`: TODO to document extension points if added.

## Control plane

- `docs/engineering/architecture/services/control-plane/worker/workflows/certificates.mdx`: TODO to document challenge routing, HTTP-01 provider details, and renewal scheduling intervals.
- `docs/engineering/architecture/services/control-plane/worker/workflows/github-app.mdx`: TODO to document webhook event types that trigger deployments.

## Operations

- `docs/engineering/infra/operations/index.mdx`: content pending for on-call and maintenance.

## Architecture

- `docs/engineering/architecture/services/index.mdx`: content pending for service architecture docs.
- `docs/engineering/architecture/workflows/index.mdx`: content pending for workflow architecture docs.

## Reference

- `docs/engineering/infra/operations/reference/index.mdx`: content pending for glossary.
