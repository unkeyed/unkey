---
title: "Anti-patterns"
description: "Common testing mistakes to avoid"
notion:
  owners:
    - andreas
  tags:
    - Quality
    - Testing
---

## Sleeping instead of synchronizing

Do not use `time.Sleep` to wait for async work. Use `require.Eventually` or a test clock.

## Testing implementation details

Verify public behavior, not internal fields. Refactors must not break tests when behavior is unchanged.

## Treating tests as throwaway code

Tests are production code. Do not accept unclear names, hidden setup, flaky timing, duplicated fixtures, unsafe casts, ignored errors, or comments that would be rejected in application code.

## Hiding the guarantee

Do not make readers infer the guarantee from setup and assertions alone. Use a precise test name, table case name, and docstring when the protected behavior is not obvious.

```go
// Bad: the guarantee is hidden.
func TestAuth(t *testing.T) {
    // ...
}

// Good: the guarantee is explicit.
// TestAuthRejectsRevokedRootKeys guarantees that revoked credentials cannot be
// used after revocation has been persisted.
func TestAuthRejectsRevokedRootKeys(t *testing.T) {
    // ...
}
```

## Raw SQL in tests

Do not write raw SQL in tests. Tests must read and write data through the same typed layer as production code.

Use the first option that covers your case:

1. A seed or harness helper, such as `h.CreateWorkspace()`, `h.CreateKey(...)`, or `h.Seed.CreateAPI(...)`.
2. A generated sqlc query from `pkg/db`, such as `db.Query.InsertProject(ctx, h.DB.RW(), db.InsertProjectParams{...})`. Search `pkg/db/queries/` before you decide that a query does not exist.
3. A new query in `pkg/db/queries/`, named by `pkg/db/NAMING_STANDARDS.md`, generated with `mise run generate`. A query that only tests use is acceptable if its doc comment says so.

Raw SQL in a test breaks silently when the schema changes, skips the column types and enums that sqlc generates, and can put the database in a state that production code cannot create.

```go
// Bad: raw SQL, although db.Query.SoftDeleteUnkeyRootKey exists.
_, err := h.DB.RW().ExecContext(ctx, "UPDATE unkey_root_keys SET deleted_at = 1 WHERE id = ?", key.KeyID)
require.NoError(t, err)

// Good: generated query with typed parameters.
_, err := db.Query.SoftDeleteUnkeyRootKey(ctx, h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
    Now:         sql.NullInt64{Int64: 1, Valid: true},
    ID:          key.KeyID,
    WorkspaceID: workspace.ID,
})
require.NoError(t, err)
```

Set the state you need when you create the row. Pass the values to the seed request or the `Insert...Params` struct. Do not insert a default row and then patch it with `UPDATE`.

For ClickHouse, insert rows as `pkg/clickhouse/schema` structs with `clickhouse.InsertQuery[T]()` and `batch.AppendStruct`. Do not write `INSERT INTO ... VALUES` strings. Read rows through the reader functions in `pkg/clickhouse` when they exist.

Raw SQL is acceptable only in these cases. If the test name does not make the case obvious, add a short comment that says why the raw SQL is necessary:

- The SQL itself is the subject of the test, such as a migration, an SQL parser, an SQL comment injector, a code generator, or an endpoint that accepts user SQL.
- The test builds a corrupt state on purpose, such as a dangling foreign reference, and no query in production can create that state.
- The test reads ClickHouse system tables or a table that has no reader function.

## Raw JSON in tests

Do not write JSON as string literals in tests. Build a typed value and let the encoder produce the JSON. Structs cost almost nothing to write, the compiler checks field names, and renamed fields fail at build time instead of at runtime.

Use the existing request and response types first: OpenAPI types in `svc/api/openapi`, handler `Request` and `Response` types, and `testutil.CallRoute[Req, Res]`. Decode responses into typed structs and compare structs. Do not compare JSON strings.

```go
// Bad: hand-written JSON.
w.Write([]byte(`{"meta":{"requestId":"test"},"data":{}}`))

// Good: typed value, encoded.
require.NoError(t, json.NewEncoder(w).Encode(openapi.V2ProjectsCreateProjectResponseBody{
    Meta: openapi.Meta{RequestId: "test"},
}))
```

If no named type exists, use an anonymous struct or `map[string]any` and marshal it. In TypeScript, build a typed object and call `JSON.stringify`.

Raw JSON is acceptable only when the bytes themselves are the subject of the test: malformed or truncated input, parser edge cases such as duplicate keys or field order, signature or redaction tests that depend on exact bytes, and fixed wire-format fixtures. If the test name does not make it obvious, add a short comment that says why the exact bytes matter.

## Bypassing `pkg/fuzz`

Fuzz tests must use `pkg/fuzz`. Do not fuzz typed parameters directly, use ad hoc byte slicing, or introduce separate randomness with `math/rand`. Use `fuzz.Seed(f)`, `fuzz.New(t, data)`, and the consumer helpers so every generated value remains controlled by the fuzzer.

## Shared mutable state

Avoid global state between tests. Isolate resources per test or reset state explicitly.

## Ignoring setup errors

Fail fast on setup errors so failures are clear and localized.

## Hardcoded identifiers

Do not hardcode IDs such as `"ws_123"`, `"key_test"`, or `"api_1"` in tests. Generate them with `uid.New` and the matching prefix from `pkg/uid`. Use `uid.TestPrefix` only when no entity prefix applies.

Hardcoded IDs collide when tests run in parallel against shared MySQL, ClickHouse, Redis, S3, or Restate. They also hide which values the test controls and which values come from the system under test.

```go
// Bad: collides with any other test that uses the same ID.
workspaceID := "ws_123"

// Good: unique for every run.
workspaceID := uid.New(uid.WorkspacePrefix)
```

Assert on the generated variable, not on a copy of the literal. A hardcoded ID is acceptable only when the exact value is the subject of the test, such as ID parsing, URN formatting, or a golden output.

## Over-mocking

Prefer real dependencies when feasible. Mocks often assert calls instead of behavior.

## Missing subtests

Use `t.Run` for table cases so failures identify the case.

## Forgetting `t.Helper()`

Helpers that assert must call `t.Helper()` so failures point at the call site.
