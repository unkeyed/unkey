package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordingExecDB captures the statements the generated instance queries
// execute, so a test can tell a targeted UPDATE apart from an upsert without a
// database.
type recordingExecDB struct {
	statements []string
}

func (r *recordingExecDB) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	r.statements = append(r.statements, query)
	return driver.RowsAffected(0), nil
}

func (r *recordingExecDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected PrepareContext")
}

func (r *recordingExecDB) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("unexpected QueryContext")
}

func (r *recordingExecDB) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

// TestUpdateInstanceIsTargetedUpdate pins the write for an instance that is
// already known: an UPDATE by id locks only that row. The INSERT ... ON
// DUPLICATE KEY UPDATE it replaces probes the id and (k8s_name, region_id)
// unique indexes and allocates an auto-increment value on every report, and
// those gap locks deadlock under concurrent reports (errno 1213).
func TestUpdateInstanceIsTargetedUpdate(t *testing.T) {
	rec := &recordingExecDB{}
	err := NewQueries(rec).UpdateInstance(context.Background(), UpdateInstanceParams{
		ID:            "inst_4f8a",
		Address:       "10.0.0.7",
		CpuMillicores: 250,
		MemoryMib:     256,
		Status:        InstancesStatusRunning,
	})
	require.NoError(t, err)
	require.Len(t, rec.statements, 1)
	require.Contains(t, rec.statements[0], "UPDATE instances")
	require.Contains(t, rec.statements[0], "WHERE id = ?")
	require.NotContains(t, rec.statements[0], "ON DUPLICATE KEY UPDATE", "known instances must update in place")
}

// TestUpsertInstanceKeepsDuplicateKeyHandling pins the new-row path: two
// reports can race to insert the same k8s_name in a region, so the insert must
// still absorb a duplicate key.
func TestUpsertInstanceKeepsDuplicateKeyHandling(t *testing.T) {
	rec := &recordingExecDB{}
	err := NewQueries(rec).UpsertInstance(context.Background(), UpsertInstanceParams{
		ID:           "inst_2c19",
		DeploymentID: "d_1",
		WorkspaceID:  "ws_1",
		ProjectID:    "p_1",
		AppID:        "a_1",
		RegionID:     "us-east-1",
		K8sName:      "k8s-1",
		Address:      "10.0.0.8",
		Status:       InstancesStatusRunning,
	})
	require.NoError(t, err)
	require.Len(t, rec.statements, 1)
	require.Contains(t, rec.statements[0], "ON DUPLICATE KEY UPDATE")
}
