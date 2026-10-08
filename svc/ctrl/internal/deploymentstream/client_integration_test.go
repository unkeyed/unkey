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
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestWatch_VitessDeletesFilteringAndResume(t *testing.T) {
	for _, test := range []struct {
		name     string
		query    string
		expected []string
	}{
		{name: "soft delete", query: "UPDATE deployment_topology SET desired_status = 'stopped', updated_at = 2 WHERE region_id IN (?, ?)", expected: []string{"included", "historical", "included", "historical"}},
		{name: "hard delete", query: "DELETE FROM deployment_topology WHERE region_id IN (?, ?)", expected: []string{"included", "historical", "included", "historical"}},
		{name: "touch stopped", query: "UPDATE deployment_topology SET updated_at = 2 WHERE desired_status = 'stopped' AND region_id IN (?, ?)", expected: []string{"included", "historical", "historical"}},
	} {
		t.Run(test.name, func(t *testing.T) {
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
		VALUES ('test', 'included', ?, 'running', 1), ('test', 'excluded', ?, 'running', 1),
		('test', 'historical', ?, 'stopped', 1)`, region, region+"_other", region)
			require.NoError(t, err)
			client, err := New(cdc.Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
			require.NoError(t, err)
			var delivered []string
			var token []byte
			updated := false
			err = client.Watch(ctx, region, nil, func(event Event) error {
				if event.DeploymentID != "" {
					require.Empty(t, event.ResumeToken)
					require.Equal(t, updated, event.HasBefore)
					delivered = append(delivered, event.DeploymentID)
					return nil
				}
				require.NotEmpty(t, event.ResumeToken)
				token = event.ResumeToken
				if len(delivered) == 2 && !updated {
					updated = true
					_, err := database.ExecContext(ctx, test.query, region, region+"_other")
					return err
				}
				if len(delivered) >= len(test.expected) {
					cancel()
				}
				return nil
			})
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, test.expected, delivered, "stopped-row updates must be delivered while other regions stay excluded")
			require.NotEmpty(t, token)

			resumeCtx, resumeCancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer resumeCancel()
			err = client.Watch(resumeCtx, region+"_other", token, func(Event) error { return errors.New("unexpected cross-region event") })
			require.ErrorIs(t, err, cdc.ErrInvalidToken)
			_, err = database.ExecContext(resumeCtx, `INSERT INTO deployment_topology
		(workspace_id, deployment_id, region_id, desired_status, created_at)
		VALUES ('test', 'while_offline', ?, 'stopped', 2)`, region)
			require.NoError(t, err)
			_, err = database.ExecContext(resumeCtx, "UPDATE deployment_topology SET updated_at = 3 WHERE region_id = ? AND deployment_id = 'while_offline'", region)
			require.NoError(t, err)
			delivered = nil
			err = client.Watch(resumeCtx, region, token, func(event Event) error {
				if event.DeploymentID != "" {
					require.Equal(t, len(delivered) > 0, event.HasBefore)
					delivered = append(delivered, event.DeploymentID)
					return nil
				}
				if len(delivered) == 2 {
					resumeCancel()
				}
				return nil
			})
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, []string{"while_offline", "while_offline"}, delivered, "resume must deliver stopped-row updates without copying existing history")
		})
	}
}

func TestWatch_VitessRetriesFailedDeliveryAndResumesPartialSnapshot(t *testing.T) {
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
		strings.TrimSuffix(strings.Repeat("('test', ?, ?, 'running', 1),", total), ","), args...)
	require.NoError(t, err)
	client, err := New(cdc.Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
	require.NoError(t, err)
	failure := errors.New("apply failed")
	attempts := 0
	err = client.Watch(ctx, region, nil, func(event Event) error {
		if event.DeploymentID == "" {
			if attempts > 0 {
				return errors.New("checkpoint advanced past failed delivery")
			}
			return nil
		}
		attempts++
		if attempts == 2 {
			return failure
		}
		return nil
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, 2, attempts)
	delivered := make(map[string]int)
	stop := errors.New("disconnect at checkpoint")
	var token []byte
	err = client.Watch(ctx, region, nil, func(event Event) error {
		if event.DeploymentID != "" {
			delivered[event.DeploymentID]++
			return nil
		}
		if len(delivered) == 0 {
			return nil
		}
		token = event.ResumeToken
		return stop
	})
	require.ErrorIs(t, err, stop)
	require.Less(t, len(delivered), total, "disconnect must occur before the snapshot completes")
	err = client.Watch(ctx, region, token, func(event Event) error {
		if event.DeploymentID != "" {
			delivered[event.DeploymentID]++
			return nil
		}
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
