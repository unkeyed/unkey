package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{5, "5 B"},
		{9, "9 B"},
		{10, "10 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{16000, "16 kB"},
		{1000000, "1.0 MB"},
		{1048576, "1.0 MB"},
		{1500000, "1.5 MB"},
		{1000000000, "1.0 GB"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, humanizeBytes(tc.in), "humanizeBytes(%d)", tc.in)
	}
}

func TestMaskInsufficientPermissionsAsNotFound(t *testing.T) {
	t.Parallel()

	keys := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix)).Keyspace(uid.New(uid.KeySpacePrefix)).Key("*")
	query := rbac.Or(
		rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: uid.New(uid.APIPrefix), Action: rbac.CreateKey}),
		rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.CreateKey}),
		rbac.U(keys, permissions.Write),
	)
	denial := rbac.Check(query, nil)
	notFound := codes.Data.Api.NotFound.URN()
	public := "The specified API was not found."

	t.Run("read access returns 403 naming the missing permission", func(t *testing.T) {
		t.Parallel()

		err := MaskInsufficientPermissionsAsNotFound(denial, []string{keys.String() + "#read"}, query, notFound, public)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		detail := fault.UserFacingMessage(err)
		require.Contains(t, detail, "#write")
		require.Contains(t, detail, "Log in again")
		require.Contains(t, detail, "grant it")
		require.NotContains(t, detail, public)
		require.NotContains(t, detail, "#read")
	})

	t.Run("no access returns not found", func(t *testing.T) {
		t.Parallel()

		err := MaskInsufficientPermissionsAsNotFound(denial, nil, query, notFound, public)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, notFound, code)
		require.Equal(t, public, fault.UserFacingMessage(err))
		require.NotContains(t, fault.UserFacingMessage(err), "permission")
	})

	t.Run("write without read stays not found", func(t *testing.T) {
		t.Parallel()

		err := MaskInsufficientPermissionsAsNotFound(denial, []string{keys.String() + "#write"}, query, notFound, public)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, notFound, code)
		require.Equal(t, public, fault.UserFacingMessage(err))
	})

	t.Run("other errors pass through", func(t *testing.T) {
		t.Parallel()

		original := fault.New("database down", fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
		err := MaskInsufficientPermissionsAsNotFound(original, []string{keys.String() + "#read"}, query, notFound, public)
		require.Same(t, original, err)
	})
}

func TestMaskInsufficientPermissionsWithVisibility(t *testing.T) {
	t.Parallel()

	environment := urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(uid.New(uid.ProjectPrefix)).App(uid.New(uid.AppPrefix)).Environment(uid.New(uid.EnvironmentPrefix))
	required := rbac.U(environment.Deployment("*"), permissions.Write)
	visibility := rbac.Or(
		rbac.U(environment, permissions.Read),
		rbac.U(environment.Deployment("*"), permissions.Read),
	)
	denial := rbac.Check(required, nil)
	notFound := codes.Data.Environment.NotFound.URN()
	public := "The requested project, app, or environment does not exist."

	t.Run("parent read returns 403 naming the required permission", func(t *testing.T) {
		t.Parallel()

		err := MaskInsufficientPermissionsWithVisibility(denial, []string{environment.String() + "#read"}, required, visibility, notFound, public)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		detail := fault.UserFacingMessage(err)
		require.Contains(t, detail, required.Value)
		require.NotContains(t, detail, "#read")
	})

	t.Run("no read returns not found", func(t *testing.T) {
		t.Parallel()

		err := MaskInsufficientPermissionsWithVisibility(denial, nil, required, visibility, notFound, public)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, notFound, code)
		require.Equal(t, public, fault.UserFacingMessage(err))
	})
}
