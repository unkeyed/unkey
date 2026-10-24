# DeepSec

This workspace reviews Unkey with Codex through Vercel AI Gateway.
An AI Gateway API key authenticates local and CI runs without an interactive
Vercel login or a local Codex subscription.

## Local use

From the repository root, install the pinned workspace dependencies:

```bash
mise exec -- npm ci --prefix .deepsec --ignore-scripts
```

Set `AI_GATEWAY_API_KEY` in your environment or in `.deepsec/.env.local`.
Never commit the key. The workspace uses DeepSec's default Codex model
with `xhigh` reasoning. Your Gateway account needs credits and access to
that model.

Run these commands from `.deepsec/`:

```bash
mise exec -- deepsec scan --project-id unkey
mise exec -- deepsec process --project-id unkey --concurrency 2
mise exec -- deepsec export --format md-dir --out ./findings
```

`scan` performs local pattern matching. `process` makes paid model calls.
Local runs retain state under `data/unkey/`.

## CI

Add `DEEPSEC_VERCEL_GATEWAY_KEY` as a Depot CI repository secret. The workflow
maps it to DeepSec's `AI_GATEWAY_API_KEY` environment variable. Use a dedicated
Gateway key and configure its spending limit in Vercel.

The `.depot/workflows/deepsec.yaml` workflow reviews same-repository PRs,
including drafts. It loads this
workspace from the PR's base commit and reviews a separate checkout of the
PR head. Configuration changes take effect after merge.

CI installs the supplied `package-lock.json` with `npm ci`, then uses that
workspace's CLI so its runtime dependencies also match the lockfile.
It does not restore prior scan state. Each run reviews the selected changed
files again and updates one advisory PR comment. Scan time is limited to
20 minutes; this is not a dollar spending cap.

## Files to keep

- `deepsec.config.ts`: model, Gateway route, and project registration
- `generated-matchers.ts`: custom matchers from the local setup
- `data/unkey/INFO.md`: repository context supplied to investigations
- `package.json` and `package-lock.json`: pinned workspace dependencies

Credentials, scan results, setup logs, and machine-specific project state
are ignored. Do not add them to Git.
