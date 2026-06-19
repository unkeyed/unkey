# Upsert WorkOS permissions

This tool syncs Unkey's WorkOS permission definitions into a WorkOS
environment.

The source of truth is
[`pkg/auth/workos/permissions.go`](../../pkg/auth/workos/permissions.go).
The tool creates missing WorkOS permissions and updates the name and
description for permissions that already exist.

It never deletes WorkOS permissions. If WorkOS contains a permission that is not
defined in `pkg/auth/workos/permissions.go`, the tool prints it as unmanaged so
you can review it manually.

## Usage

Preview the permissions without calling WorkOS:

```bash
mise exec -- bazel run //tools/upsert-workos-permissions -- -dry-run
```

Sync permissions into WorkOS:

```bash
WORKOS_API_KEY=sk_... \
  mise exec -- bazel run //tools/upsert-workos-permissions
```

You can also pass the key as a flag:

```bash
mise exec -- bazel run //tools/upsert-workos-permissions -- \
  -api-key sk_...
```

## Output

The tool prints one line per managed permission:

```text
updated projects:read
created keys:create
```

If WorkOS has permissions that Unkey no longer defines, the tool prints them
after the upsert pass:

```text
unmanaged legacy:permission
```

Unmanaged permissions are not deleted.
