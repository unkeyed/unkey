package clidevice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

type fakeDevices struct {
	code          workos.DeviceCode
	authorization workos.DeviceAuthorization
}

func (f *fakeDevices) CreateDevice(context.Context, string) (workos.DeviceCode, workos.DeviceAuthorization, error) {
	return f.code, f.authorization, nil
}

func TestDeviceLoginIssuesOnlySelectedPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	devices := &fakeDevices{
		code: "device-secret",
		authorization: workos.DeviceAuthorization{
			UserCode:                "zhvg-sxqm",
			VerificationURIComplete: "https://auth.example/device?user_code=ZHVG-SXQM",
			ExpiresIn:               300,
			Interval:                5,
		},
	}
	svc := &Service{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
		Devices: devices, ClientID: "client_test", DashboardBaseURL: "http://localhost:3000",
	}

	started, err := svc.Start(t.Context(), "Florians-MacBook-Pro.local")
	require.NoError(t, err)
	require.Equal(t, "ZHVG-SXQM", started.UserCode)
	require.NotContains(t, started.VerificationURIComplete, "device-secret")
	require.Contains(t, started.VerificationURIComplete, "user_code=ZHVG-SXQM")
	require.NotEmpty(t, started.LoginID)

	pending, err := svc.Poll(t.Context(), started.LoginID, "127.0.0.1", "unkey-cli")
	require.NoError(t, err)
	require.Equal(t, PollPermissionsRequired, pending.Status)
	require.Nil(t, pending.Key)

	view, err := svc.Get(t.Context(), "zhvg-sxqm")
	require.NoError(t, err)
	require.Equal(t, "Florians-MacBook-Pro.local", view.DeviceName)
	require.Equal(t, "https://auth.example/device?user_code=ZHVG-SXQM", view.WorkOSVerificationURI)

	read := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	verify := "unkey:v1:" + workspace.ID + ":projects/*/keyspaces/*/keys/*#verify"
	admin := adminPrincipal(workspace, "user_admin")
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal:   admin,
		UserCode:    started.UserCode,
		Name:        "CLI on Florians-MacBook-Pro.local",
		Permissions: []string{verify, read, read},
	}))

	developer := adminPrincipal(workspace, "user_admin")
	developer.Source = principal.JWTSource{Roles: []string{"developer"}}
	developer.Permissions = workos.PermissionsForRoles(workspace.ID, []string{"developer"})
	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Name:        "too much",
		Permissions: []string{read},
	})
	require.Error(t, err)

	done, err := svc.Poll(t.Context(), started.LoginID, "127.0.0.1", "unkey-cli")
	require.NoError(t, err)
	require.Equal(t, PollComplete, done.Status)
	require.NotNil(t, done.Key)
	require.NotEmpty(t, *done.Key)
	require.NotContains(t, *done.Key, "*")

	stored, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   workspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   done.KeyID,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{read, verify}, stored)

	_, err = svc.Poll(t.Context(), started.LoginID, "127.0.0.1", "unkey-cli")
	require.Error(t, err)
}

func TestApproveRejectsADifferentAccount(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	devices := &fakeDevices{
		code: "device-secret",
		authorization: workos.DeviceAuthorization{
			UserCode:                "ABCD-EFGH",
			VerificationURIComplete: "https://auth.example/device", ExpiresIn: 300, Interval: 5,
		},
	}
	svc := &Service{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
		Devices: devices, ClientID: "client_test", DashboardBaseURL: "http://localhost:3000",
	}
	started, err := svc.Start(t.Context(), "laptop")
	require.NoError(t, err)

	perm := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	require.NoError(t, svc.Approve(t.Context(), ApproveRequest{
		Principal: adminPrincipal(workspace, "user_admin"), UserCode: started.UserCode, Name: "CLI", Permissions: []string{perm},
	}))
	err = svc.Approve(t.Context(), ApproveRequest{
		Principal: adminPrincipal(workspace, "user_other"), UserCode: started.UserCode, Name: "CLI", Permissions: []string{perm},
	})
	require.Error(t, err)

	done, err := svc.Poll(t.Context(), started.LoginID, "", "")
	require.NoError(t, err)
	require.Equal(t, PollComplete, done.Status)
	require.NotNil(t, done.Key)
}

func TestApproveRejectsPermissionsOutsideTheCaller(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	devices := &fakeDevices{
		code: "device-secret",
		authorization: workos.DeviceAuthorization{
			UserCode: "QRST-UVWX", VerificationURIComplete: "https://auth.example/device",
			ExpiresIn: 300, Interval: 5,
		},
	}
	svc := &Service{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
		Devices: devices, ClientID: "client_test", DashboardBaseURL: "http://localhost:3000",
	}
	started, err := svc.Start(t.Context(), "laptop")
	require.NoError(t, err)

	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   adminPrincipal(workspace, "user_admin"),
		UserCode:    started.UserCode,
		Name:        "CLI",
		Permissions: []string{"unkey:v1:ws_other:rootKeys/*#read"},
	})
	require.Error(t, err)

	developer := adminPrincipal(workspace, "user_dev")
	developer.Source = principal.JWTSource{Roles: []string{"developer"}}
	developer.Permissions = workos.PermissionsForRoles(workspace.ID, []string{"developer"})
	err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developer,
		UserCode:    started.UserCode,
		Name:        "CLI",
		Permissions: []string{"unkey:v1:" + workspace.ID + ":projects/*/keyspaces/*/keys/*#verify"},
	})
	require.Error(t, err)
}

func TestEnabledRejectsANilDeviceClient(t *testing.T) {
	var client *workos.DeviceClient
	svc := &Service{
		Devices:          client,
		ClientID:         "client_test",
		DashboardBaseURL: "http://localhost:3000",
	}
	require.False(t, svc.Enabled())
	_, err := svc.Start(context.Background(), "laptop")
	require.Error(t, err)
	require.Error(t, svc.Approve(context.Background(), ApproveRequest{}))
	require.Error(t, svc.Deny(context.Background(), nil, "ABCD-EFGH"))

	blankID := &Service{Devices: &fakeDevices{}, ClientID: " ", DashboardBaseURL: "http://localhost:3000"}
	require.False(t, blankID.Enabled())

	ready := &Service{Devices: &fakeDevices{}, ClientID: "client_test", DashboardBaseURL: "http://localhost:3000"}
	require.True(t, ready.Enabled())
}

func adminPrincipal(workspace db.Workspace, userID string) *principal.Principal {
	return &principal.Principal{
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: userID, Name: "Admin", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
}
