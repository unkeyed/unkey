package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// ClientConfig holds configuration for creating a [Client] instance.
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

// Client provides access to GitHub API using App authentication.
//
// Client handles JWT generation for App-level authentication and installation
// token retrieval for repository-level operations. It is safe for concurrent use.
type Client struct {
	config     ClientConfig
	httpClient *http.Client
	signer     jwt.Signer[jwt.RegisteredClaims]
	logger     logging.Logger
}

// NewClient creates a [Client] with the given configuration. Returns an error if
// the private key cannot be parsed for JWT signing.
func NewClient(config ClientConfig, logger logging.Logger) (*Client, error) {
	signer, err := jwt.NewRS256Signer[jwt.RegisteredClaims](config.PrivateKeyPEM)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create JWT signer"))
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		signer:     signer,
		logger:     logger,
	}, nil
}

// generateJWT creates a short-lived JWT for GitHub App authentication.
func (c *Client) generateJWT() (string, error) {
	now := time.Now()
	// nolint:exhaustruct
	claims := jwt.RegisteredClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(10 * time.Minute).Unix(),
		Issuer:    fmt.Sprintf("%d", c.config.AppID),
	}
	return c.signer.Sign(claims)
}

// InstallationToken represents a GitHub installation access token. The token
// provides repository access for a specific App installation and expires after
// 1 hour.
type InstallationToken struct {
	// Token is the installation access token for API requests.
	Token string `json:"token"`

	// ExpiresAt indicates when the token expires, typically 1 hour from issuance.
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken retrieves an access token for a specific installation.
// The installation ID is provided by GitHub when the App is installed on an
// organization or user account. Returns an error if the installation ID is zero
// or if the GitHub API request fails.
func (c *Client) GetInstallationToken(installationID int64) (*InstallationToken, error) {
	if err := assert.NotNilAndNotZero(installationID, "installationID must be provided"); err != nil {
		return nil, err
	}

	c.logger.Info("Getting GitHub installation token", "installation_id", installationID)

	token, err := c.generateJWT()
	if err != nil {
		return nil, err
	}

	tokenPreview := ""
	if len(token) > 20 {
		tokenPreview = token[:8] + "..." + token[len(token)-8:]
	} else {
		tokenPreview = "<too_short>"
	}
	c.logger.Info("Generated JWT for GitHub API", "jwt_preview", tokenPreview)

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	c.logger.Info("Calling GitHub API", "url", url)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create request"))
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to get installation token"))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("GitHub API returned unexpected status",
			"status_code", resp.StatusCode,
			"installation_id", installationID,
			"response_body", string(body),
			"url", url,
		)
		return nil, fault.New("failed to get installation token",
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))),
		)
	}

	var installToken InstallationToken
	if err := json.NewDecoder(resp.Body).Decode(&installToken); err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to decode installation token"))
	}

	return &installToken, nil
}

// DownloadRepoTarball downloads a repository tarball for a specific ref.
// The repoFullName should be in "owner/repo" format. The ref can be a branch
// name, tag, or commit SHA. The entire tarball is loaded into memory, so this
// should only be used for reasonably-sized repositories.
func (c *Client) DownloadRepoTarball(installationID int64, repoFullName, ref string) ([]byte, error) {
	token, err := c.GetInstallationToken(installationID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", repoFullName, ref)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create tarball request"))
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to download tarball"))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fault.New("failed to download tarball",
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))),
		)
	}

	tarball, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to read tarball"))
	}

	return tarball, nil
}

// VerifyWebhookSignature verifies a GitHub webhook signature using constant-time
// comparison. The signature should be the value of the X-Hub-Signature-256 header
// (e.g., "sha256=..."). Returns true only if the signature is valid and matches
// the expected HMAC-SHA256 of the payload.
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := signature[7:]

	mac := hmacSHA256([]byte(secret), payload)
	actualSig := fmt.Sprintf("%x", mac)

	return hmacEqual([]byte(expectedSig), []byte(actualSig))
}

// hmacSHA256 computes the HMAC-SHA256 of data using the provided key.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// hmacEqual compares HMAC digests in constant time to avoid timing attacks.
func hmacEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
