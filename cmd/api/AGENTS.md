# API resource commands

Read `.claude/skills/gen-api-cli/SKILL.md` before adding or regenerating an
endpoint. Use `svc/api/openapi/openapi-generated.yaml` and the Go SDK version
pinned in `go.mod` as the request contract. Do not upgrade the SDK or edit the
skill's bundled specification as part of routine command generation.

## Shared command handling

Canonical paths in `resources.go` and legacy `unkey api` paths reuse fresh
endpoint constructors. Keep request construction, typed `--body` handling,
destructive confirmation, errors, and response rendering shared. Use
`util.Output` for the complete SDK response body. Do not add leaf-specific
renderers or prompt loops.

`ResourceCommands` attaches `internal/guidance` only to canonical commands.
Guidance runs before required-flag validation and sends its result through the
existing typed body path. Preserve these boundaries:

- Start automatic forms only when required input is missing or an update has
  a target but no changes. Require both stdin and stderr to be terminals.
- Use `--interactive` for optional settings and list or search filters, even
  when required inputs are already supplied.
- Do not guide legacy `unkey api` commands, explicit `--body` requests,
  `--no-input`, or `--interactive=false`. Nonterminal calls never prompt.
  An explicit `--interactive` without both terminals fails.
- Write prompts and guidance diagnostics to stderr. Redirecting stdout or
  selecting JSON does not disable guidance. Cancellation must not send the
  endpoint request.
- Preserve explicitly supplied flags. Use `Flag.IsSet()` or
  `cmd.FlagIsSet(...)`, never nonzero or nonempty checks to infer presence.
  Keep update omission, `null`, `false`, `0`, and `[]` distinct through SDK
  serialization. Do not fill omitted update fields from current resource data
  or create defaults.
- Keep `--yes` separate from input collection. Guided delete and remove
  commands still use `util.Confirm` before sending the destructive request.
- Confirm guided actions with their action label, Back, and Cancel. Show field
  editing only after Back. Read-only commands run after filter collection.
  Hide secrets and arbitrary JSON in previews, and preserve sibling values
  when editing a field or list item.
- Offer correction only for typed `invalid_input` 400 responses with field
  errors. Keep answers in memory, require edits and explicit resubmission,
  and never automatically retry failed resource requests.

Keep error rendering at `Command.OnError`, after input preparation and action
wrappers. Wrappers must receive original typed errors. Preserve structured
flag diagnostics and support `--error-format=json` even after a parse failure.

Resource browsing belongs in `internal/guidance/picker.go`. Use the pinned
SDK, preserve parent scope, and send search and pagination to the API only
where the list endpoint supports them. Always offer manual IDs without first
requiring list permission. Do not display key secrets in resource choices.

## Regenerate metadata

After changing endpoint mappings, request schemas, or metadata extraction,
regenerate the embedded input definitions from the repository root:

```bash
mise exec -- go generate ./cmd/api/internal/guidance
mise exec -- go generate ./cmd/api/internal/discovery
```

`tools/cliguidance` reads `resourcePaths` and the authoritative OpenAPI spec.
Keep endpoint-version overrides aligned with the SDK method each command
calls. Its output, `internal/guidance/schemas_generated.json`, has one compact
endpoint per line. Do not hand-edit it, bulk pretty-print it, or embed the full
OpenAPI specification.

`unkey describe` uses separate metadata in `internal/discovery`. Preserve full
request schemas and their referenced definitions for agents, not the reduced
form schemas. Discovery must stay offline and omit runtime flag values and
credentials. Its catalogue covers canonical API endpoints, not orchestration.

## Verify endpoint changes

Add request-capture tests for the endpoint and extend the shared guidance
contract tests in `guidance_test.go`. Use `guidance_terminal_linux_test.go`
for focused PTY tests of new interactive behavior, not only mocked prompts.
Cover supplied flags, omitted updates and explicit empty values, cancellation
without mutation, stdout/stderr separation, and nonterminal, body, and legacy
paths. For picker changes, cover search, pagination, parent scope, and manual
fallback when listing is denied.

```bash
mise exec -- rask ./tools/cliguidance ./cmd/api/...
```

The generator's `TestGeneratedSchemasCurrent` checks metadata drift and the
compact layout. Keep `TestEveryResourceHasGuidance` passing for new endpoints.
Follow the root build requirements for Go changes. Document guided behavior
in `docs/platform/cli/output-and-flags.mdx` without changing script contracts.
