// Package caches provides shared cache instances for control plane operations.
//
// This package manages in-memory caches for ACME challenge data and
// domain information. These caches improve performance by reducing database
// queries for frequently accessed data during certificate issuance
// and challenge processing.
//
// # Cache Types
//
// [Domains]: Cache for custom domain lookups during ACME challenges.
// Used to validate domain ownership and prevent duplicate registrations.
//
// [Challenges]: Cache for ACME challenge token tracking.
// Stores challenge tokens and authorization data with short TTL due to
// rapid changes during certificate issuance process.
//
// # Configuration
//
// Both caches use different TTL values based on data volatility:
//   - Domain cache: 5 minutes fresh, 10 minutes stale
//   - Challenge cache: 10 seconds fresh, 30 seconds stale
//
// # Key Types
//
// [Caches]: Container holding all cache instances
// [Config]: Configuration for cache initialization
//
// # Usage
//
// Creating caches for control plane:
//
//	caches, err := caches.New(caches.Config{
//		Logger: logger,
//		Clock:  clock.New(),
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Access caches
//	domain, found := caches.Domains.Get("example.com")
//	challenge, found := caches.Challenges.Get("challenge-token")
package caches
