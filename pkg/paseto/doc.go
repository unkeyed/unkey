// Package paseto implements PASETO version 4 local and public tokens.
//
// Payload types must embed [Claims]. Local tokens use XChaCha20 encryption and
// keyed BLAKE2b authentication. Public tokens use Ed25519 signatures. Every
// token operation uses an empty implicit assertion.
//
// Decrypt and Verify authenticate claims but do not apply application policy.
// Callers must check expiration, not-before, issuer, audience, and replay rules.
package paseto
