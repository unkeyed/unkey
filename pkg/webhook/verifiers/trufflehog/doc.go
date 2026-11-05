// Package trufflehog verifies requests from TruffleHog's Partner Revocation
// Program, which reports leaked credentials it finds in public sources.
//
// Each request carries an ECDSA P-256 signature over the raw body, encoded as
// base64 ASN.1 DER in Truffle-Public-Key-Signature, and the hex SHA-256 of the
// signing key's DER SubjectPublicKeyInfo in Truffle-Public-Key-Identifier.
// TruffleHog publishes a single static key and has no key endpoint, so the
// verifier holds the key in memory and never fetches.
//
// # Errors
//
// Missing signing headers wrap [webhook.ErrBadRequest]. An unknown key
// identifier, malformed signature, or signature mismatch returns a plain error,
// which the receiver reports as unauthorized.
//
// # Usage
//
//	verifier, err := trufflehog.New(trufflehog.PublicKeyPEM)
//	if err != nil {
//		return err
//	}
//	receiver := webhook.New("trufflehog_secret_scanning", verifier).
//		On([]string{trufflehog.EventType}, handle)
//
// See [github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets] for the
// GitHub equivalent, which fetches rotating keys.
package trufflehog
