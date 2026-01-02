# Go Documentation Guidelines

This document outlines the documentation standards for our Go codebase.

## Core Principles

1. **Everything public MUST be documented** - No exceptions
2. **Internal code should explain "why", not "how"** - Focus on reasoning and trade-offs
3. **Document what matters** - Focus on non-obvious behavior, constraints, and context that helps users succeed
4. **Add substantial value** - Documentation should teach, not just restate the obvious
5. **Follow Go conventions** - Start with the item name, use present tense

## Documentation Philosophy

**Match documentation depth to complexity.** Simple functions need simple documentation. Complex functions with edge cases, performance implications, or subtle behavior need detailed documentation. Don't force every function into the same verbose template.

**Focus on what users need to know.** Document the "what" (what does it do), the "when" (when should I use it), the "watch out" (what can go wrong), and the "why" (why this design). Skip sections that don't apply. Most simple functions only need "what" and maybe "watch out."

**Every piece of documentation should add substantial value.** If the documentation doesn't teach something beyond what's obvious from the function signature, it's sufficient to just explain the purpose clearly. Not every function needs parameter explanations, error details, performance notes, and concurrency information.

**Don't document irrelevant details.** If a function has no special concurrency considerations, don't document that "it's safe for concurrent use" unless the type is designed for concurrent access. If performance is O(1) and unremarkable, don't document it. If context handling is standard, don't explain it.

**Prioritize practical examples for non-trivial usage.** Simple getters, setters, and straightforward functions don't need examples. Focus examples on complex workflows, non-obvious usage patterns, and common integration scenarios.

**Make functionality discoverable.** Use cross-references to help developers find related functions and understand how pieces fit together. If a function works with or is an alternative to another function, mention that explicitly.

**Write naturally.** Use prose for explanations. Use lists (bullet points, parameter lists, error lists) only when they genuinely improve readability for enumerable items. Don't force structured sections into every docstring.

## Package-Level Documentation

**Every package MUST have a dedicated `doc.go` file** containing only package documentation and the package declaration. Do not put package documentation in random `.go` files above the package declaration.

The `doc.go` file should explain what the package does, why it exists, how it fits into the larger system, key concepts and terminology, basic usage examples, and cross-references to key types and functions.

### Format - doc.go File

Create a `doc.go` file in each package with this structure:

```go
// Package ratelimit implements distributed rate limiting with lease-based coordination.
//
// The package uses a two-phase commit protocol to ensure consistency across
// multiple nodes in a cluster. Rate limits are enforced through sliding time
// windows with configurable burst allowances.
//
// This implementation was chosen over simpler approaches because we need
// strong consistency guarantees for billing and security use cases.
//
// # Key Types
//
// The main entry point is [RateLimiter], which provides the [RateLimiter.Allow]
// method for checking rate limits. Configuration is handled through [Config].
//
// # Usage
//
// Basic rate limiting:
//
//   cfg := ratelimit.Config{Window: time.Minute, Limit: 100}
//   limiter := ratelimit.New(cfg)
//   allowed, err := limiter.Allow(ctx, "user:123", 1)
//   if err != nil {
//       // Handle system error
//   }
//   if !allowed {
//       // Rate limited - reject request
//   }
//
// For advanced configuration and cluster setup, see the examples in the
// /examples directory.
//
// # Error Handling
//
// The package distinguishes between rate limiting (expected behavior) and
// system errors (unexpected failures). See [ErrRateLimited] and [ErrClusterUnavailable]
// for the main error types.
package ratelimit
```

### Requirements for doc.go Files

- **File name**: Must be exactly `doc.go` in the package root
- **Content**: Only package documentation and package declaration - no other code
- **Format**: Start with "Package [name] [verb]..."
- **Structure**: Use `#` headers to organize sections (Key Types, Usage, Error Handling, etc.)
- **Purpose**: Explain what the package does and why it exists
- **Architecture**: Include reasoning for non-trivial design decisions
- **Examples**: Provide complete, runnable code examples
- **Cross-references**: Use `[TypeName]` and `[FunctionName]` format extensively
- **Related packages**: Reference external dependencies and related internal packages

## Function and Method Documentation

Every exported function and method must be documented. Start with what it does. Then add details only if they're non-obvious or important:

- Parameter meanings (only if not clear from name/type)
- Return value specifics (only if behavior is subtle)
- Important side effects or behavioral quirks
- Error conditions (only if non-standard or important to handle)
- Performance or concurrency notes (only if they significantly impact usage)

### Simple Functions - Minimal Documentation

Most functions are straightforward and need only a clear explanation:

```go
// GetUserID extracts the user ID from the request context.
// Returns an empty string if no user ID is present in the context.
func GetUserID(ctx context.Context) string

// Close releases all resources held by the client, including network connections
// and background goroutines. After calling Close, the client must not be used.
func (c *Client) Close() error

// SetTimeout updates the request timeout duration for all future requests.
func (c *Client) SetTimeout(d time.Duration)
```

### Complex Functions - Detailed Documentation

Complex functions with edge cases, distributed behavior, or subtle semantics need thorough documentation:

```go
// Allow determines whether the specified identifier can perform the requested number
// of operations within the configured rate limit window.
//
// This method implements distributed rate limiting with strong consistency guarantees
// across all nodes in the cluster. It uses a lease-based algorithm to coordinate
// between nodes and ensure accurate rate limiting even under high concurrency.
//
// The identifier should be a stable business identifier (user ID, API key, IP address).
// The cost is typically 1 for single operations, but can be higher for batch requests.
// Cost must be positive or an error is returned.
//
// Returns (true, nil) if allowed, (false, nil) if rate limited, or (false, error)
// if a system error occurs. Possible errors include ErrInvalidCost for invalid cost
// values, ErrClusterUnavailable when <50% of cluster nodes are reachable,
// context.DeadlineExceeded on timeout (default 5s), and network errors on storage failures.
//
// This method is safe for concurrent use. Context is used for timeout and cancellation;
// if cancelled, no rate limit counters are modified.
func (r *RateLimiter) Allow(ctx context.Context, identifier string, cost int) (bool, error)
```

The documentation is detailed because this function has distributed behavior, multiple error conditions,
and important concurrency guarantees. Compare to a simpler alternative that would be insufficient:

```go
// Allow checks if a request is allowed.
func (r *RateLimiter) Allow(ctx context.Context, identifier string, cost int) (bool, error)
```

This doesn't explain the distributed coordination, error conditions, or the meaning of the bool return
vs error return, so it would be insufficient.

### When to Include Specific Details

**Parameters**: Document them when the purpose isn't obvious from the name and type, or when there
are constraints (e.g., "must be positive", "should be stable across calls").

**Return values**: Explain if the return pattern is subtle (e.g., bool success + separate error),
or if there are multiple success states.

**Error conditions**: List specific errors only when callers need to handle them differently, or
when they're not obvious from context. Generic "returns error on failure" is usually sufficient.

**Concurrency**: Only document if the function/type is designed for concurrent use OR if it
explicitly must not be used concurrently. Don't document for simple stateless functions.

**Performance**: Only mention if there are non-obvious performance characteristics that affect
usage decisions (e.g., "O(n²) - use [AlternativeFunc] for large inputs" or "blocks until response").

**Context**: Only document context behavior if it's non-standard (e.g., uses context values,
has specific timeout behavior, or has special cancellation semantics).

### Internal Functions (Focus on "Why")

```go
// retryWithBackoff handles retries for failed lease acquisitions.
//
// We use exponential backoff with jitter instead of linear backoff because
// under high load, linear backoff causes thundering herd problems when many
// clients retry simultaneously. The exponential approach with randomization
// spreads out retry attempts and reduces system load.
//
// Max retry count is limited to prevent infinite loops during system outages.
func (r *RateLimiter) retryWithBackoff(ctx context.Context, fn func() error) error
```

## Type Documentation

Document what the type represents and any non-obvious aspects like invariants, constraints,
or lifecycle requirements. Document struct fields only when their purpose isn't clear from
the name and type.

### Structs

Simple config structs with self-explanatory fields need minimal documentation:

```go
// Config holds rate limiter configuration.
type Config struct {
    // Window is the time period over which the rate limit applies.
    Window time.Duration

    // Limit is the maximum number of operations allowed within Window.
    Limit  int64
}
```

Add more context when there are constraints, relationships, or non-obvious semantics:

```go
// Config holds the configuration for a rate limiter instance.
//
// Window and Limit work together to define rate limiting behavior.
// For example, Window=1m and Limit=100 means "100 operations per minute".
type Config struct {
    Window time.Duration
    Limit  int64

    // ClusterNodes lists all nodes in the cluster. Required for distributed
    // operation; for single-node deployments, include only the local node.
    ClusterNodes []string
}
```

### Interfaces

Document the interface's purpose and any important implementation requirements like
concurrency safety, guarantees, or trade-offs:

```go
// Cache provides a generic caching interface with support for distributed invalidation.
//
// Implementations must be safe for concurrent use. The cache may return stale data
// during network partitions to maintain availability, but will eventually converge
// when connectivity is restored.
type Cache[T any] interface {
    // Get retrieves a value by key. Returns the value and whether it was found.
    // A cache miss (found=false) is not an error.
    Get(ctx context.Context, key string) (value T, found bool, err error)

    // Set stores a value. The value will be replicated to other cache nodes
    // asynchronously. Use SetSync if you need immediate consistency.
    Set(ctx context.Context, key string, value T) error
}
```

## Error Documentation

Document sentinel errors with what they mean and when they occur:

```go
var (
    // ErrRateLimited is returned when an operation exceeds the configured rate limit.
    ErrRateLimited = errors.New("rate limit exceeded")

    // ErrClusterUnavailable indicates that insufficient cluster nodes are reachable.
    ErrClusterUnavailable = errors.New("insufficient cluster nodes available")
)
```

Only list specific error conditions in function docs when callers need to handle them differently:

```go
// ProcessRequest handles incoming rate limit requests.
//
// Returns ErrRateLimited if the request exceeds configured limits, ErrClusterUnavailable
// if distributed consensus cannot be achieved, or other errors for system problems.
func ProcessRequest(ctx context.Context, req *Request) (*Response, error)
```

## Constants and Variables

Document the purpose. Add reasoning only for non-obvious design choices:

```go
const (
    // DefaultWindow is the standard rate limiting window.
    DefaultWindow = time.Minute

    // MaxBurstRatio determines how much bursting is allowed above the base rate.
    MaxBurstRatio = 1.5
)

var (
    // GlobalRegistry tracks all active rate limiters for monitoring and cleanup.
    GlobalRegistry = &Registry{limiters: make(map[string]*RateLimiter)}
)
```

## Complex Algorithm Documentation

For complex internal logic, explain the approach and reasoning:

```go
// distributeTokens implements the token bucket algorithm with cluster coordination.
//
// We chose token bucket over sliding window because:
// 1. Better burst handling for API use cases
// 2. Simpler mathematics for distributed scenarios
// 3. More predictable memory usage
//
// The algorithm works in two phases:
// 1. Local calculation of available tokens
// 2. Cluster consensus on token allocation
//
// Phase 2 is optimized away when the local node has sufficient tokens,
// reducing latency for the common case.
func (r *RateLimiter) distributeTokens(ctx context.Context, required int64) (granted int64, err error) {
    // Local fast path - no cluster coordination needed
    if r.localTokens.Load() >= required {
        // ... implementation
    }

    // Cluster coordination required
    // We use Raft consensus here instead of eventual consistency because
    // rate limiting must be strictly enforced for security and billing
    // ... implementation
}
```

## Examples and Usage

Include examples for non-trivial usage:

```go
// Example_basicUsage demonstrates typical rate limiter setup and usage.
func Example_basicUsage() {
    cfg := Config{
        Window: time.Minute,
        Limit:  1000,
        ClusterNodes: []string{"localhost:8080"},
    }

    limiter, err := New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer limiter.Close()

    // Check if user can make 5 API calls
    allowed, err := limiter.Allow(context.Background(), "user:alice", 5)
    if err != nil {
        log.Printf("System error: %v", err)
        return
    }

    if !allowed {
        log.Println("Rate limit exceeded")
        return
    }

    log.Println("Request allowed")
    // Output: Request allowed
}
```

## Testing Documentation

Document test helpers and complex test scenarios:

```go
// newTestLimiter creates a rate limiter configured for testing.
//
// Uses in-memory storage and shorter time windows to speed up tests.
// Not suitable for production use due to lack of persistence.
func newTestLimiter(t *testing.T, limit int64) *RateLimiter {
    // ... implementation
}

// TestConcurrentAccess verifies that the rate limiter maintains accuracy
// under high concurrency.
//
// This test is critical because our production workload often has hundreds
// of goroutines hitting the same rate limiter simultaneously.
func TestConcurrentAccess(t *testing.T) {
    // ... implementation
}
```

## Consistency and Style

**Terminology must be consistent** across the entire codebase:

- Use the same terms for the same concepts (e.g., always "identifier", never mix with "key" or "ID")
- Define domain-specific terms in package documentation
- Create a glossary for complex domains

**Parameter naming should be predictable:**

- `ctx context.Context` (always first parameter)
- `id string` or `identifier string` for rate limit keys
- `cost int` or `count int` for operation quantities

## Go Documentation Conventions

Follow these formatting and style conventions:

1. **Active voice and clear, concise explanations**
2. Use `//` for function comments, `/* */` only for package-level overviews
3. **Present tense** ("Returns..." not "Will return...")
4. **Omit redundant phrases** like "This function..." - go straight to the verb
5. **Document parameters by name** without quotes
6. **Start sentences with capital letters and end with periods**
7. **Reference RFC or standards** when implementing them
8. **Document side effects** or mutating behavior
9. **Self-contained documentation** - provide all necessary information
10. **Cross-references** using Go's `[Reference]` format:
    - `[OtherFunc]` for functions
    - `[TypeName]` for structs/interfaces
    - `[ConstantName]` for constants

## Best Practices and Anti-Patterns

**Highlight non-obvious behaviors and edge cases** by documenting nil input handling, concurrency hazards, silent failures, performance bottlenecks, scalability concerns, and conditions where functions behave unexpectedly.

**Document what NOT to do** with specific examples:

```go
// Allow checks rate limits for the given identifier.
//
// IMPORTANT: Do not call Allow() in a loop without backoff - this can
// overwhelm the system. Instead use:
//
//   // Bad:
//   for !limiter.Allow(ctx, id, 1) { /* busy wait */ }
//
//   // Good:
//   if allowed, err := limiter.Allow(ctx, id, 1); !allowed {
//       return ErrRateLimited
//   }
```

**Examples should be high-quality and idiomatic.** Follow Go best practices including proper `defer` usage, preferring slices over arrays, using realistic data and real-world scenarios, showing both correct usage and common pitfalls, following Go naming conventions such as `err` for errors and `ctx` for contexts, and formatting using Go's `ExampleFunc` style for `godoc`.

## Documentation Checklist

Before submitting code, verify:

- [ ] **Every package has a dedicated `doc.go` file** with package documentation
- [ ] Every exported function, method, type, constant, and variable is documented
- [ ] Documentation depth matches code complexity (simple code = simple docs)
- [ ] Only relevant details are included (skip irrelevant concurrency, performance, context notes)
- [ ] Internal code explains "why" decisions were made, not just "what" it does
- [ ] Error conditions are mentioned when callers need to handle them differently
- [ ] Complex algorithms include reasoning for the chosen approach
- [ ] Examples are provided for non-trivial usage patterns
- [ ] Documentation starts with the item name and uses present tense
- [ ] All documentation follows Go formatting conventions
- [ ] Cross-references use proper `[Reference]` format
- [ ] Edge cases and non-obvious behaviors are documented
- [ ] Concurrency guarantees are documented if and only if the code is designed to be concurrently safe

## Deprecation and Breaking Changes

When deprecating APIs, provide clear migration paths:

```go
// Deprecated: Use NewRateLimiterV2 instead. This function will be removed in v2.0.
//
// Migration example:
//   // Old:
//   limiter := NewRateLimiter(100, time.Minute)
//
//   // New:
//   limiter := NewRateLimiterV2(Config{Limit: 100, Window: time.Minute})
func NewRateLimiter(limit int, window time.Duration) *RateLimiter
```

## Tools and Validation

Use these tools to validate documentation:

```bash
# Check for missing documentation
go vet ./...

# Generate and review documentation
godoc -http=:6060

# Check documentation formatting
go fmt ./...
```

Remember: Good documentation is an investment in your future self and your teammates. Take the time to write it well.
