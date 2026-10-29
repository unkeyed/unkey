package github

// tunnel.go implements the `dev github tunnel` subcommand.
//
// Flow:
//  1. Reads UNKEY_GITHUB_APP_ID from dev/.env.github (written by `dev github setup`).
//  2. Reads the app private key from dev/.github-private-key.pem.
//  3. Starts an ngrok process and reads its HTTPS tunnel URL from the child's JSON logs.
//  4. Generates a short-lived GitHub App JWT signed with the private key; the JWT's `iss`
//     claim is the app ID -- GitHub uses this to identify which app is authenticating.
//  5. Calls PATCH /app/hook/config with the JWT; GitHub updates the webhook URL for the
//     app identified by the JWT issuer, so no app ID is needed in the request URL.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/jwt"
)

var tunnelCmd = &cli.Command{
	Name:  "tunnel",
	Usage: "Start an ngrok tunnel to localhost:7091 and update the GitHub App webhook URL automatically",
	Flags: []cli.Flag{
		cli.String("env-file", "Path to .env.github", cli.Default("dev/.env.github")),
		cli.String("pem-file", "Path to .github-private-key.pem", cli.Default("dev/.github-private-key.pem")),
		cli.String("port", "Local port to tunnel", cli.Default("7091")),
	},
	Action: startTunnel,
}

func startTunnel(ctx context.Context, cmd *cli.Command) error {
	envFile := cmd.String("env-file")
	pemFile := cmd.String("pem-file")
	port := cmd.String("port")

	// Fail fast if ngrok is not installed
	if _, err := exec.LookPath("ngrok"); err != nil {
		return fmt.Errorf("ngrok is not installed or not in PATH, install it from https://ngrok.com/download: %w", err)
	}

	appID, err := readAppID(envFile)
	if err != nil {
		return err
	}

	pem, err := os.ReadFile(filepath.Clean(pemFile))
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", pemFile, err)
	}

	// Start ngrok in the background
	fmt.Printf("Starting ngrok tunnel to localhost:%s...\n", port)
	address := "http://localhost:" + port
	ngrok := exec.CommandContext(ctx, "ngrok", "http", address, "--log=stdout", "--log-format=json", "--log-level=info")
	output, err := ngrok.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to read ngrok output: %w", err)
	}
	if err := ngrok.Start(); err != nil {
		return fmt.Errorf("failed to start ngrok: %w", err)
	}
	defer func() {
		_ = ngrok.Process.Kill()
		_ = ngrok.Wait()
	}()

	publicURL, err := waitForNgrokURL(ctx, output, address)
	if err != nil {
		return err
	}

	webhookURL := publicURL + "/webhooks/github"
	fmt.Printf("Tunnel URL: %s\n", publicURL)
	fmt.Printf("Updating GitHub App webhook to %s...\n", webhookURL)

	token, err := generateAppJWT(appID, string(pem))
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}

	if err := patchWebhookURL(token, webhookURL); err != nil {
		return err
	}

	fmt.Printf("✔ Webhook URL updated\n")
	fmt.Printf("✔ Tunnel running (press Ctrl+C to stop)\n\n")
	fmt.Printf("Inspect traffic at: http://localhost:4040\n")

	if _, err := io.Copy(io.Discard, output); err != nil && ctx.Err() == nil {
		return fmt.Errorf("failed to read ngrok output: %w", err)
	}
	return nil
}

// waitForNgrokURL only trusts startup events from the child, not the shared admin API.
func waitForNgrokURL(ctx context.Context, output io.Reader, address string) (string, error) {
	ready := make(chan string, 1)
	failed := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			var event struct {
				Message string `json:"msg"`
				Address string `json:"addr"`
				URL     string `json:"url"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				continue
			}
			if event.Message != "started tunnel" || event.Address != address {
				continue
			}
			publicURL, err := url.Parse(event.URL)
			if err != nil || publicURL.Scheme != "https" || publicURL.Hostname() == "" || publicURL.User != nil ||
				(publicURL.Path != "" && publicURL.Path != "/") || publicURL.RawQuery != "" || publicURL.ForceQuery || publicURL.Fragment != "" {
				continue
			}
			ready <- strings.TrimSuffix(publicURL.String(), "/")
			return
		}
		if err := scanner.Err(); err != nil {
			failed <- fmt.Errorf("failed to read ngrok output: %w", err)
			return
		}
		failed <- fmt.Errorf("ngrok output ended before a valid HTTPS tunnel was ready")
	}()

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case publicURL := <-ready:
		return publicURL, nil
	case err := <-failed:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "", fmt.Errorf("timed out waiting for ngrok tunnel\n\nMake sure ngrok is authenticated: ngrok config add-authtoken <token>")
	}
}

// readAppID parses UNKEY_GITHUB_APP_ID from the env file.
func readAppID(envFile string) (int64, error) {
	val, err := readEnvFileValue(envFile, "UNKEY_GITHUB_APP_ID")
	if err != nil {
		return 0, fmt.Errorf("%w\n\nRun `go run ./build/cli dev github setup` first", err)
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid UNKEY_GITHUB_APP_ID in %s: %w", envFile, err)
	}
	return id, nil
}

// readEnvFileValue parses KEY=VALUE from a simple .env-style file.
func readEnvFileValue(envFile, key string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(envFile))
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", envFile, err)
	}
	prefix := key + "="
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if val, ok := strings.CutPrefix(line, prefix); ok {
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", fmt.Errorf("%s not found in %s", key, envFile)
}

// generateAppJWT creates a short-lived GitHub App JWT for API authentication.
func generateAppJWT(appID int64, privateKeyPEM string) (string, error) {
	signer, err := jwt.NewRS256Signer[jwt.RegisteredClaims](privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	now := time.Now()
	// nolint:exhaustruct
	claims := jwt.RegisteredClaims{
		IssuedAt:  now.Add(-60 * time.Second).Unix(),
		ExpiresAt: now.Add(10 * time.Minute).Unix(),
		Issuer:    strconv.FormatInt(appID, 10),
	}

	return signer.Sign(claims)
}

// patchWebhookURL calls PATCH /app/hook/config to update the app's webhook URL.
func patchWebhookURL(jwtToken, webhookURL string) error {
	body, err := json.Marshal(map[string]string{
		"url":          webhookURL,
		"content_type": "json",
	})
	if err != nil {
		return fmt.Errorf("failed to build request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, "https://api.github.com/app/hook/config", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "unkey-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
