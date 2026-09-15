package deploymentstream

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestWatch_VitessSnapshotLiveFilteringAndResume(t *testing.T) {
	vitess := containers.Vitess(t)
	database, err := sql.Open("mysql", vitess.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	region := uid.New(uid.RegionPrefix)
	t.Cleanup(func() {
		_, err := database.ExecContext(context.Background(), "DELETE FROM deployment_topology WHERE region_id IN (?, ?)", region, region+"_other")
		require.NoError(t, err)
	})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	_, err = database.ExecContext(ctx, `INSERT INTO deployment_topology
		(workspace_id, deployment_id, region_id, desired_status, created_at)
		VALUES ('poc', 'included', ?, 'running', 1), ('poc', 'excluded', ?, 'running', 1),
		('poc', 'historical', ?, 'stopped', 1)`, region, region+"_other", region)
	require.NoError(t, err)
	client, err := New(Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	var delivered []string
	var token []byte
	updated := false
	err = client.Watch(ctx, region, nil, func(id string) error {
		delivered = append(delivered, id)
		return nil
	}, func(next []byte) error {
		token = next
		if len(delivered) == 1 && !updated {
			updated = true
			_, err := database.ExecContext(ctx, "UPDATE deployment_topology SET desired_status = 'stopped' WHERE region_id IN (?, ?)", region, region+"_other")
			return err
		}
		if len(delivered) >= 2 {
			cancel()
		}
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"included", "included"}, delivered, "a non-projected status change must still be delivered")
	require.NotEmpty(t, token)

	resumeCtx, resumeCancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer resumeCancel()
	_, err = database.ExecContext(resumeCtx, `INSERT INTO deployment_topology
		(workspace_id, deployment_id, region_id, desired_status, created_at)
		VALUES ('poc', 'while_offline', ?, 'running', 2)`, region)
	require.NoError(t, err)
	delivered = nil
	err = client.Watch(resumeCtx, region, token, func(id string) error {
		delivered = append(delivered, id)
		return nil
	}, func([]byte) error {
		if len(delivered) > 0 {
			resumeCancel()
		}
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"while_offline"}, delivered, "resume must not copy the existing rows again")
}

func TestWatch_VitessResumesPartialSnapshot(t *testing.T) {
	vitess := containers.Vitess(t)
	database, err := sql.Open("mysql", vitess.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	region := uid.New(uid.RegionPrefix)
	t.Cleanup(func() {
		for {
			result, err := database.ExecContext(context.Background(), "DELETE FROM deployment_topology WHERE region_id = ? LIMIT 5000", region)
			require.NoError(t, err)
			deleted, err := result.RowsAffected()
			require.NoError(t, err)
			if deleted == 0 {
				break
			}
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	const total = 20000
	args := make([]any, 0, total*2)
	for i := range total {
		args = append(args, fmt.Sprintf("snapshot_%05d", i), region)
	}
	_, err = database.ExecContext(ctx, `INSERT INTO deployment_topology
		(workspace_id, deployment_id, region_id, desired_status, created_at) VALUES `+
		strings.TrimSuffix(strings.Repeat("('poc', ?, ?, 'running', 1),", total), ","), args...)
	require.NoError(t, err)
	client, err := New(Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	delivered := make(map[string]int)
	change := func(id string) error {
		delivered[id]++
		return nil
	}
	stop := errors.New("disconnect at checkpoint")
	var token []byte
	err = client.Watch(ctx, region, nil, change, func(next []byte) error {
		if len(delivered) == 0 {
			return nil
		}
		token = next
		return stop
	})
	require.ErrorIs(t, err, stop)
	require.Less(t, len(delivered), total, "disconnect must occur before the snapshot completes")
	position, err := client.position(region, token)
	require.NoError(t, err)
	require.NotEmpty(t, position.ShardGtids[0].TablePKs, "partial copy cursor must survive encoding")
	err = client.Watch(ctx, region, token, change, func([]byte) error {
		if len(delivered) == total {
			return stop
		}
		return nil
	})
	require.ErrorIs(t, err, stop)
	for i := range total {
		require.Equal(t, 1, delivered[fmt.Sprintf("snapshot_%05d", i)], "resume must neither skip nor recopy snapshot rows")
	}
}
