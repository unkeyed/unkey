package db

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"strings"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
)

type ListPrivateNetworkConnectionsParams struct {
	CallerDeploymentIds     []string
	AfterCallerDeploymentID string
	AfterConnectionID       string
	Limit                   int32
	Platform                string
}

// ListPrivateNetworkConnectionsRow is one connection saved by a caller
// deployment. DeploymentID, Port, and EnvironmentID describe the selected
// target and are empty unless it is ready on the platform.
type ListPrivateNetworkConnectionsRow struct {
	WorkspaceID        string
	ProjectID          string
	AppID              string
	AppSlug            string
	K8sNamespace       string
	DeploymentID       string
	Port               int32
	EnvironmentID      string
	CallerDeploymentID string
	ConnectionID       string
	ConnectionName     string
}

// ListPrivateNetworkConnections returns the directed app connections saved by
// the given caller deployments, paged as ListPrivateNetworkConnectionSelections
// pages them, with the target deployment each connection selects now:
//   - a pin selects its deployment;
//   - an automatic connection from a production-kind environment selects the
//     live deployment;
//   - an automatic connection from a git caller elsewhere selects the newest
//     deployment that was ever ready on the caller's branch and fork in a
//     preview environment;
//   - an explicit environment selects the live deployment when it is there,
//     else, for a preview environment, its newest deployment that was ever
//     ready.
//
// The target is empty unless the selected deployment is ready on the platform;
// it never falls back to an older deployment. Each page runs at most four
// queries however many distinct selections it has, so call it in one
// transaction to read one snapshot.
func (q *Queries) ListPrivateNetworkConnections(ctx context.Context, arg ListPrivateNetworkConnectionsParams) ([]ListPrivateNetworkConnectionsRow, error) {
	selections, err := q.ListPrivateNetworkConnectionSelections(ctx, ListPrivateNetworkConnectionSelectionsParams{
		CallerDeploymentIds:     arg.CallerDeploymentIds,
		AfterCallerDeploymentID: arg.AfterCallerDeploymentID,
		AfterConnectionID:       arg.AfterConnectionID,
		Limit:                   arg.Limit,
	})
	if err != nil {
		return nil, err
	}

	deploymentIDs, err := q.selectPrivateNetworkDeployments(ctx, selections)
	if err != nil {
		return nil, err
	}

	ids := slices.Compact(slices.Sorted(slices.Values(deploymentIDs)))
	if len(ids) > 0 && ids[0] == "" {
		ids = ids[1:]
	}
	targets := make(map[string]ListPrivateNetworkTargetsRow, len(ids))
	if len(ids) > 0 {
		rows, err := q.ListPrivateNetworkTargets(ctx, ListPrivateNetworkTargetsParams{
			DeploymentIds: ids,
			Platform:      arg.Platform,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			targets[row.ID] = row
		}
	}

	connections := make([]ListPrivateNetworkConnectionsRow, 0, len(selections))
	for index, selection := range selections {
		connection := ListPrivateNetworkConnectionsRow{
			WorkspaceID:        selection.WorkspaceID,
			ProjectID:          selection.ProjectID,
			AppID:              selection.AppID,
			AppSlug:            selection.AppSlug,
			K8sNamespace:       selection.K8sNamespace,
			DeploymentID:       "",
			Port:               0,
			EnvironmentID:      "",
			CallerDeploymentID: selection.CallerDeploymentID,
			ConnectionID:       selection.ConnectionID,
			ConnectionName:     selection.ConnectionName,
		}
		if target, ok := targets[deploymentIDs[index]]; ok &&
			target.AppID == selection.AppID &&
			target.WorkspaceID == selection.WorkspaceID &&
			target.ProjectID == selection.ProjectID {
			connection.DeploymentID = target.ID
			connection.Port = target.Port
			connection.EnvironmentID = target.EnvironmentID
		}
		connections = append(connections, connection)
	}
	return connections, nil
}

func (q *Queries) selectPrivateNetworkDeployments(ctx context.Context, selections []ListPrivateNetworkConnectionSelectionsRow) ([]string, error) {
	previews := newDeploymentLookupBatch(findPrivateNetworkPreviewDeploymentId, func(params FindPrivateNetworkPreviewDeploymentIdParams) []any {
		return []any{params.AppID, params.GitBranch, params.ForkKey, params.WorkspaceID, params.ProjectID}
	})
	environments := newDeploymentLookupBatch(findPrivateNetworkEnvironmentDeploymentId, func(params FindPrivateNetworkEnvironmentDeploymentIdParams) []any {
		return []any{params.AppID, params.EnvironmentID, params.WorkspaceID, params.ProjectID}
	})

	deploymentIDs := make([]string, len(selections))
	for index, selection := range selections {
		switch selection.SelectionMode {
		case DeploymentConnectionAppTargetsSelectionModeDeployment:
			deploymentIDs[index] = selection.TargetDeploymentID

		case DeploymentConnectionAppTargetsSelectionModeAutomatic:
			if selection.CallerEnvironmentKind == mysqltype.EnvironmentKindProduction {
				deploymentIDs[index] = selection.LiveDeploymentID
				continue
			}
			if selection.CallerSource != DeploymentsSourceGit || selection.CallerGitBranch == "" {
				continue
			}
			previews.add(index, FindPrivateNetworkPreviewDeploymentIdParams{
				AppID:       selection.AppID,
				GitBranch:   sql.NullString{String: selection.CallerGitBranch, Valid: true},
				ForkKey:     sql.NullString{String: selection.CallerForkKey, Valid: true},
				WorkspaceID: selection.WorkspaceID,
				ProjectID:   selection.ProjectID,
			})

		case DeploymentConnectionAppTargetsSelectionModeEnvironment:
			if selection.TargetEnvironmentID == "" {
				continue
			}
			if selection.LiveEnvironmentID == selection.TargetEnvironmentID {
				deploymentIDs[index] = selection.LiveDeploymentID
				continue
			}
			environments.add(index, FindPrivateNetworkEnvironmentDeploymentIdParams{
				AppID:         selection.AppID,
				EnvironmentID: selection.TargetEnvironmentID,
				WorkspaceID:   selection.WorkspaceID,
				ProjectID:     selection.ProjectID,
			})
		}
	}
	if err := previews.resolveInto(ctx, q.db, deploymentIDs); err != nil {
		return nil, err
	}
	if err := environments.resolveInto(ctx, q.db, deploymentIDs); err != nil {
		return nil, err
	}
	return deploymentIDs, nil
}

type deploymentLookupBatch[K comparable] struct {
	query            string
	queryArgs        func(K) []any
	keys             []K
	indexByKey       map[K]int
	indexBySelection map[int]int
}

func newDeploymentLookupBatch[K comparable](query string, queryArgs func(K) []any) *deploymentLookupBatch[K] {
	return &deploymentLookupBatch[K]{
		query:            query,
		queryArgs:        queryArgs,
		keys:             nil,
		indexByKey:       make(map[K]int),
		indexBySelection: make(map[int]int),
	}
}

func (batch *deploymentLookupBatch[K]) add(selectionIndex int, key K) {
	lookupIndex, ok := batch.indexByKey[key]
	if !ok {
		lookupIndex = len(batch.keys)
		batch.indexByKey[key] = lookupIndex
		batch.keys = append(batch.keys, key)
	}
	batch.indexBySelection[selectionIndex] = lookupIndex
}

func (batch *deploymentLookupBatch[K]) resolveInto(ctx context.Context, db DBTX, deploymentIDs []string) error {
	if len(batch.keys) == 0 {
		return nil
	}

	var query strings.Builder
	var params []any
	for lookupIndex, key := range batch.keys {
		if lookupIndex > 0 {
			query.WriteString(" UNION ALL ")
		}
		query.WriteString("SELECT " + strconv.Itoa(lookupIndex) + " AS slot, newest.id FROM (" + batch.query + ") AS newest")
		params = append(params, batch.queryArgs(key)...)
	}

	rows, err := db.QueryContext(ctx, query.String(), params...)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]string, len(batch.keys))
	for rows.Next() {
		var lookupIndex int
		var id string
		if err := rows.Scan(&lookupIndex, &id); err != nil {
			return err
		}
		ids[lookupIndex] = id
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for selectionIndex, lookupIndex := range batch.indexBySelection {
		deploymentIDs[selectionIndex] = ids[lookupIndex]
	}
	return nil
}
