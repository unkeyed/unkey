package db

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApproveCLIDeviceLoginBindsJSONAsText(t *testing.T) {
	// Vitess rejects JSON built from a binary-charset string. With
	// interpolateParams the driver sends []byte as _binary'...' and
	// json.RawMessage as a character string. CAST keeps that string a JSON value.
	require.Contains(t, approveCLIDeviceLogin, "approver_roles = CAST(? AS JSON)")
	require.Contains(t, approveCLIDeviceLogin, "permissions = CAST(? AS JSON)")

	params := reflect.TypeOf(ApproveCLIDeviceLoginParams{})
	roles, ok := params.FieldByName("ApproverRoles")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(json.RawMessage(nil)), roles.Type)
	permissions, ok := params.FieldByName("Permissions")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(json.RawMessage(nil)), permissions.Type)
}
