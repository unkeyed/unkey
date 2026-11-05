package privatenetwork

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestConnectionLoad seeds private network connection data, grows one
// relationship at a time, and times every connection query on each dataset.
// It writes a loadReport to the path in UNKEY_CONNECTION_LOAD.
//
//	UNKEY_CONNECTION_LOAD=/tmp/before.json \
//	UNKEY_CONNECTION_LOAD_SCALES=1,10,100,1000 \
//	UNKEY_CONNECTION_LOAD_SAMPLES=40 \
//	mise exec -- go test -run TestConnectionLoad -timeout 3h ./svc/ctrl/internal/privatenetwork/
func TestConnectionLoad(t *testing.T) {
	out := os.Getenv("UNKEY_CONNECTION_LOAD")
	if out == "" {
		t.Skip("set UNKEY_CONNECTION_LOAD to a report path to run the connection load test")
	}
	scales := loadEnvInts(t, "UNKEY_CONNECTION_LOAD_SCALES", []int{1, 10, 100, 1000})
	samples := loadEnvInts(t, "UNKEY_CONNECTION_LOAD_SAMPLES", []int{40})[0]

	report := loadReport{Label: os.Getenv("UNKEY_CONNECTION_LOAD_LABEL"), Samples: samples, Results: nil}
	for _, dataset := range loadDatasets(scales) {
		t.Run(fmt.Sprintf("%s-x%d", dataset.axis, dataset.scale), func(t *testing.T) {
			database, err := sql.Open("mysql", containers.MySQLLogicalIsolated(t).DSN)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })

			started := time.Now()
			probe := seedLoad(t, database, dataset.shape)
			t.Logf("seeded in %s", time.Since(started).Round(time.Millisecond))

			conn, err := database.Conn(t.Context())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })
			for _, result := range measureLoad(t, conn, loadOps(t, conn, probe), samples) {
				result.Axis, result.Scale = dataset.axis, dataset.scale
				report.Results = append(report.Results, result)
				t.Logf("%-38s p50 %9s  rows read %8d  rows %5d %s", result.Op,
					time.Duration(loadMedian(result.SamplesNs)).Round(time.Microsecond), result.RowsRead, result.Rows, result.Error)
			}
		})
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(out, encoded, 0o644))
}

type loadDataset struct {
	axis  string
	scale int
	shape loadShape
}

func loadDatasets(scales []int) []loadDataset {
	datasets := []loadDataset{{axis: "base", scale: 1, shape: loadShapeBase}}
	for _, axis := range loadAxes {
		for _, scale := range scales {
			if scale > 1 {
				datasets = append(datasets, loadDataset{axis: axis.name, scale: scale, shape: axis.scale(loadShapeBase, scale)})
			}
		}
	}
	return datasets
}

type loadReport struct {
	Label   string       `json:"label"`
	Samples int          `json:"samples"`
	Results []loadResult `json:"results"`
}

type loadResult struct {
	Axis      string  `json:"axis"`
	Scale     int     `json:"scale"`
	Op        string  `json:"op"`
	SamplesNs []int64 `json:"samplesNs"`
	RowsRead  int64   `json:"rowsRead"`
	Rows      int     `json:"rows"`
	Error     string  `json:"error,omitempty"`
}

type loadOp struct {
	name string
	run  func(ctx context.Context, conn *sql.Conn) (int, error)
}

func loadOps(t *testing.T, conn *sql.Conn, probe loadProbe) []loadOp {
	t.Helper()
	queries := db.NewQueries(conn)
	selections, err := queries.ListPrivateNetworkConnectionSelections(t.Context(), db.ListPrivateNetworkConnectionSelectionsParams{
		CallerDeploymentIds: probe.callerIDs, AfterCallerDeploymentID: "", AfterConnectionID: "", Limit: connectionPageSize,
	})
	require.NoError(t, err)
	selected, err := selectDeployments(t.Context(), queries, selections)
	require.NoError(t, err)
	targetIDs := slices.DeleteFunc(slices.Compact(slices.Sorted(slices.Values(selected))), func(id string) bool { return id == "" })
	regionIDs := slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(probe.callerIDs), targetIDs...))))

	var previewApps, previewCallers, environmentIDs []string
	for _, s := range selections {
		switch {
		case s.SelectionMode == db.DeploymentConnectionAppTargetsSelectionModeAutomatic &&
			s.CallerEnvironmentKind != mysqltype.EnvironmentKindProduction && s.CallerGitBranch != "":
			previewApps, previewCallers = append(previewApps, s.AppID), append(previewCallers, s.CallerDeploymentID)
		case s.SelectionMode == db.DeploymentConnectionAppTargetsSelectionModeEnvironment && s.TargetEnvironmentID != s.LiveEnvironmentID:
			environmentIDs = append(environmentIDs, s.TargetEnvironmentID)
		}
	}
	distinctIDs := func(ids []string) []string { return slices.Compact(slices.Sorted(slices.Values(ids))) }
	pinned := sql.NullString{String: probe.pinnedID, Valid: true}

	return []loadOp{
		{"ctrl.ListPrivateNetworkReplicas", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkReplicas(ctx, db.ListPrivateNetworkReplicasParams{DeploymentIds: probe.callerIDs, Platform: probe.platform})
			return len(rows), err
		}},
		{"ctrl.ListPrivateNetworkConnectionSelections", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkConnectionSelections(ctx, db.ListPrivateNetworkConnectionSelectionsParams{
				CallerDeploymentIds: probe.callerIDs, AfterCallerDeploymentID: "", AfterConnectionID: "", Limit: connectionPageSize,
			})
			return len(rows), err
		}},
		{"ctrl.ListPrivateNetworkPreviewDeploymentIds", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkPreviewDeploymentIds(ctx, db.ListPrivateNetworkPreviewDeploymentIdsParams{
				CallerDeploymentIds: distinctIDs(previewCallers), AppIds: distinctIDs(previewApps),
			})
			return len(rows), err
		}},
		{"ctrl.ListPrivateNetworkEnvironmentDeploymentIds", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkEnvironmentDeploymentIds(ctx, distinctIDs(environmentIDs))
			return len(rows), err
		}},
		{"ctrl.ListPrivateNetworkTargets", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkTargets(ctx, db.ListPrivateNetworkTargetsParams{DeploymentIds: targetIDs, Platform: probe.platform})
			return len(rows), err
		}},
		{"ctrl.ListPrivateNetworkDeploymentRegions", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := db.NewQueries(conn).ListPrivateNetworkDeploymentRegions(ctx, db.ListPrivateNetworkDeploymentRegionsParams{DeploymentIds: regionIDs, Platform: probe.platform})
			return len(rows), err
		}},
		{"ctrl.listConnections (page)", func(ctx context.Context, conn *sql.Conn) (int, error) {
			page, err := listConnections(ctx, db.NewQueries(conn), db.ListPrivateNetworkConnectionSelectionsParams{
				CallerDeploymentIds: probe.callerIDs, AfterCallerDeploymentID: "", AfterConnectionID: "", Limit: connectionPageSize,
			}, probe.platform)
			return len(page), err
		}},
		{"ctrl.ExistsAppConnectionPinningDeployment", func(ctx context.Context, conn *sql.Conn) (int, error) {
			_, err := db.NewQueries(conn).ExistsAppConnectionPinningDeployment(ctx, db.ExistsAppConnectionPinningDeploymentParams{DeploymentID: pinned})
			return 1, err
		}},
		{"api.ListAppConnections", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListAppConnections(ctx, conn, apidb.ListAppConnectionsParams{
				WorkspaceID: probe.workspaceID, ProjectID: probe.projectID, AppID: probe.callerID, EnvironmentID: probe.callerProductionID, IDCursor: "", Limit: 101,
			})
			return len(rows), err
		}},
		{"api.ListRetainedAppConnections", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListRetainedAppConnections(ctx, conn, apidb.ListRetainedAppConnectionsParams{
				WorkspaceID: probe.workspaceID, ProjectID: probe.projectID, AppID: probe.callerID, EnvironmentID: probe.callerProductionID, IDCursor: "", Limit: 101,
			})
			return len(rows), err
		}},
		{"api.ListAppConnectionTargetDeployments", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListAppConnectionTargetDeployments(ctx, conn, apidb.ListAppConnectionTargetDeploymentsParams{
				WorkspaceID: probe.workspaceID, AppID: probe.hubID, CursorID: "", Limit: 101,
			})
			return len(rows), err
		}},
		{"api.FindAppConnectionPinnedDeployment", func(ctx context.Context, conn *sql.Conn) (int, error) {
			_, err := apidb.Query.FindAppConnectionPinnedDeployment(ctx, conn, apidb.FindAppConnectionPinnedDeploymentParams{
				WorkspaceID: probe.workspaceID, AppID: probe.callerID, EnvironmentID: probe.callerProductionID, TargetAppID: probe.nextID,
			})
			return 1, err
		}},
		{"api.ListDeploymentConnectionSummaries", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListDeploymentConnectionSummaries(ctx, conn, apidb.ListDeploymentConnectionSummariesParams{
				WorkspaceID: probe.workspaceID, ProjectID: probe.projectID, DeploymentID: probe.callerLiveID, IDCursor: "", Limit: 101,
			})
			return len(rows), err
		}},
		{"api.ListSnapshotDeploymentConnectionPins", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListSnapshotDeploymentConnectionPins(ctx, conn, apidb.ListSnapshotDeploymentConnectionPinsParams{
				WorkspaceID: probe.workspaceID, ProjectID: probe.projectID, TargetAppID: probe.nextID, DeploymentID: pinned, PinCursor: "", Limit: 101,
			})
			return len(rows), err
		}},
		{"api.ListCurrentDeploymentConnectionPins", func(ctx context.Context, conn *sql.Conn) (int, error) {
			rows, err := apidb.Query.ListCurrentDeploymentConnectionPins(ctx, conn, apidb.ListCurrentDeploymentConnectionPinsParams{
				WorkspaceID: probe.workspaceID, ProjectID: probe.projectID, TargetAppID: probe.nextID, DeploymentID: pinned, PinCursor: "", Limit: 101,
			})
			return len(rows), err
		}},
	}
}

func measureLoad(t *testing.T, conn *sql.Conn, ops []loadOp, samples int) []loadResult {
	t.Helper()
	const warmup = 3
	results := make([]loadResult, len(ops))
	reads := make([][]int64, len(ops))
	for i, op := range ops {
		results[i].Op = op.name
	}
	overhead := handlerReadsOverhead(t, conn)
	order := make([]int, len(ops))
	for i := range order {
		order[i] = i
	}
	random := rand.New(rand.NewPCG(3, 4))
	for round := range warmup + samples {
		random.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		for _, i := range order {
			if results[i].Error != "" {
				continue
			}
			before := handlerReads(t, conn)
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			started := time.Now()
			rows, err := ops[i].run(ctx, conn)
			elapsed := time.Since(started)
			cancel()
			read := handlerReads(t, conn) - before - overhead
			if err != nil {
				results[i].Error = err.Error()
				continue
			}
			if round >= warmup {
				results[i].SamplesNs = append(results[i].SamplesNs, elapsed.Nanoseconds())
				reads[i] = append(reads[i], read)
				results[i].Rows = rows
			}
		}
	}
	for i := range results {
		results[i].RowsRead = loadMedian(reads[i])
	}
	return results
}

// handlerReads returns the rows the session read from storage engines so far.
// MySQL reports rows examined only through session status variables, which no
// generated query reads.
func handlerReads(t *testing.T, conn *sql.Conn) int64 {
	t.Helper()
	rows, err := conn.QueryContext(t.Context(), "SHOW SESSION STATUS LIKE 'Handler_read%'")
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var total int64
	for rows.Next() {
		var name, value string
		require.NoError(t, rows.Scan(&name, &value))
		n, err := strconv.ParseInt(value, 10, 64)
		require.NoError(t, err)
		total += n
	}
	require.NoError(t, rows.Err())
	return total
}

func handlerReadsOverhead(t *testing.T, conn *sql.Conn) int64 {
	t.Helper()
	before := handlerReads(t, conn)
	return handlerReads(t, conn) - before
}

func loadEnvInts(t *testing.T, name string, fallback []int) []int {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	var values []int
	for field := range strings.SplitSeq(value, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(field))
		require.NoError(t, err, "parse %s", name)
		values = append(values, n)
	}
	return values
}
