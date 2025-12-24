// Package hash provides deterministic hashing for control plane resources.
//
// This package implements SHA-256 based hashing for sentinel and
// deployment resources. Hashes are used to detect configuration
// changes and ensure deterministic identification for workflow
// processing and caching.
//
// # Hashing Strategy
//
// The package uses SHA-256 for cryptographically secure
// hashing of resource configurations. The hash includes all
// relevant configuration fields to ensure that any change in
// deployment or sentinel parameters results in a different hash.
//
// # Resource Types
//
// Sentinel: Hash includes ID, image, replicas, CPU, and memory
// Deployment: Hash includes ID, replicas, image, region, resources, and desired state
//
// # Usage
//
// Creating resource hashes:
//
//	sentinelHash := hash.Sentinel(sentinelDB)
//	deploymentHash := hash.Deployment(deploymentDB)
//
// These hashes can be used for:
//   - Configuration change detection
//   - Cache key generation
//   - Resource identification in workflows
//   - Deterministic sorting and comparison
//
// The hash output is a hex-encoded SHA-256 digest.
package hash
