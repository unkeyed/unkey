# Tools Management

This project uses Go 1.24+'s native `tool` directive to manage development tools like `buf`, `golangci-lint`, `sqlc`, `oapi-codegen` and `protoc-gen-go-restate`.

## Architecture

### Separate Tools Module (`tools/go.mod`)

We maintain a completely separate Go module for tools:

```
repo/
├── go.mod              # Main project dependencies
└── tools/
    ├── go.mod          # Tool dependencies (isolated)
    └── go.sum
```

**Why separate?**

- **Dependency isolation** - Tools often have heavy dependencies (linters pull in analysis libraries, buf needs protobuf tooling, etc.). We don't want these bleeding into our production code.
- **Different version requirements** - A tool might need an older version of a shared dependency that conflicts with our main module.
- **Cleaner main module** - Our primary `go.mod` only contains actual runtime and test dependencies.
- **Easier auditing** - Security scanning and dependency reviews focus on what actually ships, not build tools.

### Go Workspace (`go.work`)

The workspace ties the main module and tools module together:

```go
go 1.25.1

use (
    .       // Main module
    ./tools // Tools module
)
```

**Why a workspace?**

- **Unified tool access** - Run `go tool <name>` from anywhere in the repo without specifying which module
- **No `-modfile`** - Without workspace, you'd need `go tool -modfile=tools/go.mod <name>` every time
- **Multi-module development** - If you're working on both the main code and tool configurations, the workspace makes it seamless

## Setup

First-time setup:

```sh
make install  # Creates workspace and installs tools
```

Or the workspace is auto-created when you run any tool-dependent command:

```sh
make fmt       # Auto-creates workspace if needed
make generate  # Auto-creates workspace if needed
```

## Usage

Run tools via `go tool`:

```sh
go tool buf generate
go tool golangci-lint run
go tool sqlc generate
go tool protoc-gen-go-restate --version
```

## Updating Tools

To update a tool:

```sh
cd tools
go get -tool github.com/bufbuild/buf/cmd/buf@v1.60.0
cd ..
```

The new version is locked in `tools/go.mod` and `tools/go.sum`, ensuring everyone uses the same version.

## Maintenance Guide

### Adding a New Tool

```sh
cd tools
go get -tool github.com/example/tool/cmd/tool@v1.2.3
```

### Critical: Always Use `go tool` Prefix

**❌ Wrong - uses globally installed version:**

```go
//go:generate sqlc generate
//go:generate buf generate
```

```makefile
generate:
 sqlc generate
 buf generate
```

**✅ Correct - uses version from tools/go.mod:**

```go
//go:generate go tool sqlc generate
//go:generate go tool buf generate
```

```makefile
generate:
 go tool sqlc generate
 go tool buf generate
```

### How to Verify You're Using the Right Version

```sh
# Check what version will be used
go tool sqlc version        # Uses tools/go.mod version
sqlc version                # Uses global install
```

### Common Mistakes to Avoid

1. **Forgetting `go tool` prefix in `//go:generate` directives**

   - Always use `go tool <name>` instead of just `<name>`
   - Search codebase: `grep -r "//go:generate" --include="*.go"` and verify all tool calls have `go tool` prefix

2. **Running tools directly from command line**

   - Use `go tool sqlc generate` not `sqlc generate`
   - Update your muscle memory and shell aliases

3. **Mixing global and managed tools**

   - Don't rely on globally installed versions
   - Consider uninstalling global tools to catch mistakes: `rm $(which sqlc)`

4. **Forgetting to commit `tools/go.sum`**
   - Always commit both `tools/go.mod` and `tools/go.sum`
   - They work together to ensure reproducible builds
