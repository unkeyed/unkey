// Package github provides utilities for GitHub App authentication and API access.
package github

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Config holds GitHub App configuration.
type Config struct {
	AppID         string
	PrivateKeyPEM string
	WebhookSecret string
}

// Client provides access to GitHub API using App authentication.
type Client struct {
	config     Config
	httpClient *http.Client
	privateKey *rsa.PrivateKey
	logger     logging.Logger
}

// NewClient creates a new GitHub App client.
func NewClient(config Config, logger logging.Logger) (*Client, error) {
	// Handle escaped newlines and surrounding quotes in PEM (common in env vars)
	pemData := strings.ReplaceAll(config.PrivateKeyPEM, "\\n", "\n")
	pemData = strings.Trim(pemData, "\"")
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fault.New("failed to parse PEM block from private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		keyInterface, pkcs8Err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if pkcs8Err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to parse private key"))
		}
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fault.New("private key is not RSA")
		}
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		privateKey: key,
		logger:     logger,
	}, nil
}

// generateJWT creates a JWT for GitHub App authentication.
func (c *Client) generateJWT() (string, error) {
	now := time.Now()

	appIDStr := strings.Trim(c.config.AppID, "\"")
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to parse GitHub App ID as integer"))
	}

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}

	payload := map[string]interface{}{
		"iat": now.Add(-60 * time.Second).Unix(), // 60 seconds in the past for clock drift
		"exp": now.Add(10 * time.Minute).Unix(),  // 10 minutes max
		"iss": appID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to marshal JWT header"))
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to marshal JWT payload"))
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signatureInput := encodedHeader + "." + encodedPayload

	hash := sha256.Sum256([]byte(signatureInput))
	signature, err := rsa.SignPKCS1v15(nil, c.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to sign JWT"))
	}

	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	return signatureInput + "." + encodedSignature, nil
}

// InstallationToken represents a GitHub installation access token.
type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken retrieves an access token for a specific installation.
func (c *Client) GetInstallationToken(installationID int64) (*InstallationToken, error) {

	if err := assert.NotNilAndNotZero(installationID, "installationID must be provided"); err != nil {
		return nil, err
	}

	c.logger.Info("Getting GitHub installation token", "installation_id", installationID)

	jwt, err := c.generateJWT()
	if err != nil {
		return nil, err
	}

	// Log JWT safely - just show first/last few characters for debugging
	jwtPreview := ""
	if len(jwt) > 20 {
		jwtPreview = jwt[:8] + "..." + jwt[len(jwt)-8:]
	} else {
		jwtPreview = "<too_short>"
	}
	c.logger.Info("Generated JWT for GitHub API", "jwt_preview", jwtPreview)

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	c.logger.Info("Calling GitHub API", "url", url)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create request"))
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
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

	var token InstallationToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to decode installation token"))
	}

	return &token, nil
}

// DownloadRepoTarball downloads a repository tarball for a specific ref.
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

// VerifyWebhookSignature verifies a GitHub webhook signature.
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := signature[7:]

	mac := hmacSHA256([]byte(secret), payload)
	actualSig := fmt.Sprintf("%x", mac)

	return hmacEqual([]byte(expectedSig), []byte(actualSig))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hmacEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
