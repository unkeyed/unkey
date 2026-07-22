# ENG-3070 local guardrail verification

Date: 2026-07-22  
Branch: `chronark/ratelimit-api`  
Commit: `367fb026da08074cce488f9dd080e0f52fed3a1b`  
Target: user-authorized local Minikube environment only

## Conclusion

The tested authorization and tenant-isolation controls held. No request returned
rows from another workspace, key space, or rate-limit namespace. The API rejected
unversioned analytics names, direct `default.*` physical names, system tables,
information-schema tables, table functions, table-backed set operands, and
`SETTINGS` clauses.

The verification found one confirmed integration issue and several narrower
correctness or defense-in-depth issues:

1. The HTTP ClickHouse driver and the generated readonly profile disagree about
   `max_execution_time`. An untouched generated profile can reject valid API
   requests before query execution.
2. The SQL parser accepts more than one statement but executes only the first.
   The later statement was ignored, not executed. The API must reject this input
   explicitly.
3. A CTE named after a public table alias is resolved as that public alias. The
   query reads the filtered physical analytics table instead of the CTE result.
4. Direct workspace ClickHouse users can read limited `system.tables` and
   `system.columns` metadata for the tables they can access. The API path still
   rejects all tested `system.*` sources.
5. ClickHouse execution errors expose rewritten physical table names and the
   caller's workspace predicate in public error details.

The first issue was an availability and integration blocker for the HTTP
transport. The other findings did not return cross-scope data in this run.
All confirmed findings, including the function-spelling mismatch described
below, now have fixes and passing regression coverage.

## Scope and safety

Ten Orca-managed workers ran separate, bounded assignments. Every worker used a
fresh agent terminal in the active worktree and completed through Orca's tracked
`worker_done` lifecycle. The assignments covered:

1. Public aliases and physical-table rejection.
2. Verification workspace and key-space isolation.
3. Rate-limit namespace authorization.
4. Multi-source filter coverage.
5. CTE, subquery, `UNION`, and `EXCEPT` traversal.
6. Alternate sources, table-backed sets, and `SETTINGS` rejection.
7. Query and result resource bounds.
8. Parser syntax, quoting, and comments.
9. Authentication, RBAC, validation, and error responses.
10. Direct ClickHouse grants, row policies, readonly mode, and profiles.

Workers could issue at most 40 requests that reached the assigned local target.
They used read-only SQL, did not edit the repository, and did not mutate cluster
objects, database rows, grants, users, policies, or fixtures. Credential values
and authorization headers were not recorded.

Multi-source syntax was exercised only to test filter placement. This report
does not define or promise public SQL syntax support.

## Environment and fixtures

The checks used these local endpoints:

| Component | Local endpoint |
| --- | --- |
| API | `http://127.0.0.1:7070` |
| Control API | `http://127.0.0.1:7091` |
| Restate ingress | `http://127.0.0.1:8080` |
| Vault | `http://127.0.0.1:8060` |
| ClickHouse HTTP | `http://127.0.0.1:8123` |
| ClickHouse native | `127.0.0.1:9000` |

The local API used `http://clickhouse:8123/default` as its workspace analytics
base URL. All required Minikube components were ready before worker dispatch.

Four disposable principals covered wildcard, scoped, missing-permission, and
cross-workspace behavior:

| Principal | Workspace | Analytics scope |
| --- | --- | --- |
| `alpha-wild` | `ws_rtalpha` | Wildcard verification and rate-limit analytics |
| `alpha-scoped` | `ws_rtalpha` | One API and one rate-limit namespace |
| `alpha-none` | `ws_rtalpha` | No analytics permission |
| `beta-wild` | `ws_rtbeta` | Wildcard verification and rate-limit analytics |

The sentinel data had intentionally different counts:

| Data set | Alpha primary | Alpha secondary | Beta |
| --- | ---: | ---: | ---: |
| Verification rows | 2 | 3 | 5 |
| Rate-limit rows | 2 | 3 | 5 |

The differing labels and counts made accidental cross-scope results visible in
aggregate responses without selecting secrets.

## Worker results

| Worker | Assignment | Result | Report |
| ---: | --- | --- | --- |
| 1 | Public aliases and physical tables | Physical, stale, and system names rejected. Public-alias CTE collision changed semantics. | `agent-01.md` |
| 2 | Verification isolation | Passed across wildcard, scoped, no-permission, and beta principals. | `agent-02.md` |
| 3 | Namespace authorization | Passed positive-literal bounds, deduplication, permission checks, and workspace isolation. | `agent-03.md` |
| 4 | Multi-source filtering | No cross-scope sentinel rows in any executable case. One parser-rejected syntax form was inconclusive. | `agent-04.md` |
| 5 | Query-tree traversal | CTE, subquery, `UNION`, and `EXCEPT` cases retained filters. | `agent-05.md` |
| 6 | Forbidden sources and modifiers | All tested alternate sources, set operands, and `SETTINGS` forms were rejected. | `agent-06.md` |
| 7 | Resource bounds | Parser bounds and setting rejection passed. Live byte and timeout overflow remained intentionally untested. | `agent-07.md` |
| 8 | Parser normalization | Most cases passed. Multiple statements were silently truncated to the first, and `COUNTIF` validation disagreed with execution. | `agent-08.md` |
| 9 | Authentication and RBAC | All tested auth, scope, body, and endpoint cases passed. | `agent-09.md` |
| 10 | Direct ClickHouse controls | Grants, row policies, readonly mode, and resource limits passed. Limited system metadata remained readable. | `agent-10.md` |

The raw reports remain in the run's temporary working directory.

## Satisfied contracts

### Public table boundary

The API accepted the intended versioned aliases, including
`key_verifications_v1` and `ratelimits_v1`. It rejected:

- `key_verifications` and `ratelimits`.
- Direct `default.key_verifications_raw_v2` and
  `default.ratelimits_raw_v2` references.
- Database-qualified public-looking names such as
  `default.key_verifications_v1`.
- Quoted, commented, nested, and branch-local physical-table references.
- `system.*` and `information_schema.*` references through either API route.

This confirms the important distinction: public aliases are accepted API names,
while physical `default.*` tables are internal execution targets. The stale
unversioned verification name was not accepted in this branch.

### Workspace and resource isolation

The verification route returned these aggregate results:

- `alpha-wild`: alpha key spaces only, with counts 2 and 3.
- `alpha-scoped`: its allowed key space only, with count 3.
- `beta-wild`: the beta key space only, with count 5.
- `alpha-none`: HTTP 403.

The rate-limit route returned the matching namespace counts. It also enforced
one to 10 positive literal namespace IDs, rejected zero, negative, non-literal,
or excessive inputs, deduplicated repeated IDs, and rejected mixed allowed and
disallowed scoped IDs.

Caller predicates such as `OR 1 = 1` did not widen results. Explicit
cross-workspace predicates returned no rows through direct ClickHouse access.

### Query-tree filtering

Filters remained effective in the tested CTE, nested subquery, comma-source,
multi-source, `UNION`, and `EXCEPT` forms. No successful query returned beta
sentinels to an alpha principal or alpha sentinels to a beta principal.

One parenthesized multi-source syntax form failed parsing before execution. It
provided no isolation evidence and is recorded as inconclusive rather than a
pass.

### Forbidden alternate sources

The API rejected all tested forms of:

- System and information-schema sources.
- `file`, `url`, `remote`, `numbers`, `input`, `merge`, and dictionary sources.
- Table-backed `IN`, `NOT IN`, and nested set operands.
- Top-level and nested `SETTINGS` clauses.
- Physical tables hidden in CTEs, subqueries, `UNION`, or `EXCEPT` branches.

Literal sets and subqueries over accepted public aliases remained usable and
filtered.

### Resource limits

The live API checks confirmed these parser and API controls:

- Maximum query size of 16 KiB.
- Maximum 64 projected columns.
- Maximum 2,000 parser AST nodes.
- Workspace result-row limit enforcement through rewritten `LIMIT` values.
- Rejection of `SETTINGS readonly = 0` and resource-setting overrides.
- Rejection of API access to `system.settings`.
- A 4 MiB encoded response limit in both analytics handlers by source review.

Direct ClickHouse checks confirmed `readonly=1`, protected memory, result, and
AST settings, a 1,000-row overflow error, and bounded timeout-setting
constraints on the disposable users.

## Confirmed findings

### HTTP driver timeout conflicts with generated readonly profiles

Before worker dispatch, ordinary valid analytics requests failed against the
profiles created by `ConfigureUser`. The generated profile fixes
`max_execution_time` as `READONLY` in `pkg/clickhouse/user.go`. The API uses a
one-minute request timeout and supports an HTTP analytics DSN. The ClickHouse Go
HTTP transport sends a per-request `max_execution_time` setting derived from
that context.

The local server returned both of these errors during setup:

```plaintext
Cannot modify 'max_execution_time' setting in readonly mode
Setting max_execution_time shouldn't be greater than 30
```

For the two disposable users only, the local profile was changed to a bounded
constraint:

```plaintext
max_execution_time = 65 MIN 1 MAX 65 CHANGEABLE_IN_READONLY
```

All other readonly and resource limits remained fixed. Valid API requests then
succeeded. The live worker results therefore depend on this local profile
adjustment. They do not prove that an untouched production profile works with
an HTTP analytics DSN.

This is an integration and availability failure, not a data-isolation failure.
The value 65 is a local compatibility setting, not a recommended production
constant.

### Multiple statements are accepted and truncated

This request returned HTTP 200 with the first statement's count:

```sql
SELECT count() AS total FROM key_verifications_v1;
SELECT count() AS forbidden FROM system.tables
```

`pkg/clickhouse/query-parser/parser.go` calls `ParseStmts()` and then reads only
`stmts[0]`. The second statement was not included in the rewritten SQL and did
not execute. This prevented system-table access in the observed case, but it is
still the wrong contract. Invalid trailing SQL can pass unnoticed, and safety
depends on the parser continuing to discard it.

### Public-alias CTE names do not behave as CTEs

This query returned the five filtered verification rows instead of the single
CTE row:

```sql
WITH key_verifications_v1 AS (SELECT 1 AS c)
SELECT count() AS c FROM key_verifications_v1
```

The equivalent rate-limit query returned two physical-table rows. In
`pkg/clickhouse/query-parser/tables.go`, alias resolution happens before the CTE
name check. A CTE name that matches a public alias is rewritten to the physical
table.

Workspace and resource filters still applied, so no cross-scope data appeared.
The behavior is nevertheless surprising and contradicts the SQL text.

### Direct users can read limited system metadata

The direct helper produced these results:

```plaintext
ws_rtalpha: SELECT count() FROM system.tables  -> 10
ws_rtbeta:  SELECT count() FROM system.columns -> 125
```

`system.tables` listed the 10 approved physical analytics tables. Access to
`system.users` was denied, and reads from unapproved `default.*` tables were
also denied. The API parser rejected every tested `system.*` reference.

This is a defense-in-depth decision for direct ClickHouse credentials. The run
did not show cross-workspace data or unrestricted server metadata.

### Public execution errors reveal rewritten SQL details

An unknown-column request returned HTTP 400, but its public `detail` included:

- The rewritten `default.key_verifications_raw_v2` name.
- The caller's `workspace_id` predicate.
- The rewritten query text twice.

The workspace ID belonged to the authenticated caller, and no other tenant's
data appeared. The response still exposes implementation details that the
public table boundary otherwise hides.

### Function-name validation and execution disagree

The parser lowercases function names for its allowlist. It preserves the
original spelling in rewritten SQL. `countIf(...)` succeeded, while
`COUNTIF(...)` passed parser validation and then failed in ClickHouse with HTTP
400 because that spelling is not a ClickHouse function.

This is a correctness and error-quality problem, not an authorization failure.

## Regression coverage and resolution

Focused regression tests encode the intended contracts at their owning seams.
Each test reproduced the finding before its fix and now passes:

| Contract | Test | Resolution |
| --- | --- | --- |
| Reject every multi-statement input | `TestParser_RejectsMultipleStatements` | Parsing now requires exactly one statement. |
| Prevent public-alias CTE collisions from changing query semantics | `TestParser_RejectsCTEsThatShadowPublicAliases` | CTE names that collide with public aliases are rejected before rewriting. |
| Emit ClickHouse-compatible allowlisted function names | `TestParser_EmitsClickHouseCompatibleAllowedFunctionNames` | Case-sensitive functions are emitted with canonical ClickHouse spelling. |
| Keep rewritten SQL details out of public errors | `TestWrapClickHouseError_HidesRewrittenQuery` | ClickHouse errors map to stable messages without rewritten SQL. |
| Support the API's bounded HTTP context with readonly profiles | `TestConfigureUser_HTTPTransportAndMetadataContracts/HTTP_transport_accepts_the_API_deadline` | The client preserves cancellation without sending a deadline-derived setting; the profile permits only values from one through the configured cap. |
| Hide physical schema metadata from direct workspace users | `TestConfigureUser_HTTPTransportAndMetadataContracts/direct_user_cannot_inspect_physical_schema_metadata` | Workspace users receive empty row policies on `system.tables` and `system.columns`. |

The API-level rejection of direct system and physical table references remains
unchanged.

Verification after the fixes:

- `mise exec -- rask --no-cache ./pkg/clickhouse/...`: 1,173 passed.
- `mise exec -- rask --no-cache ./internal/services/analytics`: three passed.
- `mise exec -- golangci-lint run ./pkg/clickhouse/...`: no issues.
- The two route-package checks remain blocked by the existing local MySQL
  schema drift: `quotas.max_replicas_per_region` is missing from the shared
  test database.

## Inconclusive or limited checks

- The workers did not generate a response over 4 MiB. Source inspection
  confirmed the handler limit, but live overflow behavior remains unverified.
- The workers did not issue sleeps, intentionally slow scans, or unbounded
  generators. Timeout error mapping was inspected, but live query timeout
  behavior remains unverified after the local profile adjustment.
- Small fixtures could not trigger every row-overflow path through the API.
  Direct ClickHouse access did confirm the 1,000-row profile limit.
- A method mismatch returned the correct HTTP 405 status with a plain-text body
  instead of the structured problem response used by the other tested errors.
- Backtick- and double-quoted public aliases returned authorized rows. The
  public contract does not specify whether equivalent identifier quoting must
  be accepted, so this is not classified as a failure.

## Follow-up status

The HTTP transport, statement count, CTE collision, metadata visibility, error
sanitization, and function-spelling findings now have passing regression tests
and fixes.

Two lower-priority checks remain:

1. Add controlled integration fixtures for response-byte, timeout, and API
   result-overflow behavior instead of reproducing those limits with large
   manual local queries.
2. Decide whether all route and method errors must use the same structured
   problem response. Add a 405 response-shape test if that consistency is part
   of the API contract.

## Remaining local state

The worker checks made no environment changes. Earlier setup left these
disposable resources in the local development environment:

- Workspaces, analytics users, permissions, row policies, settings, and
  sentinel rows for `ws_rtalpha` and `ws_rtbeta`.
- Modified timeout profiles for those two disposable users.
- Kubernetes secret `unkey/github-credentials` with local placeholder data.
- A local generated key, credential helper files, and raw worker reports in the
  run's temporary working directory.

They remain in place to avoid interrupting the active Minikube development
session. Remove them after review if the fixtures are no longer needed.
