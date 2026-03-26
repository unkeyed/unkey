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

## Step 5: Verify

After generating, run:
```
make bazel && make build && make fmt
```
Fix any errors before finishing. Never use `go build` directly — always use `make build` (bazel), as it runs stricter linters.
