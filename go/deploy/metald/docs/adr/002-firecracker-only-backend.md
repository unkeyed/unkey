# ADR-002: Firecracker as the Only Supported Backend

## Status
Accepted

## Context
The codebase includes interfaces and partial implementations for both Firecracker and Cloud Hypervisor backends. However, only Firecracker is fully implemented and tested.

## Decision
We only support Firecracker as the VM backend. The Cloud Hypervisor code exists but is incomplete and should not be used.

## Consequences

### Positive
- Clear focus on one well-tested backend
- Simplified testing and maintenance
- Firecracker is production-proven at scale

### Negative
- No backend diversity
- Locked into Firecracker's feature set

### Future Considerations
If we need Cloud Hypervisor support:
1. Complete the implementation in `internal/backend/cloudhypervisor/`
2. Add comprehensive tests
3. Update the configuration validation to allow it
4. Document the differences and use cases