package portal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// requireMappingNotFound pins the error shape a caller that named an unowned or
// absent resource sees: a plain not-found, so the caller cannot tell the two
// apart.
func requireMappingNotFound(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok, "mapping not-found must carry a code")
	require.Equal(t, codes.Data.Portal.NotFound.URN(), code)
	require.Equal(t, portal.ErrMsgMappingNotFound, fault.UserFacingMessage(err))
}

// A portal URN is project-scoped, so resolving a mapping has to yield the
// project that owns the mapped resource, not only whether the caller owns it.
func TestResolveMappingProject(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "mine",
		Slug:             "mine",
		DeleteProtection: false,
	})

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	app := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Name:             "mine",
		Slug:             "mine",
		DefaultBranch:    "main",
		DeleteProtection: false,
		SourceType:       "",
		ImageReference:   "",
	})

	t.Run("keyspace mapping returns the project of the owning api", func(t *testing.T) {
		got, err := portal.ResolveMappingProject(ctx, h.DB.RO(), workspace.ID, portal.Mapping{
			Type: portal.MappingTypeKeyspace,
			ID:   api.KeyAuthID.String,
		})
		require.NoError(t, err)
		require.Equal(t, project.ID, got)
	})

	t.Run("app mapping returns the app's project", func(t *testing.T) {
		got, err := portal.ResolveMappingProject(ctx, h.DB.RO(), workspace.ID, portal.Mapping{
			Type: portal.MappingTypeApp,
			ID:   app.ID,
		})
		require.NoError(t, err)
		require.Equal(t, project.ID, got)
	})
}

// A resource in another workspace, or none at all, is reported identically: the
// difference would leak that another tenant holds the id.
func TestResolveMappingProjectRejectsForeignAndAbsentResources(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace

	other := h.CreateWorkspace()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      other.ID,
		Name:             "theirs",
		Slug:             "theirs",
		DeleteProtection: false,
	})
	otherApi := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   other.ID,
		ProjectID:     otherProject.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      other.ID,
		ProjectID:        otherProject.ID,
		Name:             "theirs",
		Slug:             "theirs",
		DefaultBranch:    "main",
		DeleteProtection: false,
		SourceType:       "",
		ImageReference:   "",
	})

	testCases := map[string]portal.Mapping{
		"keyspace owned by another workspace": {Type: portal.MappingTypeKeyspace, ID: otherApi.KeyAuthID.String},
		"app owned by another workspace":      {Type: portal.MappingTypeApp, ID: otherApp.ID},
		"keyspace that exists nowhere":        {Type: portal.MappingTypeKeyspace, ID: "ks_doesnotexist"},
		"app that exists nowhere":             {Type: portal.MappingTypeApp, ID: "app_doesnotexist"},
	}

	for name, mapping := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := portal.ResolveMappingProject(ctx, h.DB.RO(), workspace.ID, mapping)
			require.Empty(t, got)
			requireMappingNotFound(t, err)
		})
	}
}

// A soft-deleted keyspace is gone as far as a portal is concerned, so it must
// not resolve to its old project.
func TestResolveMappingProjectRejectsDeletedKeyspace(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     "",
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	_, err := h.DB.RW().ExecContext(ctx,
		"UPDATE key_auth SET deleted_at_m = ? WHERE id = ?",
		time.Now().UnixMilli(), api.KeyAuthID.String,
	)
	require.NoError(t, err)

	got, err := portal.ResolveMappingProject(ctx, h.DB.RO(), workspace.ID, portal.Mapping{
		Type: portal.MappingTypeKeyspace,
		ID:   api.KeyAuthID.String,
	})
	require.Empty(t, got)
	requireMappingNotFound(t, err)
}

// A remap may change which resource a portal serves, but not which project the
// portal belongs to: its URN is project-scoped and stays fixed.
func TestVerifyMappingInProject(t *testing.T) {
	t.Parallel()

	portalProject := uid.New(uid.ProjectPrefix)

	t.Run("accepts a mapping in the portal's project", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, portal.VerifyMappingInProject(portalProject, portalProject))
	})

	t.Run("rejects a mapping in another project", func(t *testing.T) {
		t.Parallel()

		err := portal.VerifyMappingInProject(portalProject, uid.New(uid.ProjectPrefix))
		requireMappingNotFound(t, err)
	})
}
