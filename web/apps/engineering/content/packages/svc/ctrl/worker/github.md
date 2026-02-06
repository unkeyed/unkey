---
title: github
description: "provides a GitHub App client for repository access"
---

Package github provides a GitHub App client for repository access.

This package provides authenticated access to GitHub repositories via the GitHub App installation API. It handles JWT generation for App authentication and installation token retrieval for repository-level operations.

### Authentication Flow

GitHub Apps use a two-step authentication process. The client generates a short-lived JWT signed with the App's private key to authenticate as the App, then exchanges it for an installation access token scoped to a specific organization or user account. Installation tokens provide repository access and are valid for one hour.

### Key Types

\[Client] is the main entry point for GitHub API operations. Configure it via \[ClientConfig] and create instances with \[NewClient]. It provides \[Client.GetInstallationToken] for access tokens and \[Client.DownloadRepoTarball] for repository archives. \[InstallationToken] holds the access token and its expiration time.

### Webhook Verification

\[VerifyWebhookSignature] validates incoming webhook payloads using HMAC-SHA256. This ensures webhooks originate from GitHub and have not been tampered with.

### Usage

Create a client and download a repository tarball:

	client, err := github.NewClient(github.ClientConfig{
	    AppID:         12345,
	    PrivateKeyPEM: privateKey,
	}, logger)
	if err != nil {
	    return err
	}

	tarball, err := client.DownloadRepoTarball(installationID, "owner/repo", "main")

Verify a webhook signature:

	if !github.VerifyWebhookSignature(payload, signature, webhookSecret) {
	    return errors.New("invalid webhook signature")
	}

## Functions

### func VerifyWebhookSignature

```go
func VerifyWebhookSignature(payload []byte, signature, secret string) bool
```

VerifyWebhookSignature verifies a GitHub webhook signature using constant-time comparison. The signature should be the value of the X-Hub-Signature-256 header (e.g., "sha256=..."). Returns true only if the signature is valid and matches the expected HMAC-SHA256 of the payload.


## Types

### type Client

```go
type Client struct {
	config     ClientConfig
	httpClient *http.Client
	signer     jwt.Signer[jwt.RegisteredClaims]
	tokenCache cache.Cache[int64, InstallationToken]
}
```

Client provides access to GitHub API using App authentication.

Client handles JWT generation for App-level authentication and installation token retrieval for repository-level operations. It is safe for concurrent use.

#### func NewClient

```go
func NewClient(config ClientConfig) (*Client, error)
```

NewClient creates a \[Client] with the given configuration. Returns an error if the private key cannot be parsed for JWT signing.

#### func (Client) GetInstallationToken

```go
func (c *Client) GetInstallationToken(installationID int64) (InstallationToken, error)
```

GetInstallationToken retrieves an access token for a specific installation. The installation ID is provided by GitHub when the App is installed on an organization or user account. Returns an error if the installation ID is zero or if the GitHub API request fails.

### type ClientConfig

```go
type ClientConfig struct {
	// AppID is the numeric ID assigned to the GitHub App during registration.
	AppID int64

	// PrivateKeyPEM is the RSA private key in PEM format for signing JWTs.
	// Generate this in the GitHub App settings under "Private keys".
	PrivateKeyPEM string

	// WebhookSecret is the shared secret for verifying webhook signatures.
	// Set this in the GitHub App settings under "Webhook secret".
	WebhookSecret string
}
```

ClientConfig holds configuration for creating a \[Client] instance.

### type GitHubClient

```go
type GitHubClient interface {
	// GetInstallationToken retrieves an access token for a specific installation.
	GetInstallationToken(installationID int64) (InstallationToken, error)
}
```

GitHubClient defines the interface for GitHub API operations.

### type InstallationToken

```go
type InstallationToken struct {
	// Token is the installation access token for API requests.
	Token string `json:"token"`

	// ExpiresAt indicates when the token expires, typically 1 hour from issuance.
	ExpiresAt time.Time `json:"expires_at"`
}
```

InstallationToken represents a GitHub installation access token. The token provides repository access for a specific App installation and expires after 1 hour.

### type Noop

```go
type Noop struct{}
```

Noop is a no-op implementation of GitHubClient that returns errors for all operations. Use this when GitHub credentials are not configured.

#### func NewNoop

```go
func NewNoop() *Noop
```

NewNoop creates a new no-op GitHub client.

#### func (Noop) GetInstallationToken

```go
func (n *Noop) GetInstallationToken(installationID int64) (InstallationToken, error)
```

GetInstallationToken returns an error indicating GitHub is not configured.

