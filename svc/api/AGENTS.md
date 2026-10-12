# API

- In tenant-scoped handlers, use the authenticated principal's
  `AuthorizedWorkspaceID`, not a caller-supplied workspace ID. Get the principal
  from `zen.Session`; authentication and resource authorization are separate.
- A caller without read access must not distinguish a missing resource from
  an unreadable one. Match the not-found status, error type, title, and detail.
  When masking authorization failures, construct a fresh fault rather than
  wrapping the original: public messages in the fault chain can leak IDs.
  Use `MaskInsufficientPermissionsAsNotFound` in
  [internal/errors/errors.go](internal/errors/errors.go). The regression pattern
  is [403_existence_leak_test.go](routes/v2_apis_list_keys/403_existence_leak_test.go).
- The public schema source is [openapi/openapi-split.yaml](openapi/openapi-split.yaml)
  and [openapi/spec/](openapi/spec/), not `openapi-generated.yaml` or `gen.go`.
  Regenerate with `mise run generate` from the repository root.
