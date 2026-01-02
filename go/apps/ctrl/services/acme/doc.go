// Package acme provides ACME certificate challenge management.
//
// This service implements the ProcessChallenge method for handling
// ACME protocol challenges. It coordinates with DNS providers
// and certificate authorities to automatically provision SSL/TLS
// certificates for custom domains.
//
// # Architecture
//
// The service manages the complete ACME challenge workflow:
//  1. Domain validation and ownership verification
//  2. Challenge acquisition and setup
//  3. Certificate issuance from Certificate Authorities
//  4. Certificate persistence and renewal management
//
// It integrates with multiple DNS providers:
//   - HTTP-01 challenges for regular domains via local HTTP service
//   - DNS-01 challenges for wildcard domains via Cloudflare API
//   - DNS-01 challenges for wildcard domains via AWS Route53 API
//
// # Key Components
//
// [Service]: Main ACME challenge processing service
// [Config]: Service configuration with caching layers
//
// # Usage
//
// Creating ACME service:
//
//	svc := acme.New(acme.Config{
//		DB:           database,
//		Logger:        logger,
//		DomainCache:   caches.Domains,
//		ChallengeCache: caches.Challenges,
//	})
//
// Processing challenges:
//
//	resp, err := svc.ProcessChallenge(ctx, &hydrav1.ProcessChallengeRequest{
//		WorkspaceId: "ws_123",
//		Domain:      "api.example.com",
//	})
//	if err != nil {
//		// Handle error
//	}
//
//	if resp.Status == "success" {
//		// Certificate issued successfully
//	}
//
// # Error Handling
//
// The service provides comprehensive error handling:
//   - Domain validation errors for invalid or unauthorized domains
//   - DNS provider errors for API failures or rate limits
//   - Certificate authority errors for issuance problems
//   - System errors for unexpected failures or misconfigurations
//
// # Security Considerations
//
// Private keys are encrypted before storage using the vault service.
// ACME account credentials are workspace-scoped to prevent
// cross-workspace access. Challenge tokens have short
// TTL to prevent replay attacks.
package acme
