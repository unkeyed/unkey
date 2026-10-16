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

	selected := make([]string, len(selections))
	previews := newNewestLookups[FindPrivateNetworkPreviewDeploymentIdParams](findPrivateNetworkPreviewDeploymentId, func(k FindPrivateNetworkPreviewDeploymentIdParams) []any {
		return []any{k.AppID, k.WorkspaceID, k.ProjectID, k.GitBranch, k.ForkRepositoryFullName}
	})
	environments := newNewestLookups[FindPrivateNetworkEnvironmentDeploymentIdParams](findPrivateNetworkEnvironmentDeploymentId, func(k FindPrivateNetworkEnvironmentDeploymentIdParams) []any {
		return []any{k.AppID, k.EnvironmentID, k.WorkspaceID, k.ProjectID}
	})
	for i, s := range selections {
		switch s.SelectionMode {
		case DeploymentConnectionAppTargetsSelectionModeDeployment:
			selected[i] = s.TargetDeploymentID
		case DeploymentConnectionAppTargetsSelectionModeAutomatic:
			if s.CallerEnvironmentKind == mysqltype.EnvironmentKindProduction {
				selected[i] = s.LiveDeploymentID
			} else if s.CallerSource == DeploymentsSourceGit && s.CallerGitBranch != "" {
				previews.add(i, FindPrivateNetworkPreviewDeploymentIdParams{
					AppID: s.AppID, WorkspaceID: s.WorkspaceID, ProjectID: s.ProjectID,
					GitBranch:              sql.NullString{String: s.CallerGitBranch, Valid: true},
					ForkRepositoryFullName: sql.NullString{String: s.CallerForkRepositoryFullName, Valid: true},
				})
			}
		case DeploymentConnectionAppTargetsSelectionModeEnvironment:
			if s.TargetEnvironmentID == "" {
				break
			}
			if s.LiveEnvironmentID == s.TargetEnvironmentID {
				selected[i] = s.LiveDeploymentID
			} else {
				environments.add(i, FindPrivateNetworkEnvironmentDeploymentIdParams{
					AppID: s.AppID, EnvironmentID: s.TargetEnvironmentID, WorkspaceID: s.WorkspaceID, ProjectID: s.ProjectID,
				})
			}
		}
	}
	if err := previews.resolve(ctx, q.db, selected); err != nil {
		return nil, err
	}
	if err := environments.resolve(ctx, q.db, selected); err != nil {
		return nil, err
	}

	ids := slices.Compact(slices.Sorted(slices.Values(selected)))
	if len(ids) > 0 && ids[0] == "" {
		ids = ids[1:]
	}
	targets := make(map[string]ListPrivateNetworkTargetsRow, len(ids))
	if len(ids) > 0 {
		rows, err := q.ListPrivateNetworkTargets(ctx, ListPrivateNetworkTargetsParams{DeploymentIds: ids, Platform: arg.Platform})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			targets[row.ID] = row
		}
	}

	connections := make([]ListPrivateNetworkConnectionsRow, 0, len(selections))
	for i, s := range selections {
		connection := ListPrivateNetworkConnectionsRow{
			WorkspaceID: s.WorkspaceID, ProjectID: s.ProjectID, AppID: s.AppID, AppSlug: s.AppSlug, K8sNamespace: s.K8sNamespace,
			DeploymentID: "", Port: 0, EnvironmentID: "",
			CallerDeploymentID: s.CallerDeploymentID, ConnectionID: s.ConnectionID, ConnectionName: s.ConnectionName,
		}
		if target, ok := targets[selected[i]]; ok && target.AppID == s.AppID && target.WorkspaceID == s.WorkspaceID && target.ProjectID == s.ProjectID {
			connection.DeploymentID, connection.Port, connection.EnvironmentID = target.ID, target.Port, target.EnvironmentID
		}
		connections = append(connections, connection)
	}
	return connections, nil
}

type newestLookups[K comparable] struct {
	lookup     string
	args       func(K) []any
	keys       []K
	slots      map[K]int
	selections map[int]int
}

func newNewestLookups[K comparable](lookup string, args func(K) []any) *newestLookups[K] {
	return &newestLookups[K]{lookup: lookup, args: args, keys: nil, slots: map[K]int{}, selections: map[int]int{}}
}

func (l *newestLookups[K]) add(selection int, key K) {
	slot, ok := l.slots[key]
	if !ok {
		slot = len(l.keys)
		l.slots[key] = slot
		l.keys = append(l.keys, key)
	}
	l.selections[selection] = slot
}

func (l *newestLookups[K]) resolve(ctx context.Context, db DBTX, selected []string) error {
	if len(l.keys) == 0 {
		return nil
	}
	var query strings.Builder
	var params []any
	for slot, key := range l.keys {
		if slot > 0 {
			query.WriteString(" UNION ALL ")
		}
		query.WriteString("SELECT " + strconv.Itoa(slot) + " AS slot, newest.id FROM (" + l.lookup + ") AS newest")
		params = append(params, l.args(key)...)
	}
	rows, err := db.QueryContext(ctx, query.String(), params...)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]string, len(l.keys))
	for rows.Next() {
		var slot int
		var id string
		if err := rows.Scan(&slot, &id); err != nil {
			return err
		}
		ids[slot] = id
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for selection, slot := range l.selections {
		selected[selection] = ids[slot]
	}
	return nil
}
