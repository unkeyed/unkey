# Docker-image apps as a first-class source (ENG-3006)

The backend should model GitHub repositories and Docker images as explicit app
sources. Docker deployments must resolve mutable tags to immutable digests so
rollbacks are stable and a later auto-update worker can compare artifacts
without another schema migration.

This project covers the backend foundation and the existing dashboard new-app
flow. The dashboard must create GitHub and Docker apps with an explicit source
instead of creating a legacy app before the user chooses. It does not include
an auto-update worker, private-registry credentials, or source switching.
Minimal changes to existing server-side GitHub connection writers are included
when they are required to preserve the new database invariants.

## Current behavior

The deployment pipeline already accepts prebuilt images. `CreateDeployment`
prioritizes an explicit `docker_image`, and Hydra carries Docker and Git sources
as a `oneof`
([create_deployment.go](svc/ctrl/services/deployment/create_deployment.go),
[deploy.proto](svc/ctrl/proto/hydra/v1/deploy.proto)). Krane then passes the
stored image string to Kubernetes unchanged.

The missing pieces are source ownership and artifact identity:

- An app is treated as Git-backed when a `github_repo_connections` row exists.
  There is no declared source type.
- Docker apps have no stored default image. A deploy without an explicit image
  reuses the live deployment's image, which cannot work for the first deploy.
- `apps.default_branch` and `app_build_settings` put Git-only configuration on
  every app.
- Docker tags are stored and deployed unchanged. Recreating a deployment that
  used a mutable tag can run different bytes.
- Deployment provenance is inferred from Git metadata or the app's current
  repository connection. Image redeploys also copy stale Git metadata from the
  previous deployment.

## Target model

```diagram
┌──────────────────────── apps ────────────────────────┐
│ id, workspace_id, project_id, name, slug             │
│ source_type: enum('legacy', 'github', 'docker_image')│
│ current_deployment_id, ...                           │
└──────────┬───────────────────────────┬───────────────┘
           │ source_type=github        │ source_type=docker_image
           ▼                           ▼
┌─ github_repo_connections ─┐   ┌─ app_docker_sources ─┐
│ installation_id           │   │ image_reference      │
│ repository_id/full_name   │   │ nginx:stable         │
│ default_branch            │   └───────────┬──────────┘
│ + app_build_settings rows │               │ resolve
└───────────────────────────┘               ▼
                                nginx@sha256:<digest>
                                            │
                                            ▼
┌────────────────────── deployments ───────────────────┐
│ source: enum('unknown', 'git_build', 'docker_image') │
│ requested_image: nginx:stable                        │
│ image: nginx@sha256:<digest>                         │
│ git_*: populated only for git_build                  │
└──────────────────────────────────────────────────────┘
```

`legacy` is a rollout state for apps created by callers that do not yet send a
source. It preserves the current connection-row inference until those callers
are upgraded. New source-aware callers create only `github` or `docker_image`
apps. A later migration can remove `legacy` after every writer sends a source.

The Docker source stores desired configuration. A deployment stores both what
was requested and the immutable artifact that ran:

- `app_docker_sources.image_reference`: the app default, such as
  `ghcr.io/acme/api:stable`.
- `deployments.requested_image`: the reference selected for this deployment,
  including one-off overrides.
- `deployments.image`: the digest reference passed to Kubernetes, such as
  `ghcr.io/acme/api@sha256:...`.

An auto-update worker can later resolve `image_reference` and compare it with
the current deployment's `image`. Update policy does not belong in this project;
maintenance windows and enablement may be environment-specific.

## Database changes

### Apps and source configuration

1. Add `apps.source_type` with `legacy`, `github`, and `docker_image` values.
   Existing apps start as `legacy` so current GitHub connect and disconnect
   writers retain their behavior during a backend-only rollout.
2. Add `app_docker_sources` with:
   - `pk`
   - `workspace_id`
   - `app_id`, unique
   - `image_reference varchar(512)`
   - `created_at`, `updated_at`
3. New Docker apps must create their app and Docker source row in the existing
   `CreateApp` transaction. A `docker_image` app without a source row is invalid
   for new writes.
4. App deletion must remove `app_docker_sources` because the schemas do not use
   foreign-key cascades.

### Git-only configuration

1. Add nullable `default_branch` to `github_repo_connections` and backfill it
   from `apps.default_branch` for existing connections.
2. Ctrl reads the connection's branch first and falls back to
   `apps.default_branch` while old writers remain deployed.
3. Repository connection writers store the branch on
   `github_repo_connections`. The old app column is removed only after every
   writer and reader has moved.
4. GitHub webhook deployment-context queries use the connection branch first
   and retain the app-column fallback during rollout.
5. Split `FindAppWithSettings` into source-neutral app/runtime loading and
   Git-only build-settings loading. Docker deploys must not require an
   `app_build_settings` row.
6. `CreateApp` creates build-settings rows only for GitHub and legacy apps.
   Existing Docker-app rows can remain until a later data cleanup.

The compatibility columns and dead rows can be removed in a trailing migration,
but the code ownership moves in this project.

### Deployment provenance and image identity

1. Add `deployments.source` as
   `enum('unknown', 'git_build', 'docker_image') NOT NULL DEFAULT 'unknown'`.
2. Add nullable `deployments.requested_image varchar(512)`.
3. Widen `deployments.image` from 256 to 512 characters.
4. Set `source` when inserting every new deployment. Set `requested_image` at
   insert time for Docker deployments; leave `image` empty until the workflow
   resolves the digest.
5. Docker deployments leave every `git_*` field NULL, including explicit image
   deployments whose caller also supplied incidental Git metadata.

Historical provenance cannot be reconstructed perfectly. Failed Git builds may
have no `build_id`, while old image redeploys can contain copied Git fields.
Backfill only rows whose source is certain:

- `git_build` where durable build records prove a Git build.
- `docker_image` where durable records prove a prebuilt-image deployment.
- `unknown` for everything ambiguous.

Do not null historical Git metadata or label every row without a `build_id` as
Docker. Readers keep their existing compatibility inference for `unknown` rows.

## API changes

### CreateApp

Extend `CreateAppRequest` with both source choices:

```protobuf
oneof source {
  GitHubSource github = 6;
  DockerImageSource docker_image = 7;
}

message GitHubSource {}

message DockerImageSource {
  string image_reference = 1;
}
```

Behavior:

- GitHub source creates the app, default environments, runtime settings, and
  build settings. Repository connection remains a separate operation because
  the GitHub App redirect cannot happen inside `CreateApp`.
- Docker source validates the reference and creates the app, default
  environments, runtime settings, and `app_docker_sources` row. It does not
  create build settings.
- An omitted source creates a `legacy` app during the compatibility window. It
  preserves the current behavior for callers that have not adopted the proto
  fields yet.

### Docker source updates

Add an intent-specific RPC for changing a Docker app's default reference. It
must validate workspace ownership, validate the reference, update
`app_docker_sources`, and write an audit log.

This RPC does not deploy automatically and does not switch a GitHub app to
Docker. General `UpdateApp` and source switching are out of scope.

### Dashboard app creation

The new-app wizard collects the app name and slug before source selection, but
must delay `CreateApp` until the user chooses GitHub or submits a Docker image
reference. The request then sends the matching source variant. Docker app
creation stores that image as the app default before the first deployment is
created.

The app list exposes the declared source and Docker image reference so the
dashboard can show Docker apps without inferring their type from a missing
GitHub connection. Docker apps show the configured image instead of Git-only
build settings. The image setting calls the intent-specific update RPC and
refreshes app data after saving. Updating the desired reference does not create
a deployment or replace the currently running artifact.

## Deployment resolution

`CreateDeployment` resolves the source in this order:

1. An explicit Docker image override creates a Docker deployment without
   changing the app's default source.
2. An explicit Git commit requires a GitHub or compatible legacy app with a
   repository connection.
3. A GitHub app requires a repository connection and resolves its commit from
   GitHub.
4. A Docker app reads `app_docker_sources.image_reference`.
5. A legacy app preserves the current behavior: use Git when a connection
   exists, otherwise reuse the live deployment image.

The live-image fallback remains only for legacy compatibility. New Docker apps
always have an authoritative source row, so their first deployment needs no
live predecessor.

The selected source controls deployment metadata. Docker selection clears the
commit fields before insertion; Git selection populates them. The deployment's
source is never inferred from the app after insertion.

## Digest resolution

Digest resolution happens inside the durable deployment workflow:

1. `CreateDeployment` inserts `source=docker_image` and the requested reference.
2. The workflow parses the reference. A digest reference is already immutable;
   a tagged reference is resolved through the OCI registry API.
3. For a multi-platform image, store the OCI image-index digest. Kubernetes can
   select the platform-specific manifest while the index remains immutable.
4. Persist the normalized `repository@sha256:...` reference in
   `deployments.image` before applying the deployment.
5. Pass only the resolved digest reference to Krane.

Resolution uses a maintained OCI registry client. Public images use anonymous
Bearer-token flows. Images in Unkey's build registry reuse the control plane's
existing registry credentials so legacy redeploys can still resolve their
per-deployment tags. User-managed private-registry credentials remain out of
scope, so audit other private images that Kubernetes can pull with cluster
credentials but ctrl cannot resolve before rollout.

Registry lookup failures are deployment failures, not synchronous
`CreateDeployment` failures. Running resolution inside the workflow gives them
durable retries and keeps a failed deployment record with its requested image.

Image references are validated when written to an app source or deployment.
They must parse and include an explicit tag or digest. Validation does not probe
the registry.

## Provenance consumers

All backend behavior that reconstructs a deployment must use
`deployments.source`:

- operational rebuilds
- approval authorization, which currently reconstructs only Git sources
- API deployment mapping
- redeploy endpoints outside the dashboard UI
- rollback and wake paths that reapply `deployments.image`

`git_build` rebuilds use the stored commit and repository connection.
`docker_image` rebuilds reuse the stored resolved digest, not the app's current
tag. `unknown` deployments retain the legacy inference path.

The existing public API image field continues to report the requested image
when available, because that is the reference the caller supplied. Execution
and rebuild paths use the resolved `deployments.image` digest. Exposing both
values as separate public fields can be added later without changing storage.

## Tests

Add focused integration coverage for:

- creating GitHub, Docker, and legacy apps
- Docker app creation without build-settings rows
- first Docker deployment using the stored default reference
- explicit Docker override on a GitHub app
- tagged reference resolving to and persisting a digest
- digest input bypassing tag resolution
- multi-platform reference pinning the image-index digest
- registry resolution failure marking the deployment failed
- Docker deployments persisting no Git metadata
- Git, Docker, and unknown rebuild behavior
- app deletion removing the Docker source
- default-branch compatibility reads during rollout
- dashboard app creation sending GitHub and Docker source variants

Use a local fake OCI registry in tests so digest behavior does not depend on an
external registry.

## Delivery sequence

1. Add source tables and additive compatibility columns, update both MySQL and
   Drizzle schemas, then regenerate database and protobuf code.
2. Split source-neutral and Git-only settings loading. Move default-branch
   reads and writes behind the compatibility fallback.
3. Extend `CreateApp`, add Docker-source updates, source-aware app deletion, and
   image-reference validation.
4. Add deployment source/requested-image persistence and remove Git metadata
   copying from Docker deployments.
5. Add workflow digest resolution and persist the resolved image before Krane
   applies it.
6. Change rebuild and API mapping to use deployment provenance.
7. Update the dashboard new-app flow and app presentation to use the declared
   source.
8. Run focused ctrl tests, dashboard type checks, service build, generated-code
   checks, and migration verification.
9. Remove compatibility columns and legacy rows only after downstream callers
   have adopted explicit sources.

Expected implementation effort is roughly seven to ten engineer-days. The
largest uncertainty is registry compatibility, especially whether existing
deployments rely on pull credentials that are available to Kubernetes but not
ctrl.

## Out of scope

- Automatic image-update polling or scheduling
- Maintenance-window policy
- User-managed private-registry credentials
- Registry browsing or tag pickers
- App source switching or Git-to-image ejection
- Reclassifying ambiguous historical deployments
