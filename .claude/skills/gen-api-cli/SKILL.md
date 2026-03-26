---
name: gen-api-cli
description: Generate or update CLI commands in cmd/api/ from the OpenAPI spec and Go SDK. Use when the API spec changes or new endpoints are added.
disable-model-invocation: true
allowed-tools: [Read, Write, Edit, Glob, Grep, Bash]
argument-hint: "[group-name or 'all']"
---

# Generate CLI Commands from OpenAPI Spec

You generate and maintain CLI command files in `cmd/api/` that wrap the Unkey Go SDK.
Each API endpoint from the OpenAPI spec becomes a CLI subcommand.

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
   - **Flag mismatches**: request body fields in the spec that are missing as CLI flags, or CLI flags that no longer exist in the spec
   - **Type mismatches**: flag types that don't match the OpenAPI property types (e.g., spec says integer but flag is string)
4. Output a summary with file paths and specific lines that need attention
5. Do NOT modify any files in this mode

## Step 1: Update the SDK to latest

Before generating anything, update the Go SDK to the latest version:

```
go get github.com/unkeyed/sdks/api/go/v2@latest
go mod tidy
```

## Step 2: Read the inputs

1. **OpenAPI spec**: Read `svc/api/openapi/openapi-generated.yaml` — the source of truth for all endpoints.
2. **Go SDK source**: Find the exact version in `go.mod` under `github.com/unkeyed/sdks/api/go/v2`, then read the SDK source from the Go module cache at `~/go/pkg/mod/github.com/unkeyed/sdks/api/go/v2@<version>/`. You MUST read the actual SDK files to verify struct names, field names, and field types. Do not guess.
3. **Existing code**: Read `cmd/api/util/` (shared helpers — do NOT modify), `cmd/api/root.go`, and any existing subpackages to understand the current state.

## Step 3: Group endpoints

Parse all paths from the OpenAPI spec. They look like `/v2/{group}.{action}`.

Group them by resource:
- `apis` — endpoints matching `apis.*`
- `keys` — endpoints matching `keys.*`
- `identities` — endpoints matching `identities.*`
- `permissions` — endpoints matching `permissions.*`
- `ratelimit` — endpoints matching `ratelimit.*`
- `analytics` — endpoints matching `analytics.*`

Skip `deploy.*` (already handled by `cmd/deploy`) and `liveness` (GET health check, not useful as CLI).

## Step 4: Directory structure

Each group is a subpackage under `cmd/api/`. Each command is its own file:

```
cmd/api/
├── root.go                        # Cmd, imports and registers group subpackages
├── util/
│   ├── client.go                  # CreateClient, APIAction
│   ├── output.go                  # Output
│   ├── errors.go                  # FormatError
│   └── flags.go                   # RootKeyFlag, APIURLFlag, ConfigFlag, OutputFlag
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

var Cmd = &cli.Command{
	Name:        "{group}",
	Usage:       "...",
	Description: "...",
	Commands: []*cli.Command{
		{leafCmd1},
		{leafCmd2},
	},
}
```

### Leaf command file pattern

Each leaf command lives in its own file named with the kebab-case action using underscores: `create_api.go`, `verify_key.go`, `list_overrides.go`.

The variable name is unexported: `createAPICmd`, `verifyKeyCmd`, `listOverridesCmd`.

### Register new groups

After creating a new group subpackage, import it in `cmd/api/root.go` and add `{group}.Cmd` to the `Commands` slice. Only add entries that don't already exist.

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

- **Command name**: Convert the action part of the operationId to kebab-case.
  - `createApi` → `create-api`
  - `listKeys` → `list-keys`
  - `verifyKey` → `verify-key`
  - `addPermissions` → `add-permissions`
  - `limit` → `limit`
  - `multiLimit` → `multi-limit`
  - `getVerifications` → `get-verifications`

- **File name**: kebab-case action with underscores: `create_api.go`, `verify_key.go`

- **Variable name**: camelCase with SDK acronym rules: `createAPICmd`, `verifyKeyCmd`

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

### Flag descriptions
Use a short, one-sentence summary of the OpenAPI property `description`. Since every command links to full docs, flag descriptions should be concise — just enough to know what the flag does. Do NOT copy the full multi-sentence OpenAPI description.

## Flag mapping

Every leaf command MUST include these four flags first:
```go
util.RootKeyFlag(),
util.APIURLFlag(),
util.ConfigFlag(),
util.OutputFlag(),
```

Then map OpenAPI request body properties to CLI flags:

| OpenAPI type | SDK Go type | CLI flag | Read value |
|---|---|---|---|
| `string` (required) | `string` | `cli.String("name", "desc", cli.Required())` | `cmd.String("name")` |
| `string` (optional) | `*string` | `cli.String("name", "desc")` | check non-empty, then `&v` |
| `integer` (required) | `int64` | `cli.Int64("name", "desc", cli.Required())` | `cmd.Int64("name")` |
| `integer` (optional) | `*int64` | `cli.Int64("name", "desc")` | check non-zero, then `&v` |
| `boolean` (optional) | `*bool` | `cli.Bool("name", "desc", cli.Default(X))` | `ptr.P(cmd.Bool("name"))` — use `pkg/ptr` for the pointer |

**Boolean flag defaults**: Look up `default:` in the OpenAPI spec. Use `cli.Default(true)` or `cli.Default(false)` accordingly. For partial-update endpoints where omitting a boolean means "don't change" (no `default:` in spec), do NOT set a default — use `cmd.FlagIsSet("name")` to check if the user explicitly passed it:
```go
if cmd.FlagIsSet("enabled") {
    req.Enabled = ptr.P(cmd.Bool("enabled"))
}
```
| `array of strings` | `[]string` | `cli.StringSlice("name", "desc")` | `cmd.StringSlice("name")` |
| `object` / `map` / nested | complex | `cli.String("name-json", "JSON: ...")` | `json.Unmarshal` |

**JSON flags**: For complex types exposed as `--*-json` flags, keep the flag description focused on what the field does. Show the JSON shape in the command's `Examples` field instead, with realistic values. Every command that has JSON flags MUST include at least one example showing their usage.

**Flag naming**: Convert camelCase property names to kebab-case: `apiId` → `api-id`, `externalId` → `external-id`.

## Action pattern

Every leaf command uses a plain `func(ctx, cmd) error` action. No wrappers — each command
explicitly creates the client, times the call, formats errors, and prints output:

```go
Action: func(ctx context.Context, cmd *cli.Command) error {
    client, err := util.CreateClient(cmd)
    if err != nil {
        return err
    }

    // Build request from flags
    req := components.V2ApisCreateAPIRequestBody{
        Name: cmd.String("name"),  // required: assign directly
    }

    // Optional string:
    if v := cmd.String("prefix"); v != "" {
        req.Prefix = &v
    }

    // Optional int64:
    if v := cmd.Int64("limit"); v != 0 {
        req.Limit = &v
    }

    // Optional bool — look up the default in the OpenAPI spec (check `default:` on the property).
    // Use cli.Bool with cli.Default and ptr.P from pkg/ptr:
    req.Enabled = ptr.P(cmd.Bool("enabled"))   // default: true in spec
    req.Decrypt = ptr.P(cmd.Bool("decrypt"))   // default: false in spec

    // For partial-update endpoints where omitting a bool means "don't change":
    if cmd.FlagIsSet("enabled") {
        req.Enabled = ptr.P(cmd.Bool("enabled"))
    }

    // String slice:
    if v := cmd.StringSlice("permissions"); len(v) > 0 {
        req.Permissions = v
    }

    // JSON field (complex object):
    if v := cmd.String("meta-json"); v != "" {
        var meta map[string]any
        if err := json.Unmarshal([]byte(v), &meta); err != nil {
            return fmt.Errorf("invalid JSON for --meta-json: %w", err)
        }
        req.Meta = meta
    }

    // Call SDK and handle errors
    start := time.Now()
    res, err := client.Apis.CreateAPI(ctx, req)
    if err != nil {
        return fmt.Errorf("%s", util.FormatError(err))
    }

    return util.Output(cmd, res.V2ApisCreateAPIResponseBody, time.Since(start))
},
```

## Reference implementation

See `cmd/api/apis/create_api.go` for a complete working example. Follow this pattern exactly for all new commands.

## Linter requirements

The project uses strict linters via bazel nogo. Watch out for:

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

For each command, generate a documentation page at `docs/product/cli/{group}/{command-name}.mdx`.

### Doc file structure

Each command gets its own `.mdx` file using Mintlify components:

```mdx
---
title: "create-api"
description: "Create an API namespace for organizing keys by environment, service, or product"
---

Copy the full description from the OpenAPI spec here (same as the CLI Description,
but keep markdown formatting since this is a docs page).

## Usage

```bash
unkey api apis create-api [flags]
```

## Flags

<ParamField body="--name" type="string" required>
Unique identifier for this API namespace within your workspace. Use descriptive names
like 'payment-service-prod' or 'user-api-dev' to clearly identify purpose and environment.
</ParamField>

<ParamField body="--enabled" type="bool" default="true">
Whether the key is active for verification.
</ParamField>

<ParamField body="--meta-json" type="JSON string">
Arbitrary JSON metadata returned during key verification.
</ParamField>

For JSON flags, use an Expandable to show the schema:

<ParamField body="--credits-json" type="JSON string">
Credit and refill configuration.
<Expandable title="JSON schema">
  <ResponseField name="remaining" type="integer" required>
  Number of credits remaining.
  </ResponseField>
  <ResponseField name="refill" type="object">
  Auto-refill configuration.
    <Expandable title="properties">
      <ResponseField name="interval" type="string" required>
      Refill interval: "daily" or "monthly".
      </ResponseField>
      <ResponseField name="amount" type="integer" required>
      Credits to add each interval.
      </ResponseField>
    </Expandable>
  </ResponseField>
</Expandable>
</ParamField>

## Global Flags

Include this exact section on every command doc:

| Flag | Type | Description |
|------|------|-------------|
| `--root-key` | string | Override root key (`$UNKEY_ROOT_KEY`) |
| `--api-url` | string | Override API base URL (default: `https://api.unkey.com`) |
| `--config` | string | Path to config file (default: `~/.unkey/config.toml`) |
| `--output` | string | Output format — use `json` for raw JSON |

## Examples

<CodeGroup>
```bash Basic
unkey api apis create-api --name=payment-service-prod
```
```bash With options
unkey api apis create-api --name=user-api-dev --output=json
```
```bash With JSON flags
unkey api keys create-key --api-id=api_123 --meta-json='{"plan":"pro"}'
```
</CodeGroup>
```

### Key rules for docs

- **Flag descriptions**: Use the FULL OpenAPI property description here (unlike CLI help which uses short summaries). This is the detailed reference.
- **JSON schemas**: Use `<Expandable>` with nested `<ResponseField>` to document the shape of JSON flags. Read the OpenAPI spec's `$ref` schemas to get the exact field definitions.
- **Required flags**: Use `required` attribute on `<ParamField>`.
- **Default values**: Use `default` attribute on `<ParamField>`.
- **Link to API reference**: Add a note at the bottom linking to the corresponding API endpoint page.

### Register in docs.json

**IMPORTANT**: Every new doc file MUST be registered in `docs/product/docs.json` or it won't appear in the documentation site. After creating doc files, read the current `docs.json`, find the CLI navigation group, and add any missing pages. The structure should be:

```json
{
  "group": "CLI",
  "icon": "terminal",
  "pages": [
    "cli/overview",
    {
      "group": "apis",
      "pages": [
        "cli/apis/create-api",
        "cli/apis/delete-api",
        "cli/apis/get-api",
        "cli/apis/list-keys"
      ]
    },
    {
      "group": "keys",
      "pages": [
        "cli/keys/create-key",
        ...
      ]
    }
  ]
}
```

### Audit mode docs and tests check

When running in `check` mode, also report:
- **Missing docs**: CLI commands with no corresponding `.mdx` file in `docs/product/cli/`
- **Missing from nav**: Doc files that exist but aren't registered in `docs.json`
- **Missing tests**: CLI command files with no corresponding `_test.go` file

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

### Types

- Import request types from `github.com/unkeyed/unkey/svc/api/openapi` (e.g., `openapi.V2KeysCreateKeyRequestBody`)
- Use `ptr.P()` from `github.com/unkeyed/unkey/pkg/ptr` for pointer fields
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
- **Missing tests**: CLI command files with no corresponding `_test.go` file

## Step 7: Verify

After generating, run:
```
make bazel && make build && make fmt
```
Fix any errors before finishing. Never use `go build` directly — always use `make build` (bazel), as it runs stricter linters.
