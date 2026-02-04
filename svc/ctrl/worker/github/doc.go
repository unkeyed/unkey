// Package github provides a GitHub App client for repository access.
//
// This package provides authenticated access to GitHub repositories via the GitHub
// App installation API. It handles JWT generation for App authentication and
// installation token retrieval for repository-level operations.
//
// # Authentication Flow
//
// GitHub Apps use a two-step authentication process. The client generates a
// short-lived JWT signed with the App's private key to authenticate as the App,
// then exchanges it for an installation access token scoped to a specific
// organization or user account. Installation tokens provide repository access
// and are valid for one hour.
//
// # Key Types
//
// [Client] is the main entry point for GitHub API operations. Configure it via
// [ClientConfig] and create instances with [NewClient]. It provides
// [Client.GetInstallationToken] for access tokens and [Client.DownloadRepoTarball]
// for repository archives. [InstallationToken] holds the access token and its
// expiration time.
//
// # Webhook Verification
//
// [VerifyWebhookSignature] validates incoming webhook payloads using HMAC-SHA256.
// This ensures webhooks originate from GitHub and have not been tampered with.
//
// # Usage
//
// Create a client and download a repository tarball:
//
//	client, err := github.NewClient(github.ClientConfig{
//	    AppID:         12345,
//	    PrivateKeyPEM: privateKey,
//	}, logger)
//	if err != nil {
//	    return err
//	}
//
//	tarball, err := client.DownloadRepoTarball(installationID, "owner/repo", "main")
//
// Verify a webhook signature:
//
//	if !github.VerifyWebhookSignature(payload, signature, webhookSecret) {
//	    return errors.New("invalid webhook signature")
//	}
package github
