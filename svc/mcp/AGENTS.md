# MCP spike

This tree is a staging spike for WorkOS AuthKit access tokens. It is not a
product service. Do not wire it into `build/`, local Kubernetes, billing, or
`svc/api`.

The process reads `AUTHKIT_DOMAIN`, `PUBLIC_BASE_URL`, `LISTEN_ADDR`, and
`MCP_SPIKE_LOG_CLAIMS` only. It must not gain a database, a secret, or a
token log. [SPIKE.md](SPIKE.md) is the staging runbook.
[PHASE0-FINDINGS.md](PHASE0-FINDINGS.md) is the code analysis for the later
API change, which is not implemented here.
