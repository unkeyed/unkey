// Package certificate implements ACME certificate challenge workflows for SSL/TLS provisioning.
//
// This package handles the complete lifecycle of certificate provisioning using the ACME
// (Automatic Certificate Management Environment) protocol. It coordinates with certificate
// authorities to validate domain ownership and obtain SSL/TLS certificates.
//
// # Built on Restate
//
// All workflows in this package are built on top of Restate (restate.dev) for durable
// execution. This provides critical guarantees:
//
// - Automatic retries on transient failures
// - Exactly-once execution semantics for each workflow step
// - Durable state that survives process crashes and restarts
// - Virtual object concurrency control keyed by domain name
//
// The virtual object model ensures that only one certificate challenge runs per domain
// at any given time, preventing race conditions and duplicate certificate requests that
// could trigger rate limits from certificate authorities.
//
// # Key Types
//
// [Service] is the main entry point that implements the ACME certificate workflow.
// It handles the [Service.ProcessChallenge] method which orchestrates the entire
// certificate issuance process.
//
// # Usage
//
// The service is typically initialized with database connections and a vault service
// for secure storage of private keys:
//
//	svc := certificate.New(certificate.Config{
//	    DB:          mainDB,
//	    PartitionDB: partitionDB,
//	    Vault:       vaultService,
//	    Logger:      logger,
//	})
//
// Certificate challenges are processed through the ProcessChallenge RPC:
//
//	resp, err := svc.ProcessChallenge(ctx, &hydrav1.ProcessChallengeRequest{
//	    WorkspaceId: "ws_123",
//	    Domain:      "api.example.com",
//	})
//	if err != nil {
//	    // Handle error
//	}
//	if resp.Status == "success" {
//	    // Certificate issued successfully
//	}
//
// # ACME Challenge Flow
//
// The certificate challenge process follows these steps:
//
// 1. Domain validation - Verify the domain exists and belongs to the workspace
// 2. Challenge claiming - Acquire exclusive lock on the domain challenge
// 3. ACME client setup - Get or create an ACME account for the workspace
// 4. Certificate obtain/renew - Request certificate from the CA
// 5. Certificate persistence - Store certificate and encrypted private key
// 6. Challenge completion - Mark the challenge as verified with expiry time
//
// Each step is wrapped in a restate.Run call, making it durable and retryable. If the
// workflow crashes at any point, Restate will resume from the last completed step rather
// than restarting from the beginning. This ensures that ACME challenges can complete
// reliably even in the face of system failures, network partitions, or process restarts.
//
// # Security Considerations
//
// Private keys are encrypted before storage using the vault service. Certificates
// are stored in the partition database for fast access by gateways. ACME account
// credentials are workspace-scoped to prevent cross-workspace access.
//
// # Error Handling
//
// The package uses Restate's error handling model. Terminal errors with appropriate
// HTTP status codes are returned for client errors (invalid input, not found, etc.).
// System errors are returned for unexpected failures that may be retried.
package certificate
