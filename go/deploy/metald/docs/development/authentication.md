# Authentication in Metald

## Current State

Metald currently uses a **development-only** authentication mechanism that accepts tokens in the format:
```
Bearer dev_customer_<customer_id>
```

For example:
```bash
curl -H "Authorization: Bearer dev_customer_123" http://localhost:8080/v1/vms
```

## Why is it like this?

The authentication is stubbed out because metald is designed to run behind a gateway that handles real authentication. The customer context is expected to be injected by the gateway layer.

## Production Authentication

In production, you should:

1. **Use a Gateway**: Deploy metald behind an API gateway (Kong, Envoy, etc.) that handles authentication
2. **Forward Context**: Have the gateway forward authenticated customer context in headers or tokens
3. **Validate in Metald**: Update `internal/service/auth.go` to validate tokens from your auth system

## Implementing Real Authentication

To implement real authentication, modify `internal/service/auth.go`:

```go
func validateToken(ctx context.Context, token string) (*CustomerContext, error) {
    // Example: Validate JWT
    claims, err := validateJWT(token)
    if err != nil {
        return nil, err
    }
    
    return &CustomerContext{
        CustomerID:  claims.CustomerID,
        TenantID:    claims.TenantID,
        UserID:      claims.UserID,
        WorkspaceID: claims.WorkspaceID,
    }, nil
}
```

## Testing with Development Auth

For local development and testing:

```bash
# Create a VM for customer "test123"
curl -X POST http://localhost:8080/v1/vms \
  -H "Authorization: Bearer dev_customer_test123" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 2},
      "memory": {"size_bytes": 1073741824}
    }
  }'
```

## Security Warning

⚠️ **NEVER use the development authentication in production!** It provides no security and accepts any customer ID without validation.