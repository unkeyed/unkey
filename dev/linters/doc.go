// Package linters provides custom linter wrappers for use with Bazel's nogo framework.
//
// Bazel's nogo runs static analysis at build time but lacks built-in support for
// //nolint directives that golangci-lint users expect. This package bridges that gap
// by wrapping standard Go analyzers with suppression handling.
//
// # Why Custom Wrappers?
//
// When migrating from golangci-lint to nogo, we faced a choice: remove all //nolint
// comments from the codebase or teach nogo to respect them. The wrappers here take
// the second approach, preserving existing suppressions and maintaining compatibility
// with developers' mental model of how linter directives work.
//
// # Package Overview
//
// Each sub-package wraps a specific linter:
//
//   - [errcheck]: Detects unchecked error returns. Skips generated files.
//   - [exhaustive]: Ensures switch statements cover all enum values.
//   - [exhaustruct]: Requires all struct fields to be initialized explicitly.
//   - [govet]: Bundles all standard go vet checks into a single analyzer.
//   - [ineffassign]: Finds assignments to variables that are never read.
//   - [nolint]: The wrapper infrastructure used by all other packages.
//   - [reassign]: Catches reassignment of package-level variables.
//   - [unused]: Identifies dead code that can be removed.
//
// # Usage
//
// These analyzers are configured in the nogo.json file at the repository root and
// run automatically during `bazel build`. No manual invocation is needed.
//
// To suppress a specific linter:
//
//	//nolint:exhaustruct
//	cfg := Config{Name: "partial"} // only some fields set intentionally
//
// To suppress all linters on a line:
//
//	//nolint
//	_ = riskyOperation() // error intentionally ignored
package linters
