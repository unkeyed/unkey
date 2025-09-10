# Go Documentation Guidelines

This document outlines the documentation standards for our Go codebase.

## Core Principles

1. **Everything public MUST be documented** - No exceptions
2. **Internal code should explain "why", not "how"** - Focus on reasoning and trade-offs
3. **Be comprehensive and verbose** - Prefer thorough explanations over terse summaries
4. **Add substantial value** - Documentation should teach, not just restate the obvious
5. **Follow Go conventions** - Start with the item name, use present tense

## Documentation Philosophy

**Serve both beginners and experts.** Documentation should provide clear, accessible entry points for newcomers AND comprehensive details for experts who need to understand the full picture, including architectural decisions and edge cases.

**Clarity is better than terse.** We prefer comprehensive documentation that fully explains what the code does in detail, why it exists and its role in the system, how it relates to other components, what callers need to know about behavior and performance characteristics, when to use it versus alternatives, and what can go wrong and why.

**Every piece of documentation should add substantial value.** If the documentation doesn't teach something beyond what's obvious from the function signature, it needs to be expanded.

**Prioritize practical examples over theory.** Every non-trivial function should include working code examples that developers can copy and adapt. Examples should demonstrate real usage patterns, not artificial toy cases.

**Make functionality discoverable.** Use extensive cross-references to help developers find related functions and understand how pieces fit together. If a function works with or is an alternative to another function, mention that explicitly.

**Write in full sentences, not bullet points.** Code documentation should read like well-written prose that flows naturally. Avoid bullet points for general explanations, behavior descriptions, or conceptual information. Only use bullet points when they genuinely improve readability for specific lists such as error codes, configuration options, or step-by-step procedures. Most documentation should be written as coherent paragraphs that explain concepts thoroughly.

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

Every exported function and method must be documented. Focus on:
- What it does
- What parameters mean (especially if not obvious)
- What it returns
- Important behavior or side effects
- When it might fail

### Public Functions - Comprehensive Documentation

Every public function must be thoroughly documented. Here's what comprehensive documentation looks like:

```go
// Allow determines whether the specified identifier can perform the requested number
// of operations within the configured rate limit window.
//
// This method implements distributed rate limiting with strong consistency guarantees
// across all nodes in the cluster. It uses a lease-based algorithm to coordinate
// between nodes and ensure accurate rate limiting even under high concurrency.
//
// Parameters:
//   - identifier: A unique string identifying the entity being rate limited. This is
//     typically a user ID, API key, IP address, or other business identifier. The
//     identifier is used as the key for rate limit bucketing and should be stable
//     across requests from the same entity.
//   - cost: The number of operations being requested. For most use cases this is 1,
//     but can be higher for batch operations or when implementing weighted rate limiting.
//     Must be positive; zero or negative values will return an error.
//
// Behavior:
//   - Checks the current rate limit status for the identifier
//   - Coordinates with other cluster nodes if necessary to maintain consistency
//   - Updates the rate limit counters atomically if the request is allowed
//   - Implements fair queuing to prevent starvation under high load
//
// Returns:
//   - (true, nil): Request is allowed and counters have been updated
//   - (false, nil): Request is rate limited, no error condition
//   - (false, error): System error occurred, decision may be unreliable
//
// Error conditions (be specific about when each occurs):
//   - ErrInvalidCost: cost <= 0 or cost > MaxCost
//   - ErrClusterUnavailable: <50% of cluster nodes reachable
//   - context.DeadlineExceeded: operation timeout (default 5s)
//   - Network errors: underlying storage failures, retries exhausted
//
// Concurrency:
//   This method is safe for concurrent use from multiple goroutines. Internal
//   coordination ensures that concurrent requests for the same identifier are
//   handled correctly without race conditions.
//
// Context handling:
//   The context is used for request timeout and cancellation. If the context
//   is cancelled before the rate limit check completes, the method returns
//   the context error and no rate limit counters are modified.
//
// Context Guidelines:
//   - Always document timeout behavior and defaults
//   - Explain what happens on cancellation
//   - Mention if context values are used
func (r *RateLimiter) Allow(ctx context.Context, identifier string, cost int) (bool, error)
```

Compare this to insufficient documentation:
```go
// Allow checks if a request is allowed.
// Returns true if allowed, false if rate limited.
func (r *RateLimiter) Allow(ctx context.Context, identifier string, cost int) (bool, error)
```

The second example adds almost no value beyond the function signature and would be rejected.

### Function Documentation Approach

Write function documentation as natural, flowing prose that explains what actually matters for each specific function. Start with what the function does, then include whatever information is genuinely relevant and useful for callers. Some functions might need detailed parameter explanations, others might need performance notes, and simple functions might just need a clear explanation of their purpose. Don't force every function into the same template - let the function's complexity and use case guide what information to include.

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

Document all exported types, focusing on:
- What the type represents
- Its role in the system
- Important invariants or constraints
- Lifecycle considerations

### Structs
```go
// Config holds the configuration for a rate limiter instance.
//
// Window and Limit work together to define the rate limiting behavior.
// For example, Window=1m and Limit=100 means "100 operations per minute".
//
// ClusterNodes is required for distributed operation. For single-node
// deployments, use a slice with only the local node.
type Config struct {
    // Window is the time period over which operations are counted
    Window time.Duration

    // Limit is the maximum number of operations allowed within Window
    Limit int64

    // ClusterNodes lists all nodes participating in distributed rate limiting.
    // Must include at least the local node.
    ClusterNodes []string
}
```

### Interfaces
```go
// Cache provides a generic caching interface with support for distributed invalidation.
//
// Implementations must be safe for concurrent use. The cache may return stale data
// during network partitions to maintain availability, but will eventually converge
// when connectivity is restored.
//
// We chose this interface design over more specific cache types because our
// use cases vary widely (small config objects vs large binary data), and
// the generic approach allows for better testing and modularity.
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

Document error conditions and types:

```go
var (
    // ErrRateLimited is returned when an operation exceeds the configured rate limit.
    // This is expected behavior, not a system error.
    ErrRateLimited = errors.New("rate limit exceeded")

    // ErrClusterUnavailable indicates that the required number of cluster nodes
    // are not reachable. Operations may still succeed if configured to fail-open.
    ErrClusterUnavailable = errors.New("insufficient cluster nodes available")
)

// ProcessRequest handles incoming rate limit requests.
//
// Returns ErrRateLimited if the request exceeds the configured limits.
// Returns ErrClusterUnavailable if distributed consensus cannot be achieved.
// Other errors indicate system problems (network, storage, etc.).
func ProcessRequest(ctx context.Context, req *Request) (*Response, error)
```

## Constants and Variables

Document the purpose and valid values:

```go
const (
    // DefaultWindow is the standard rate limiting window for new limiters.
    // Chosen as a balance between memory usage and granularity for most use cases.
    DefaultWindow = time.Minute

    // MaxBurstRatio determines how much bursting is allowed above the base rate.
    // Set to 1.5 based on analysis of traffic patterns in production.
    MaxBurstRatio = 1.5
)

var (
    // GlobalRegistry tracks all active rate limiters for monitoring and cleanup.
    // We use a global registry instead of dependency injection here because
    // rate limiters need to be accessible from signal handlers for graceful shutdown.
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

- [ ] **Every package has a dedicated `doc.go` file** with comprehensive package documentation
- [ ] Every exported function, method, type, constant, and variable is documented
- [ ] Package documentation in `doc.go` explains purpose, key concepts but not details
- [ ] Internal code explains "why" decisions were made, not just "what" it does
- [ ] Error conditions and return values are clearly explained
- [ ] Complex algorithms include reasoning for the chosen approach
- [ ] Examples are provided for non-trivial usage patterns
- [ ] Documentation starts with the item name and uses present tense
- [ ] All documentation follows Go formatting conventions (proper line breaks, etc.)
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
