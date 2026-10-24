# DeepSec

This workspace reviews Unkey with DeepSeek V4 Flash through the Pi agent and
Vercel AI Gateway. An AI Gateway API key authenticates local and CI runs
without an interactive Vercel login.

## Local use

From the repository root, install the pinned workspace dependencies:

```bash
mise exec -- pnpm --dir .deepsec install --frozen-lockfile --ignore-scripts
```

Set `AI_GATEWAY_API_KEY` in your environment or in `.deepsec/.env.local`.
Never commit the key. The workspace uses `deepseek/deepseek-v4-flash` with
`xhigh` reasoning. Your Gateway account needs credits and access to that
model.

Run these commands from `.deepsec/`:

```bash
mise exec -- pnpm exec deepsec scan --project-id unkey
mise exec -- pnpm exec deepsec process --project-id unkey --concurrency 2
mise exec -- pnpm exec deepsec export --format md-dir --out ./findings
```

`scan` performs local pattern matching. `process` makes paid model calls.
Local runs retain state under `data/unkey/`.

## CI

Add `DEEPSEC_VERCEL_GATEWAY_KEY` as a Depot CI repository secret. The workflow
maps it to DeepSec's `AI_GATEWAY_API_KEY` environment variable. Use a dedicated
Gateway key and configure its spending limit in Vercel.

The `.depot/workflows/deepsec.yaml` workflow reviews same-repository PRs
that target `main`, including drafts. It reviews only files changed between
the PR head and its merge base with `main`. It loads the scanner workspace,
dependencies, and setup action from the PR's base commit. A separate
`review-source` checkout holds the PR head as scan input. Fork PRs are
excluded.

The scanner workspace must exist in the base branch before CI scans can run.
If it is missing, the job stops without loading configuration from the PR.
Scanner configuration changes take effect only after merge. This separation
prevents PR configuration from being loaded as scanner code, but does not
sandbox the scanner or its agent. Repository writers must still be trusted
to change CI workflows that have access to repository secrets.

CI installs `pnpm-lock.yaml` with `--frozen-lockfile`, then uses that
workspace's CLI so its runtime dependencies also match the lockfile.
The workspace overrides the Pi agent's vulnerable Undici pin to `8.12.3`.
Use the workspace CLI rather than the global mise executable to apply this fix.
It does not restore prior scan state. Each run reviews the selected changed
files again and updates one advisory PR comment.

The workflow skips Go test files, `testdata/`, and Markdown, then ranks the
remaining files: code in security-sensitive paths first, other code next,
and everything else last. The ranking pattern is `PRIORITY_PATTERN` in the
workflow. Only the first 40 ranked files are reviewed; the PR comment lists
the rest as skipped.

Selected files are split into chunks of 10. Each chunk runs as a separate
`deepsec process` with its own data directory, and all chunks run in
parallel. A chunk that fails or exceeds its 18-minute limit loses only its
own results; the PR comment lists its files as not fully reviewed. These
limits bound scan time, not dollar spend.

The PR comment ends with the run's estimated model cost and token usage.
DeepSec computes it from the Pi agent's bundled price table, which can lag
Gateway pricing. Batches that failed or were stopped are not counted, so
check Vercel AI Gateway usage for billed amounts.

## Files to keep

- `deepsec.config.ts`: model, Gateway route, and project registration
- `generated-matchers.ts`: custom matchers from the local setup
- `data/unkey/INFO.md`: repository context supplied to investigations
- `package.json`, `pnpm-workspace.yaml`, and `pnpm-lock.yaml`: pinned dependencies

Credentials, scan results, setup logs, and machine-specific project state
are ignored. Do not add them to Git.
