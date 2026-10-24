# Agent setup

This is a deepsec scanning workspace. Read `README.md` for Vercel AI Gateway
authentication and `data/unkey/INFO.md` for the supplied repository context.

## Common tasks

- **Install dependencies**: run `mise exec -- npm ci --ignore-scripts` here.
- **Scan or process**: run `mise exec -- deepsec scan` or
  `mise exec -- deepsec process`. Processing makes paid AI Gateway calls.
- **Add a new project**: run `mise exec -- deepsec init-project <root>`; it
  scaffolds `data/<id>/` and prints/writes the setup prompt for the
  new project.
- **Write a custom matcher** (only after a real true-positive shows you
  a pattern worth keeping): read
  `node_modules/deepsec/dist/docs/writing-matchers.md`.

## Reference

The deepsec skill is at `node_modules/deepsec/SKILL.md` (after
`mise exec -- npm ci --ignore-scripts`). The full docs ship at
`node_modules/deepsec/dist/docs/`.
