// Package smoke contains external smoke tests for Unkey's public API.
//
// The tests use the public Go SDK to exercise an API deployment from a
// client's perspective. They create temporary APIs, keys, identities,
// permissions, roles, and rate limit overrides, then delete resources that
// support deletion.
//
// Set UNKEY_ROOT_KEY to a root key for the workspace under test. Without this
// variable, every test is skipped. The SDK targets production by default:
//
//	UNKEY_ROOT_KEY=... mise exec -- go test -v ./svc/api/integration/smoke/...
//
// Set UNKEY_API_BASE_URL to target canary or another deployment:
//
//	UNKEY_ROOT_KEY=... UNKEY_API_BASE_URL=https://... mise exec -- go test -v ./svc/api/integration/smoke/...
//
// The root key grants broad access and the tests mutate workspace resources.
// Use a workspace where creating and deleting smoke test resources is safe.
package smoke
