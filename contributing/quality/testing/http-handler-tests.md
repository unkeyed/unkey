---
title: "HTTP handler tests"
description: "Testing API endpoints with the test harness"
notion:
  owners:
    - andreas
  tags:
    - Quality
    - Testing
---

## Testing the full stack

HTTP handler tests exercise API endpoints from request to response. A single request might authenticate a user, check permissions, validate input, query a database, update a cache, and write an audit log. Testing handlers end to end catches bugs that unit tests miss.

Every handler must have coverage for its success, validation, authentication,
and resource authorization guarantees. Coverage can come from shared
conformance tests. Do not copy a generic authentication suite into each handler
or require one file per category.

Each handler test must make the API guarantee visible. The reader must know which contract the endpoint promises, which client-visible response is expected, and which side effects are required or forbidden.

Authentication conformance belongs to the shared credential boundary. Resource
authorization belongs to the handler or resource service that knows the tenant
and permissions. Production wiring tests must exercise the production router,
including route authentication configuration, rather than only a test router
that supplies authentication automatically. A shared helper does not prove that
every production route uses the boundary correctly.

## Anatomy of a handler test

Use `svc/api/internal/testutil`. Create a harness, configure the handler with
harness dependencies, register the handler, create credentials, make a request,
and verify the response. This example tests handler behavior, not production
router registration.

```go
func TestCreateApi_Success(t *testing.T) {
    h := testutil.NewHarness(t)

    route := &handler.Handler{
        DB:        h.DB,
        Auditlogs: h.Auditlogs,
    }

    h.Register(route)

    rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_api")
    headers := http.Header{
        "Content-Type":  {"application/json"},
        "Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
    }

    req := handler.Request{Name: "my-new-api"}
    res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

    require.Equal(t, http.StatusOK, res.Status)
    require.NotEmpty(t, res.Body.Data.ApiId)
}
```

## Organizing test files

Organize tests by guarantees so the intent is clear. Keep route-specific
success, validation, and authorization cases near the handler. Extend existing
cases before creating files. Shared credential conformance and production
registration tests belong at their shared boundaries.

Test count and file layout are not evidence of coverage. Name the existing
conformance or route test that protects each changed guarantee.

## Testing success cases

Test minimal requests, full requests, and any important variations. Verify side effects such as audit log writes when relevant.

## Testing validation errors

Test boundary conditions, missing required fields, and invalid formats. These tests document the API contract.

Use test names and docstrings to identify the contract. For example, document whether invalid input must return `400`, whether the error code is stable for SDKs, and whether the handler must avoid writing partial state.

## Testing authentication

Reject missing headers, malformed tokens, and revoked credentials.

Authentication tests protect security guarantees. Document the guarantee when a case covers token confusion, revoked credentials, replay behavior, or information disclosure.

Use the shared authentication conformance for generic credential rejection.
Add handler-specific cases only when the route introduces different semantics.
Include valid credentials as a control, and use named
[red/green evidence](./index.md#fresh-reliability-and-named-witnesses) for changed
security behavior.

`testutil.CheckAuthentication` checks an unknown credential and a valid
credential on an already registered route. Its callbacks must verify denied
business effects and the successful response. Credential parsing and lifecycle
matrices belong at the shared authentication service boundary.

## Testing authorization

Verify that permissions are enforced and cross-workspace access is rejected.

Authorization tests must state the resource boundary they protect. If a response intentionally hides existence with `404`, document that behavior in the test so future changes do not weaken it by accident.

`testutil.CheckAuthorization` uses real root keys and independently supplied
grants. Provide denied and allowed requests, complete public errors, and
business-state callbacks. Denial expectations must be `403` or `404`: a `400`
validation failure or `401` authentication failure is not authorization proof.
Neither conformance helper registers routes or proves production registration.

## Helper functions

Extract repeated setup into helpers. If a helper asserts, it must call `t.Helper()`.

## Debugging failed requests

Use the raw response body to inspect failures:

```go
if res.Status != http.StatusOK {
    t.Logf("Response body: %s", res.RawBody)
}
```
