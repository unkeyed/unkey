# Vercel provider reference fixture

Vercel doesn't document its flags datafile format. These files check that the
Go provider evaluates flags the same way as Vercel's own JavaScript SDK.

| File | Purpose |
| --- | --- |
| `direct-targets.json` | Sample datafile using only the supported subset: boolean variants, direct targets, a fallthrough variant, and a paused flag. |
| `expected.json` | Results of `evaluate()` from `@vercel/flags-core` for each case in `generate-expected.mjs`. `TestReferenceFixture` requires the Go provider to return the same value and reason. |
| `generate-expected.mjs` | Regenerates `expected.json` with the `@vercel/flags-core` version that the dashboard installs through `@flags-sdk/vercel` (1.4.0 at the time of writing). |

Regenerate `expected.json` after you change `direct-targets.json` or the cases,
or after you upgrade `@flags-sdk/vercel` in the dashboard:

```bash
mise run install-web
mise exec -- node pkg/featureflag/vercel/testdata/generate-expected.mjs > pkg/featureflag/vercel/testdata/expected.json
mise exec -- rask ./pkg/featureflag/vercel
```

A diff in `expected.json` after an upgrade means Vercel changed how it
evaluates flags. Fix the provider, not the fixture.
