# ADR-001: Integrated Jailer Instead of External Binary

## Status
Accepted

## Context
Firecracker provides an external `jailer` binary for security isolation. However, we encountered a critical issue where the external jailer would create TAP network devices outside the network namespace, causing "device not found" errors when Firecracker tried to use them inside the namespace.

## Decision
We implemented jailer functionality directly within metald as an integrated component.

## Consequences

### Positive
- Full control over the order of operations (namespace entry before TAP creation)
- Better error handling and debugging
- No external binary dependencies
- Integrated with our observability stack
- Solves the network device visibility issue

### Negative
- We maintain security-critical code ourselves
- Divergence from Firecracker's recommended approach
- Need to keep up with security best practices

### Neutral
- Same security isolation model as external jailer
- Same capability requirements