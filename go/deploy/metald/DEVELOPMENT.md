# Metald Development Guide

## Quick Start

```bash
# Build metald
make build

# Run with development auth
./build/metald

# Test with curl
curl -X POST http://localhost:8080/v1/vms \
  -H "Authorization: Bearer dev_customer_test" \
  -H "Content-Type: application/json" \
  -d '{"config": {"cpu": {"vcpu_count": 2}, "memory": {"size_bytes": 1073741824}}}'
```

## Key Concepts

### 1. Integrated Jailer
We use an integrated jailer instead of the external Firecracker jailer binary. This solves network namespace issues. See `internal/jailer/README.md` for details.

### 2. Authentication
Development uses mock auth tokens like `Bearer dev_customer_123`. In production, this should be replaced with real authentication. See `docs/development/authentication.md`.

### 3. Backend Support
Only Firecracker is supported. CloudHypervisor code exists but is incomplete and should not be used.

### 4. Network Device Names
Linux limits network interface names to 15 characters. We use 8-character IDs with prefixes like `tap_`, `vh_`, `ns_vm_`.

## Common Pitfalls

1. **Don't use CloudHypervisor backend** - It's not implemented
2. **Don't use development auth in production** - It has no security
3. **Don't modify the integrated jailer** without understanding the security implications
4. **Network namespaces require root or CAP_NET_ADMIN** - Use `make install` to set capabilities

## Architecture Decisions

See `docs/adr/` for important architecture decisions:
- ADR-001: Why we use integrated jailer
- ADR-002: Why only Firecracker is supported

## Testing

```bash
# Unit tests
make test

# Integration tests (requires root/capabilities)
sudo make test-integration
```

## Debugging

Enable debug logging:
```bash
UNKEY_METALD_LOG_LEVEL=debug ./build/metald
```

Common issues:
- "Permission denied" - Run `make install` to set capabilities
- "Device not found" - Network namespace issue, check integrated jailer logs
- "Backend not supported" - Only Firecracker works, not CloudHypervisor