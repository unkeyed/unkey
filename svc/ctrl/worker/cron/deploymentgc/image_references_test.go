package deploymentgc

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type imageReferenceDB struct {
	db.Database
	owner  bool
	reused bool
	err    error
}

func (d *imageReferenceDB) DeploymentExistsIncludingDeleted(context.Context, string) (bool, error) {
	return d.owner, d.err
}

func (d *imageReferenceDB) DeploymentImageExistsIncludingDeleted(context.Context, sql.NullString) (bool, error) {
	return d.reused, d.err
}

type imageDeleteDepot struct {
	Depot
	calls int
	err   error
}

func (d *imageDeleteDepot) DeleteImage(context.Context, string, string) error {
	d.calls++
	return d.err
}

func TestImageDeletionRechecksOwnerAndReuseOnEveryAttempt(t *testing.T) {
	database := &imageReferenceDB{}
	depot := &imageDeleteDepot{}
	h := &Handler{db: database, depot: depot, registryRepository: "registry.depot.dev/test", registryProjectID: "test"}
	const tag = "proj_project-d_owner"
	for _, tc := range []struct {
		name   string
		owner  bool
		reused bool
	}{
		{name: "owner still building", owner: true},
		{name: "owner in recovery", owner: true},
		{name: "owner purged but image reused", reused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database.owner, database.reused = tc.owner, tc.reused
			removed, err := h.deleteUnreferencedImage(t.Context(), tag)
			require.NoError(t, err)
			require.False(t, removed)
			require.Zero(t, depot.calls)
		})
	}
	database.owner, database.reused = false, false
	depot.err = errors.New("temporary Depot failure")
	removed, err := h.deleteUnreferencedImage(t.Context(), tag)
	require.Error(t, err)
	require.False(t, removed)
	require.Equal(t, 1, depot.calls)

	database.reused = true
	depot.err = nil
	removed, err = h.deleteUnreferencedImage(t.Context(), tag)
	require.NoError(t, err)
	require.False(t, removed)
	require.Equal(t, 1, depot.calls, "a retry must not reuse a journaled absence check")

	database.reused = false
	database.err = errors.New("database unavailable")
	removed, err = h.deleteUnreferencedImage(t.Context(), tag)
	require.Error(t, err)
	require.False(t, removed)
	require.Equal(t, 1, depot.calls)

	database.err = nil
	removed, err = h.deleteUnreferencedImage(t.Context(), tag)
	require.NoError(t, err)
	require.True(t, removed)
	require.Equal(t, 2, depot.calls)
}
