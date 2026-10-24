# Internal Go SDK

This package generates typed clients for the `cli` OpenAPI tag. The CLI uses it
for device login, refresh, and revocation without waiting for a public SDK release.
Other CLI API commands still use the public Go SDK. This package doesn't replace
the dashboard's TypeScript SDK.

## Regenerate

Run from the repository root:

```bash
mise run generate-internal-sdk
```

The task bundles `svc/api/openapi/openapi-split.yaml` and its `spec/` files, then
generates this client with the pinned Speakeasy CLI. You need Speakeasy
authentication, through `SPEAKEASY_API_KEY` or an existing login. Generation
uses Speakeasy's service but doesn't publish an SDK. The full `mise run generate`
task also runs this generator. Building and testing the committed client don't
require Speakeasy credentials.

Edit the OpenAPI source, not generated Go files. `overlay.yaml` selects the
`cli` operations, enables them for this internal target, and adds polling
criteria. It doesn't change the public SDK's exclusions. `gen.yaml` and
`.speakeasy/gen.lock` pin generation settings and versions. The task copies Go
output from a staging directory without creating a nested Go module. Commit
generated output with its source changes.

## Use the client

Construct a client with `sdk.NewClient(apiURL, httpClient)`. A nil HTTP client
uses a 30-second timeout. Call a generated `client.Cli` method with its typed
request from `models/components`, handle the error, and read its typed response
body. Non-JSON success responses return an error. Types don't replace
server-side validation of required fields and permission checks.

Pass `sdk.Bearer(token)` to authenticated calls. The client doesn't load
credentials from the environment or retain a token for later requests.

`sdk.NewClient` limits responses to 1 MiB and returns `*sdk.ResponseError` for HTTP
failures or oversized responses. `*sdk.TransportError` wraps request and body
transfer failures. Generated methods can wrap these errors. Use `errors.As`
to inspect them. Ordinary calls don't retry. Refresh retains its separate
two-attempt policy.

## Poll for device login

Speakeasy generates `Cli.ExchangeTokenWaitForCompletion` from the overlay's
`x-speakeasy-polling` extension. Pass it through `operations.WithPolling` to
`Cli.ExchangeToken` with a `loginId`. It stops on `complete`, returns a
`polling.FailureCriteriaError` with the response for `access_denied` or
`expired_token`, and returns `polling.LimitCountError` after 120 pending responses.

The CLI uses `client.WaitForDeviceLogin` to run that generated poller with
server-provided intervals and cancellable waits. Speakeasy's generated Go loop
uses a fixed `time.Sleep`, so this adapter disables that sleep and supplies
request pacing. It retries transport failures, 429, and 5xx responses at the
last interval. Pass a context with a deadline to bound retries and in-flight
requests, the interval from `startDeviceLogin`, and nil for the default wait.
The CLI uses the login's expiry, or 10 minutes if the server omits it.
