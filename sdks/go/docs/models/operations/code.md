# Code

If the key is invalid this field will be set to the reason why it is invalid.
Possible values are:
- NOT_FOUND: the key does not exist or has expired
- FORBIDDEN: the key is not allowed to access the api
- USAGE_EXCEEDED: the key has exceeded its request limit
- RATE_LIMITED: the key has been ratelimited,
- INSUFFICIENT_PERMISSIONS: you do not have the required permissions to perform this action



## Values

| Name                          | Value                         |
| ----------------------------- | ----------------------------- |
| `CodeNotFound`                | NOT_FOUND                     |
| `CodeForbidden`               | FORBIDDEN                     |
| `CodeUsageExceeded`           | USAGE_EXCEEDED                |
| `CodeRateLimited`             | RATE_LIMITED                  |
| `CodeUnauthorized`            | UNAUTHORIZED                  |
| `CodeDisabled`                | DISABLED                      |
| `CodeInsufficientPermissions` | INSUFFICIENT_PERMISSIONS      |