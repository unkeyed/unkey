package db

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceAuthorizationAndCLISessionJSONBindsAsText(t *testing.T) {
	// Vitess rejects JSON built from a binary-charset string. With
	// interpolateParams the driver sends []byte as _binary'...' and
	// json.RawMessage as a character string. CAST keeps that string a JSON value.
	require.Contains(t, approveDeviceAuthorization, "approver_roles = CAST(? AS JSON)")
	require.Contains(t, approveDeviceAuthorization, "permissions = CAST(? AS JSON)")
	require.Contains(t, insertCLISession, "CAST(? AS JSON)")
	require.Contains(t, rotateCLISessionRefreshToken, "permissions = CAST(? AS JSON)")

	approve := reflect.TypeOf(ApproveDeviceAuthorizationParams{})
	roles, ok := approve.FieldByName("ApproverRoles")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(json.RawMessage(nil)), roles.Type)
	permissions, ok := approve.FieldByName("Permissions")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(json.RawMessage(nil)), permissions.Type)

	insert := reflect.TypeOf(InsertCLISessionParams{})
	insertPermissions, ok := insert.FieldByName("Permissions")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(json.RawMessage(nil)), insertPermissions.Type)
}
