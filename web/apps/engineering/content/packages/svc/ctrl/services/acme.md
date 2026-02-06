---
title: acme
description: "provides ACME certificate challenge management"
---

Package acme provides ACME certificate challenge management.

This service implements the ProcessChallenge method for handling ACME protocol challenges. It coordinates with DNS providers and certificate authorities to automatically provision SSL/TLS certificates for custom domains.

### Architecture

The service manages the complete ACME challenge workflow:

 1. Domain validation and ownership verification
 2. Challenge acquisition and setup
 3. Certificate issuance from Certificate Authorities
 4. Certificate persistence and renewal management

It integrates with DNS providers:

  - HTTP-01 challenges for regular domains via local HTTP service
  - DNS-01 challenges for wildcard domains via AWS Route53 API

### Key Components

\[Service]: Main ACME challenge processing service \[Config]: Service configuration with caching layers

### Usage

Creating ACME service:

	svc := acme.New(acme.Config{
		DB:           database,
		DomainCache:   caches.Domains,
		ChallengeCache: caches.Challenges,
	})

Processing challenges:

	resp, err := svc.ProcessChallenge(ctx, &hydrav1.ProcessChallengeRequest{
		WorkspaceId: "ws_123",
		Domain:      "api.example.com",
	})
	if err != nil {
		// Handle error
	}

	if resp.Status == "success" {
		// Certificate issued successfully
	}

### Error Handling

The service provides comprehensive error handling:

  - Domain validation errors for invalid or unauthorized domains
  - DNS provider errors for API failures or rate limits
  - Certificate authority errors for issuance problems
  - System errors for unexpected failures or misconfigurations

### Security Considerations

Private keys are encrypted before storage using the vault service. ACME account credentials are workspace-scoped to prevent cross-workspace access. Challenge tokens have short TTL to prevent replay attacks.

## Functions

### func GetCertificateExpiry

```go
func GetCertificateExpiry(certPEM []byte) (int64, error)
```

GetCertificateExpiry parses a PEM-encoded certificate and returns its expiration time as Unix milliseconds.

### func GetOrCreateUser

```go
func GetOrCreateUser(ctx context.Context, cfg UserConfig) (*lego.Client, error)
```

### func IsCredentialError

```go
func IsCredentialError(err error) bool
```

IsCredentialError returns true if the error is related to bad credentials.

### func IsRateLimited

```go
func IsRateLimited(err error) bool
```

IsRateLimited returns true if the error is a rate limit error.

### func ShouldRetry

```go
func ShouldRetry(err error) bool
```

ShouldRetry returns true if the error is transient and the operation should be retried.


## Types

### type ACMEErrorType

```go
type ACMEErrorType string
```

ACMEErrorType represents the type of ACME error.

```go
const (
	// ACMEErrorRateLimited indicates the request was rate limited by Let's Encrypt.
	ACMEErrorRateLimited ACMEErrorType = "rate_limited"
	// ACMEErrorUnauthorized indicates an authorization/authentication failure.
	ACMEErrorUnauthorized ACMEErrorType = "unauthorized"
	// ACMEErrorBadCredentials indicates AWS/DNS provider credential issues.
	ACMEErrorBadCredentials ACMEErrorType = "bad_credentials"
	// ACMEErrorDNSPropagation indicates DNS propagation timeout.
	ACMEErrorDNSPropagation ACMEErrorType = "dns_propagation"
	// ACMEErrorUnknown indicates an unknown error type.
	ACMEErrorUnknown ACMEErrorType = "unknown"
)
```

### type AcmeUser

```go
type AcmeUser struct {
	WorkspaceID  string
	EmailDomain  string
	Registration *registration.Resource
	key          crypto.PrivateKey
}
```

#### func (AcmeUser) GetEmail

```go
func (u *AcmeUser) GetEmail() string
```

#### func (AcmeUser) GetPrivateKey

```go
func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey
```

#### func (AcmeUser) GetRegistration

```go
func (u AcmeUser) GetRegistration() *registration.Resource
```

### type Config

```go
type Config struct {
	DB             db.Database
	DomainCache    cache.Cache[string, db.CustomDomain]
	ChallengeCache cache.Cache[string, db.AcmeChallenge]
}
```

### type ParsedACMEError

```go
type ParsedACMEError struct {
	// Type is the categorized error type.
	Type ACMEErrorType
	// Message is a human-readable error message.
	Message string
	// RetryAfter is when the request can be retried (for rate limits).
	RetryAfter time.Time
	// IsRetryable indicates if the error is transient and can be retried.
	IsRetryable bool
	// OriginalError is the underlying error.
	OriginalError error
}
```

ParsedACMEError contains parsed information from an ACME error.

#### func ParseACMEError

```go
func ParseACMEError(err error) *ParsedACMEError
```

ParseACMEError analyzes an error from ACME operations and returns structured information.

#### func (ParsedACMEError) Error

```go
func (e *ParsedACMEError) Error() string
```

### type RateLimitError

```go
type RateLimitError struct {
	Message    string
	RetryAfter time.Time
}
```

RateLimitError is a special error type for rate limits that includes retry timing. This is NOT a terminal error - the handler should sleep and retry.

#### func AsRateLimitError

```go
func AsRateLimitError(err error) (*RateLimitError, bool)
```

AsRateLimitError checks if err is a RateLimitError and returns it.

#### func NewRateLimitError

```go
func NewRateLimitError(parsed *ParsedACMEError) *RateLimitError
```

NewRateLimitError creates a RateLimitError from a parsed ACME error.

#### func (RateLimitError) Error

```go
func (e *RateLimitError) Error() string
```

### type Service

```go
type Service struct {
	ctrlv1connect.UnimplementedAcmeServiceHandler
	db             db.Database
	domainCache    cache.Cache[string, db.CustomDomain]
	challengeCache cache.Cache[string, db.AcmeChallenge]
}
```

#### func New

```go
func New(cfg Config) *Service
```

#### func (Service) VerifyCertificate

```go
func (s *Service) VerifyCertificate(
	ctx context.Context,
	req *connect.Request[ctrlv1.VerifyCertificateRequest],
) (*connect.Response[ctrlv1.VerifyCertificateResponse], error)
```

### type UserConfig

```go
type UserConfig struct {
	DB          db.Database
	Vault       vaultv1connect.VaultServiceClient
	WorkspaceID string
	EmailDomain string // Domain for ACME registration emails (e.g., "unkey.com")
}
```

