---
name: gen-api-cli
description: Generate or update CLI commands in cmd/api/ from the OpenAPI spec and Go SDK. Use when the API spec changes or new endpoints are added.
disable-model-invocation: true
allowed-tools: [Read, Write, Edit, Glob, Grep, Bash]
argument-hint: "[group-name or 'all']"
---

# Generate CLI commands from the OpenAPI spec

Generate and maintain CLI command files in `cmd/api/` that wrap the Unkey Go SDK. Each supported API endpoint has one implementation shared by its canonical resource command and legacy `unkey api` path.

The body-mode rules below apply only to endpoints with an OpenAPI request body. Bodyless operations, such as `workspace.getFlags` and `workspace.getLimits`, omit `--body` and call the SDK without a request argument. Cover their authentication, routes, empty requests, and rejection of `--body` in the shared integration suite instead of adding request-construction tests.

**Modes** (determined by the argument passed after `/gen-api-cli`):
- A group name (e.g., `keys`, `apis`): generate/update that group only
- `all` or no argument: generate/update all groups
- `check`: audit mode — do NOT write any files. Instead, compare all existing commands against the OpenAPI spec and report discrepancies

## Audit mode

When the argument is `check`:

1. Read the OpenAPI spec and build a list of all endpoints (group + action + description + request body fields)
2. Read all existing command files under `cmd/api/`
3. Report:
   - **Missing commands**: endpoints in the spec that have no CLI command
   - **Stale commands**: CLI commands whose descriptions no longer match the OpenAPI spec (compare verbatim, ignoring markdown stripping)
   - **Flag mismatches**: request body fields in the spec that are missing as CLI flags, CLI flags that no longer exist in the spec, leaves without exactly one explicit `--body` flag, or request-building flags without `cli.MutuallyExclusive("body")`. Do not require mutual exclusion on operational flags.
   - **Type mismatches**: flag types that don't match the OpenAPI property types (e.g., spec says integer but flag is string)
   - **Presentation drift**: missing canonical paths in `resourcePaths`, docs that teach legacy paths, or output that bypasses the shared human/JSON renderer
4. Output a summary with file paths and specific lines that need attention
5. Do NOT modify any files in this mode

## Step 1: Use the pinned SDK

Use the SDK version pinned in `go.mod` and `go.sum`. Do not run routine dependency upgrades or `go get ...@latest` during command generation. If the pinned SDK lacks a required endpoint or type, report the mismatch. Change the dependency only when that upgrade is explicitly in scope.

## Step 2: Read the inputs

1. **OpenAPI spec**: Read `svc/api/openapi/openapi-generated.yaml` — the source of truth for all endpoints.
2. **Go SDK source**: Find the exact version in `go.mod` under `github.com/unkeyed/sdks/api/go/v3`, then read the SDK source from the Go module cache at `~/go/pkg/mod/github.com/unkeyed/sdks/api/go/v3@<version>/`. You MUST read the actual SDK files to verify struct names, field names, and field types. Do not guess.
3. **Existing code**: Read `cmd/api/util/` (shared helpers, do not change for routine generation), `cmd/api/root.go`, `cmd/api/resources.go`, `build/cli/main.go`, and the affected subpackages. Verify the output policy in `util/output.go` and `util/human.go` rather than recreating it in a leaf command.

## Step 3: Group endpoints

Parse all paths from the OpenAPI spec. They look like `/v2/{group}.{action}`.

Group them by resource:
- `apis` — endpoints matching `apis.*`
- `apps` — endpoints matching `apps.*`
- `deployments` — endpoints matching `deployments.*`
- `environments` — endpoints matching `environments.*`
- `gateway` — endpoints matching `gateway.*`
- `keys` — endpoints matching `keys.*`
- `identities` — endpoints matching `identities.*`
- `permissions` — endpoints matching `permissions.*`
- `portal` — bearer-authenticated endpoints matching `portal.*`; exclude browser-session, cookie-authenticated, and unauthenticated operations
- `projects` — endpoints matching `projects.*`
- `ratelimit` — endpoints matching `ratelimit.*`
- `analytics` — endpoints matching `analytics.*`
- `rootkeys` — endpoints matching `rootKeys.*`, exposed as `root-keys`
- `logs` — endpoints matching `logs.*`
- `workspace` — endpoints matching `workspace.*`; the CLI entrypoint adds the local `list` and `switch` commands to this resource group

Skip `deploy.*` (already handled by `cmd/deploy`) and `liveness` (GET health check, not useful as CLI).

## Step 4: Directory structure

Each group is a subpackage under `cmd/api/`. Each command is its own file:

```
cmd/api/
├── root.go                        # Cmd, imports and registers group subpackages
├── resources.go                   # ResourceCommands and canonical resourcePaths
├── util/
│   ├── client.go                  # CreateClient
│   ├── output.go                  # Output
│   ├── human.go                   # Human-readable tables and details
│   ├── errors.go                  # FormatError
│   └── flags.go                   # Shared flags
├── apis/
│   ├── root.go                    # Cmd (group command), registers leaf commands
│   ├── create_api.go
│   ├── delete_api.go
│   └── ...
├── keys/
│   ├── root.go
│   ├── create_key.go
│   ├── verify_key.go
│   └── ...
```

### Group root.go pattern

```go
package {group}

import "github.com/unkeyed/unkey/pkg/cli"

func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "{group}",
		Usage:       "...",
		Description: "...",
		Commands: []*cli.Command{
			{leafCmd1}(),
			{leafCmd2}(),
		},
	}
}
```

### Leaf command file pattern

Each leaf command lives in its own file named with the kebab-case action using underscores: `create_api.go`, `verify_key.go`, `list_overrides.go`.

The constructor is an unexported function returning a fresh `*cli.Command`: `createAPICmd()`, `verifyKeyCmd()`, `listOverridesCmd()`.

### Register new groups

After creating a new group subpackage, import it in `cmd/api/root.go` and add `{group}.Cmd()` to `groups()`. Add each endpoint's canonical path to `resourcePaths` in `cmd/api/resources.go`. `ResourceCommands()` reuses fresh endpoint constructors for the canonical tree. Do not duplicate request handling or share mutable command instances between the two trees.

Each leaf command explicitly declares one fresh `--body` flag. Group commands and `cmd/api.Cmd()` do not add or decorate body flags. Every request-building flag on the leaf must include `cli.MutuallyExclusive("body")`. Operational flags for authentication, API URL, configuration, and output must not include that option.

### Disclaimer

Every group `root.go` must append `util.Disclaimer` to its Description:
```go
Description: "Create, read, and delete API namespaces." + util.Disclaimer,
```
Leaf commands must also append `util.Disclaimer` to their Description, after the docs link:
```go
Description: `...

For full documentation, see https://www.unkey.com/docs/api-reference/v2/...` + util.Disclaimer,
```

## Command naming conventions

- **Canonical command path**: Use the resource followed by an action. Read `resourcePaths` as the source of truth, including moved resources and nested subresources: `projects list`, `keys list`, `roles create`, `environments variables list`, and `keys permissions add`. Keep existing legacy paths working.
- **Legacy endpoint name**: Convert the action part of the operationId to kebab-case.
  - `createApi` → `create-api`
  - `listKeys` → `list-keys`
  - `verifyKey` → `verify-key`
  - `addPermissions` → `add-permissions`
  - `limit` → `limit`
  - `multiLimit` → `multi-limit`
  - `getVerifications` → `get-verifications`

- **File name**: kebab-case action with underscores: `create_api.go`, `verify_key.go`

- **Constructor name**: camelCase with SDK acronym rules: `createAPICmd`, `verifyKeyCmd`

## SDK naming conventions (Speakeasy-generated)

The SDK applies Go naming conventions with acronym uppercasing:

**Operation ID `{group}.{action}` maps to:**
- SDK namespace: PascalCase(group) — `apis` → `Apis`, `ratelimit` → `Ratelimit`
- SDK method: PascalCase(action) with acronyms — `createApi` → `CreateAPI`, `verifyKey` → `VerifyKey`

**Acronym rules** — these substrings get uppercased when followed by uppercase or end-of-string:
- `Api` → `API` (but `Apis` stays `Apis`)
- `Id` → `ID` (but `Identity` stays `Identity`)
- `Url` → `URL`

**Type names:**
- Request body: apply acronym rules to the schema name from the OpenAPI `$ref` — e.g., `V2ApisCreateApiRequestBody` → `V2ApisCreateAPIRequestBody`
- Response body field on the operation result: same pattern — e.g., `V2ApisCreateAPIResponseBody`

**CRITICAL**: Always verify type and field names by reading the actual SDK source files in the Go module cache. `grep` for the type name to confirm. If a name is wrong, the code won't compile.

## Descriptions — copy from OpenAPI verbatim

This is extremely important. Descriptions must be copied from the OpenAPI spec as closely as possible.

### Command `Usage` (short one-liner)
Use the first sentence of the OpenAPI path `description` field.

### Command `Description` (full help text)
Copy the entire OpenAPI path `description` field verbatim, with these adjustments:
- Strip markdown formatting: `**text**` → `text`, `` `code` `` → `code`
- Format permissions as a flat bullet list (no extra indentation):
  ```
  Required permissions:
  - api.*.create_api
  - api.<api_id>.create_api
  ```
- Do NOT put examples in the Description — use the `Examples` field instead (see below)
- End the description with a docs link. Look up the correct URL from https://unkey.com/docs/llms.txt — the URLs follow the pattern `https://www.unkey.com/docs/api-reference/v2/{group}/{slug}` but the slugs don't always match the operation ID (e.g., `keys.createKey` → `/v2/keys/create-api-key`). Always verify the URL exists. Format as:
  ```
  For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/create-api-key
  ```

### Command `Examples`
Use the `Examples` field ([]string) on the Command struct. Each entry is one example invocation.
These are rendered in a separate EXAMPLES section at the bottom of --help output.
Always use `--flag=value` syntax (not `--flag value`) in examples for clarity.
Every example must pass all required flags and satisfy `RequireOneOf` or any
action-level source/update requirement. Use realistic IDs and slugs, and use
syntactically valid JSON that satisfies the referenced schema. Run examples
through command parsing in tests when practical.
Endpoint constructors keep their legacy examples so `ResourceCommands()` can rewrite them using `resourcePaths`. Product documentation uses the canonical paths.

### Flag descriptions
Use a short, one-sentence summary of the OpenAPI property `description`. Since every command links to full docs, flag descriptions should be concise — just enough to know what the flag does. Do NOT copy the full multi-sentence OpenAPI description.
For ID-or-slug selectors, say so explicitly. For pagination, document the
cursor's source and the authoritative default or bounds from the schema. For
closed enums, list every valid choice.

### Closed enums

Validate closed enum-backed request-building flags before making an API call.
Build allowed values from generated SDK constants rather than duplicating
string literals. Use `cli.Enum` for scalar values. For comma-separated enum
lists, use `cli.StringSlice` with `cli.Validate` to validate every parsed value,
and include all choices in help. This validation does not apply to `--body`,
which continues to be decoded by the SDK.

## Flag mapping

Every leaf command includes these operational flags:
```go
util.RootKeyFlag(),
util.APIURLFlag(),
util.ConfigFlag(),
util.OutputFlag(),
```

`groups()` adds fresh `--json` and `--debug` flags to endpoint commands for both command trees. Do not declare duplicates in the leaves or make these operational flags mutually exclusive with `--body`.

Add one explicit `cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive.")` to the leaf. Keep flag order consistent with existing commands.

Then map OpenAPI request body properties to CLI flags:

| OpenAPI type | SDK Go type | CLI flag | Read value |
|---|---|---|---|
| `string` (required) | `string` | `cli.String("name", "desc", cli.Required())` | `cmd.String("name")` |
| `string` (optional) | `*string` | `cli.String("name", "desc")` | check non-empty, then `&v` |
| `integer` (required) | `int64` | `cli.Int64("name", "desc", cli.Required())` | `cmd.Int64("name")` |
| `integer` (optional) | `*int64` | `cli.Int64("name", "desc")` | check non-zero, then `&v` |
| `boolean` (optional) | `*bool` | `cli.Bool("name", "desc", cli.Default(X))` | `new(cmd.Bool("name"))` |

**Boolean flag defaults**: Look up `default:` in the OpenAPI spec. Use `cli.Default(true)` or `cli.Default(false)` accordingly. For partial-update endpoints where omitting a boolean means "don't change" (no `default:` in spec), do NOT set a default — use `cmd.FlagIsSet("name")` to check if the user explicitly passed it:
```go
if cmd.FlagIsSet("enabled") {
    req.Enabled = new(cmd.Bool("enabled"))
}
```
| `array of strings` | `[]string` | `cli.StringSlice("name", "desc")` | `cmd.StringSlice("name")` |
| `object` / `map` / nested | complex | `cli.String("domain-name", "Domain value as a JSON object or array.")` | `json.Unmarshal` |

**JSON flags**: Name complex flags after their domain field, such as `--meta`, `--credits`, or `--ratelimits`. Do not add a `-json` suffix. The CLI help description and product docs MUST explicitly state that the value is encoded as JSON, using wording such as "as JSON," "JSON object," or "JSON array." Show the JSON shape in the command's `Examples` field with realistic values. Every command that has JSON flags MUST include at least one example showing its usage.

Keep cohesive domain objects together instead of expanding their nested fields
into separate flags. In particular, `gateway.updatePolicy` uses one `--policy`
JSON object for all mutable policy fields, while `gateway.setPolicies` uses one
`--policies` JSON array. Project, app, environment, and policy ID remain separate
selector flags.

**Flag naming**: Convert camelCase property names to kebab-case: `apiId` → `api-id`, `externalId` → `external-id`.

## Action pattern

Every leaf command uses a plain `func(ctx, cmd) error` action. For commands without
confirmation, create the client, format errors, and print output as follows:

```go
Action: func(ctx context.Context, cmd *cli.Command) error {
    client, err := util.CreateClient(ctx, cmd)
    if err != nil {
        return err
    }

    if cmd.FlagIsSet("body") {
        body := cmd.String("body")
        res, err := util.SendBody(ctx, client.Apis.CreateAPI, body)
        if err != nil {
            return err
        }
        return util.Output(cmd, res.V2ApisCreateAPIResponseBody)
    }

    req := components.V2ApisCreateAPIRequestBody{Name: cmd.String("name")}

    // Optional string:
    if v := cmd.String("prefix"); v != "" {
        req.Prefix = &v
    }

    // Optional int64:
    if v := cmd.Int64("limit"); v != 0 {
        req.Limit = &v
    }

    // Optional bool — look up the default in the OpenAPI spec (check `default:` on the property).
    // Use cli.Bool with cli.Default and new:
    req.Enabled = new(cmd.Bool("enabled")) // default: true in spec
    req.Decrypt = new(cmd.Bool("decrypt")) // default: false in spec

    // For partial-update endpoints where omitting a bool means "don't change":
    if cmd.FlagIsSet("enabled") {
        req.Enabled = new(cmd.Bool("enabled"))
    }

    // String slice:
    if v := cmd.StringSlice("permissions"); len(v) > 0 {
        req.Permissions = v
    }

    // JSON field (complex object):
    if v := cmd.String("meta"); v != "" {
        var meta map[string]any
        if err := json.Unmarshal([]byte(v), &meta); err != nil {
            return fmt.Errorf("invalid JSON for --meta: %w", err)
        }
        req.Meta = meta
    }

    res, err := client.Apis.CreateAPI(ctx, req)
    if err != nil {
        return fmt.Errorf("%s", util.FormatError(err))
    }
    return util.Output(cmd, res.V2ApisCreateAPIResponseBody)
},
```

Every leaf command explicitly declares `cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive.")`. Add `cli.MutuallyExclusive("body")` to every endpoint request-building flag, including optional pagination, filter, and boolean flags. Do not add it to `util.RootKeyFlag()`, `util.APIURLFlag()`, `util.ConfigFlag()`, or `util.OutputFlag()`.

For commands without confirmation, explicitly supplied `--body` JSON is decoded and sent by `util.SendBody` using the endpoint's exact generated SDK request type. Branch on `cmd.FlagIsSet("body")` immediately after client creation and before normal request construction, JSON parsing, and action-level validation. Pass `cmd.String("body")` and the generated SDK method directly, for example `client.Apis.CreateAPI`. Do not wrap the method in a closure. Destructive commands use `util.RequestBody` as described below instead of an early send. An explicitly empty body activates body mode and is rejected as invalid JSON. Mutual exclusion waives required and `RequireOneOf` checks for request-building flags. Request schema and business validation belong to the API.

Authentication and transport/output flags remain active. Root-key and config bearer authentication still apply, while `--api-url` and `--output` retain their normal behavior. Do not generate browser-session or cookie-authenticated commands; this CLI supports bearer-authenticated API operations only.

Generated union and slice request types are inferred from the direct SDK method value and are supported when they implement normal JSON unmarshalling.

## Destructive confirmation

Every delete or remove command must confirm before sending, including removal of environment variables, key roles, or key permissions. Follow `cmd/api/keys/delete_key.go` and `cmd/api/environments/remove_environment_variables.go`. Do not add confirmation to promotion, rollback, stop, or general updates without an explicit requirement.
- Add `util.YesFlag()` as an operational flag, never mutually exclusive with `--body`.
- Resolve the exact SDK request with `util.RequestBody(cmd, typedFlagRequest)`. Decode and reject malformed body input before confirmation. Do not send body mode through `util.SendBody` before confirming.
- Build a target-specific question from that resolved request, not separately from flags. Show requested project/app/environment and variable names for variable removal, all environments for app/project deletion, and IDs/slugs/names for workspace targets. Never infer an environment. Distinguish permanent key erasure.
- Call `util.Confirm(cmd, question)`, then send the same typed request through the SDK and use shared error/output helpers. Confirmation uses stderr and requires both stdin and stderr terminals, explicit `y`/`yes`, default N, and cancellation on EOF. Noninteractive callers must pass `--yes`; it waives no authentication, permissions, or delete protection.
- Request-capture tests for these commands pass `--yes`. Cover both command trees and body/flag modes, cancellation without a destructive request, nonterminal rejection, and protection errors after `--yes`; use `cmd/api/destructive_test.go` for the shared behavior.

## Presentation policy

Always pass the complete SDK response body to `util.Output`, including in body mode. Never extract only `data`, print a request ID in a leaf, or implement another renderer.

- Human output is the default, including for pipes. Lists use summary tables with selected columns. Empty lists use a resource-specific message such as `No projects found.` Single resources use labeled details. Actions without data use a resource-specific confirmation such as `Key deleted.`, `Settings updated.`, or `Variables saved.` Asynchronous actions say they were requested rather than completed.
- `--json` and `--output=json` preserve the full response envelope, including all fields, metadata, and pagination. Scripts must select JSON explicitly.
- `--output` and `UNKEY_OUTPUT` accept only `human` and `json`. Unsupported values are errors, not another spelling of human output.
- Successful request IDs appear only in JSON metadata or with `--debug` on stderr, labeled `Request ID:`. API errors go to stderr and include a labeled request ID when available. Never mix diagnostics into stdout.
- Human output includes a cursor hint when more results are available. In body mode, the hint must tell the user to set `cursor` inside `--body`, never combine it with `--cursor`. JSON retains the API pagination fields.
- Login codes, URLs, instructions, and waiting messages go to stderr; only `Signed in.` or `Root key saved.` goes to stdout. Login/logout success must not expose tokens, opaque workspace IDs, or storage paths. Logout's next-active-workspace notice belongs on stderr; revocation failure must not claim success. Keep IDs where workspace switching needs them.
- Deploy progress and context go to stderr; final details/domains go to stdout. Spinners/colors require stderr TTY, `TERM=dumb` disables both, and nonempty `NO_COLOR` disables color. Do not add deploy JSON flags or move diagnostics into results.
- `unkey version` prints the installed CLI version, like `--version`; deployment operations belong under `deployments`, not placeholder version subcommands. Health checks use concise success text on stdout without emoji and errors on stderr; scripts rely on exit status.

If a new response needs a resource-specific table or confirmation, handle it in the shared renderer with focused tests. Do not add per-command formatting.

## Reference implementation

See `cmd/api/apis/create_api.go` for a non-destructive endpoint. Delete and remove commands must use the typed confirmation pattern above instead of its early body send.

## Linter requirements

The project uses strict linters via golangci-lint. Watch out for:

- **exhaustruct**: All struct fields must be initialized, even optional ones. Use `nil` for pointer fields you're not setting:
  ```go
  req := components.V2ApisListKeysRequestBody{
      APIID:  cmd.String("api-id"),
      Limit:  nil,
      Cursor: nil,
  }
  ```
- **errcheck**: All error returns must be checked. Never ignore return values from `Write`, `Fprintln`, `Close`, etc.

## Step 5: Generate documentation

Keep existing documentation filenames, URLs, and navigation entries. The page at `docs/compute/cli/projects/list-projects.mdx`, for example, teaches `unkey projects list`. For new endpoints, follow the existing `docs/{product}/cli/{group}/{legacy-action}.mdx` convention, where `{product}` is `api-management`, `compute`, or `platform`, and register the page in `docs/docs.json`.

Use `docs/compute/cli/projects/list-projects.mdx` as the page template. Titles, sidebar labels, usage, examples, and CLI link text must use canonical resource paths from `resourcePaths`. Preserve API endpoint names and link destinations.

- Include a one-sentence description and source files in frontmatter.
- Document request fields with `<ParamField>`, including their actual required status and defaults. Use the full OpenAPI property descriptions here rather than the CLI help summaries.
- For JSON flags, use `<Expandable>` and nested `<ResponseField>` components based on the referenced OpenAPI schemas. Include a valid example.
- Import `CLIOutput` from `/snippets/cli-output.mdx` inside the `--output` field instead of duplicating the output policy. Link to `/platform/cli/output-and-flags` for the complete shared flag reference, including `--json` and `--debug`.
- For confirmed commands, document `--yes` with `CLIConfirmation` from `/snippets/cli-confirmation.mdx`, and keep the command's flag metadata in `docs/scripts/manifest.json` aligned. Scripts need `--yes` even with `--body` or JSON output.
- Link to the corresponding API endpoint page. Use root-relative documentation links and `bash` fences for shell commands.

Run the rerunnable migration from the repository root after adding or changing resource mappings. It reads `cmd/api/resources.go`, updates command references and shared output guidance, and preserves page paths and link targets:

```bash
mise exec -- node docs/scripts/cli-human-output.mjs
mise exec -- node docs/scripts/cli-human-output.mjs --check
```

In audit mode, run only `--check`. Review the diff after migration, including nested commands and resources whose canonical group differs from their legacy group.

### Audit mode docs and tests check

When running in `check` mode, also report:
- **Missing docs**: CLI commands with no corresponding `.mdx` file in `docs/api-management/cli/`, `docs/compute/cli/`, or `docs/platform/cli/`
- **Missing from nav**: Doc files that exist but aren't registered in `docs.json`
- **Missing tests**: CLI command files with no corresponding `_test.go` file. Do not require per-leaf body-mode tests unless the command has structurally unusual request construction.
- **Body flag ownership**: A leaf command that does not explicitly define exactly one `--body` flag.
- **Body-mode gating**: A leaf that does not declare its own `--body`, mark every request-building flag with `cli.MutuallyExclusive("body")`, or select the typed body before request-flag validation. For destructive commands, require `util.RequestBody` followed by `util.Confirm` before the SDK call in both modes.

## Step 6: Write tests

Each command gets a table-driven test that verifies the exact request body sent to the API.

### Test file location

Each command file `{action}.go` gets a corresponding `{action}_test.go` in the same package.

### Test harness

Use `util.CaptureRequest[T]` which runs the full CLI against a local test server, captures the
request body, unmarshals it into T, and returns it. It fatals on any error.

```go
req := util.CaptureRequest[openapi.V2ApisCreateApiRequestBody](t, Cmd(), "apis create-api --name=test")
```

### Test pattern

Use table-driven tests. Each case is: name, args string, expected struct.

```go
func TestCreateAPI(t *testing.T) {
    tests := []struct {
        name string
        args string
        want openapi.V2ApisCreateApiRequestBody
    }{
        {
            name: "basic",
            args: "apis create-api --name=payment-service",
            want: openapi.V2ApisCreateApiRequestBody{
                Name: "payment-service",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := util.CaptureRequest[openapi.V2ApisCreateApiRequestBody](t, Cmd(), tt.args)
            require.Equal(t, tt.want, req)
        })
    }
}
```

### What to test

For each command, write test cases covering:
- **Minimal request**: only required flags, verify defaults are correct
- **Each optional flag individually**: verify it maps to the correct struct field
- **All flags together**: verify no interference between flags
- **Boolean defaults**: verify the correct default from the OpenAPI spec is sent
- **Boolean omission on update endpoints**: verify omitting `--enabled` does NOT send an `enabled` field (for partial-update commands using `FlagIsSet`)
- **JSON flags**: verify complex objects unmarshal correctly into the expected nested structs

The root integration suite in `cmd/api/root_test.go` owns shared `--body` behavior, including one explicit fresh flag per leaf, typed unmarshalling and SDK re-serialization, malformed JSON rejection (including an explicitly empty body), mutual exclusion with request-building flags, required and `RequireOneOf` waiver, and bearer authentication. Keep unusual generated unions and custom action-level validation covered by tests proving the body branch runs before ordinary request construction and validation.

### Types

- Import request types from `github.com/unkeyed/unkey/svc/api/openapi` (e.g., `openapi.V2KeysCreateKeyRequestBody`)
- Use `new(value)` for pointer fields, matching the repository's Go version
- Use `nullable.NewNullableWithValue()` from `github.com/oapi-codegen/nullable` for nullable fields
- Use `--flag=value` syntax in args (not `--flag value`)
- Commands are functions now (`Cmd()` not `Cmd`), so each test gets fresh flag state

### Important: commands are functions

All commands and group roots are defined as functions returning `*cli.Command`, not package-level
vars. This ensures each test invocation gets fresh flag instances with no stale state from prior
tests. Always call `Cmd()` in tests, never reference a bare `Cmd` variable.

### Reference tests

See `cmd/api/keys/create_key_test.go` and `cmd/api/keys/update_key_test.go` for complete examples.

### Audit mode test check

When running in `check` mode, also report:
- **Missing tests**: CLI command files with no corresponding `_test.go` file for request-building behavior
- **Missing unusual body-mode tests**: Structurally unusual commands whose typed body branch, mutual exclusion, or required-check waiver is not covered

## Step 7: Verify

Write command fields and each flag, example, and child command on separate lines. Run `gofmt` on changed Go files, then run scoped tests, the Go build, and documentation checks through mise:

```bash
mise exec -- gofmt -w <changed-go-files>
mise exec -- rask ./cmd/api/...
mise run build
mise run docs-lint
```

`gofmt` preserves existing line breaks in composite literals; it does not choose their layout. Review declarations for readability instead of adding a custom formatter or formatting tests.

Keep coverage for canonical and legacy command routing, human empty and populated output, full JSON envelopes, invalid output values, pagination, and stdout/stderr separation. Report unavailable checks and pre-existing failures without weakening them. Never use `go build` directly. Always use `mise run build`.
