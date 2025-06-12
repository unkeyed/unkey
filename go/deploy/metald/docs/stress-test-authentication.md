# Stress Test Authentication

The metald stress test has been enhanced to support multi-tenant authentication testing, providing comprehensive validation of the authentication and authorization system.

## Authentication Features

### Multi-Tenant Token Pool
The stress test uses a pool of development customer tokens:
- `dev_customer_stress-test-1`
- `dev_customer_stress-test-2` 
- `dev_customer_stress-test-3`
- `dev_customer_stress-test-4`
- `dev_customer_stress-test-5`

### Customer Ownership Tracking
Each VM created during stress testing is associated with a specific customer token:
- VM operations (boot, stop, delete) use the same token that created the VM
- Ensures proper customer ownership validation
- Tests cross-tenant isolation security

## Usage

### Enable Authentication (Default)
```bash
# Run stress test with authentication enabled (default behavior)
./build/stress-test

# Explicitly enable authentication
./build/stress-test -auth=true
```

### Disable Authentication
```bash
# Run stress test without authentication headers (legacy mode)
./build/stress-test -auth=false
```

### Combined with Other Options
```bash
# Heavy load testing with authentication
./build/stress-test -heavy-load -auth=true -max-vms 1000

# Quick test without authentication
./build/stress-test -intervals 2 -max-vms 10 -auth=false
```

## Security Testing Benefits

### Customer Isolation Validation
- **VM Creation**: Each VM is assigned to a random customer from the token pool
- **VM Operations**: All subsequent operations use the owning customer's token
- **Cross-Tenant Protection**: Attempts to access VMs with wrong credentials are logged and blocked

### Multi-Tenant Load Testing
- **Concurrent Customers**: Multiple customer tokens are used simultaneously
- **Resource Distribution**: VMs are distributed across different customers
- **Authentication Overhead**: Tests the performance impact of authentication/authorization

## Log Output

### With Authentication Enabled
```
level=INFO msg="VM created" vm_id=vm-1234 customer_token=dev_customer_stress-test-2 duration=1.2s
level=INFO msg="VM booted" vm_id=vm-1234 duration=800ms
```

### Without Authentication
```
level=INFO msg="VM created" vm_id=vm-1234 duration=1.2s
level=INFO msg="VM booted" vm_id=vm-1234 duration=800ms
```

## Integration with metald Security

The stress test validates the complete authentication flow:

1. **Authentication Interceptor**: All requests include `Authorization: Bearer <token>` headers
2. **Customer Context**: Tokens are validated and customer context is extracted
3. **Ownership Validation**: VM operations verify customer ownership
4. **Database Isolation**: Each customer only sees their own VMs via `ListVms`

## Backwards Compatibility

The `-auth` flag defaults to `true` but can be disabled for:
- **Legacy Testing**: Test against metald instances without authentication
- **Performance Baseline**: Measure performance without authentication overhead
- **Development**: Quick testing during development

## Example Commands

### Production-Like Testing
```bash
# Test multi-tenant authentication with realistic load
./build/stress-test \
  -intervals 10 \
  -interval-duration 5m \
  -max-vms 50 \
  -auth=true
```

### Security Validation
```bash
# Focus on authentication security with multiple customers
./build/stress-test \
  -intervals 3 \
  -max-vms 20 \
  -iteration-duration 5s \
  -auth=true
```

### Performance Comparison
```bash
# Test with authentication
./build/stress-test -max-vms 100 -auth=true

# Test without authentication  
./build/stress-test -max-vms 100 -auth=false
```

This enhanced stress testing ensures that metald's multi-tenant security is thoroughly validated under realistic load conditions.