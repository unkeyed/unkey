package authz

import (
	"testing"

	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// TestResources contains common resource IDs that tests might need.
// Tests can use these IDs in their requests and permission checks.
type TestResources struct {
	// WorkspaceID is the primary workspace ID for testing
	WorkspaceID string

	// OtherWorkspaceID is a secondary workspace for cross-workspace testing
	OtherWorkspaceID string

	// ApiID is the primary API ID for testing
	ApiID string

	// OtherApiID is a secondary API for cross-resource testing
	OtherApiID string

	// KeyAuthID is the primary key auth ID
	KeyAuthID string

	// KeyID is the primary key ID for testing
	KeyID string

	// IdentityID is the primary identity ID for testing
	IdentityID string

	// Custom can store any additional resource IDs specific to a test
	Custom map[string]string
}

// PermissionTestConfig configures a 403 authorization test suite.
// TReq is the request type, TRes is the response type.
type PermissionTestConfig[TReq any, TRes any] struct {
	// SetupHandler creates the handler with injected dependencies
	SetupHandler func(*testutil.Harness) zen.Route

	// RequiredPermissions is the list of permissions required to access this endpoint.
	// The test will automatically generate scenarios to test these permissions.
	RequiredPermissions []string

	// CreateRequest creates a valid request using the provided test resources.
	// The resources parameter contains IDs of created test data.
	CreateRequest func(resources TestResources) TReq

	// SetupResources optionally sets up test data before running tests.
	// Returns resource IDs that can be used in CreateRequest and permission checks.
	// If nil, uses default resources from harness.Resources().
	SetupResources func(*testutil.Harness) TestResources

	// AdditionalPermissionTests adds custom permission test cases beyond the standard ones.
	AdditionalPermissionTests []PermissionTestCase[TReq]
}

// PermissionTestCase represents a custom permission test scenario.
type PermissionTestCase[TReq any] struct {
	// Name of the test case
	Name string

	// Permissions to grant to the root key for this test
	Permissions []string

	// ModifyRequest optionally modifies the request for this specific test
	ModifyRequest func(TReq, TestResources) TReq

	// ExpectedStatus is the expected HTTP status code
	ExpectedStatus int

	// ValidateError optionally validates the error response
	ValidateError func(t *testing.T, errorDetail string)
}

// Auth401TestConfig configures a 401 authentication test suite.
type Auth401TestConfig[TReq any, TRes any] struct {
	// SetupHandler creates the handler with injected dependencies
	SetupHandler func(*testutil.Harness) zen.Route

	// CreateRequest creates a valid request for testing
	CreateRequest func() TReq
}
