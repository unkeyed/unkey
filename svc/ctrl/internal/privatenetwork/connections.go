package privatenetwork

import (
	"context"
	"database/sql"
	"slices"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type connection struct {
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

// listConnections returns one page of the directed app connections saved by
// the given caller deployments, paged as
// [db.Queries.ListPrivateNetworkConnectionSelections] pages them, with the
// target deployment each connection selects now:
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
// it never falls back to an older deployment. Read every page in the
// transaction that read [db.Queries.ListPrivateNetworkReplicas], so all pages
// see one snapshot.
func listConnections(ctx context.Context, queries *db.Queries, params db.ListPrivateNetworkConnectionSelectionsParams, platform string) ([]connection, error) {
	selections, err := queries.ListPrivateNetworkConnectionSelections(ctx, params)
	if err != nil {
		return nil, err
	}
	deploymentIDs, err := selectDeployments(ctx, queries, selections)
	if err != nil {
		return nil, err
	}
	targets, err := readyTargets(ctx, queries, deploymentIDs, platform)
	if err != nil {
		return nil, err
	}

	connections := make([]connection, 0, len(selections))
	for i, selection := range selections {
		c := connection{
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
		if target, ok := targets[deploymentIDs[i]]; ok &&
			target.AppID == selection.AppID &&
			target.WorkspaceID == selection.WorkspaceID &&
			target.ProjectID == selection.ProjectID {
			c.DeploymentID = target.ID
			c.Port = target.Port
			c.EnvironmentID = target.EnvironmentID
		}
		connections = append(connections, c)
	}
	return connections, nil
}

func selectDeployments(ctx context.Context, queries *db.Queries, selections []db.ListPrivateNetworkConnectionSelectionsRow) ([]string, error) {
	deploymentIDs := make([]string, len(selections))
	var previews lookups[db.FindPrivateNetworkPreviewDeploymentIdParams]
	var environments lookups[db.FindPrivateNetworkEnvironmentDeploymentIdParams]
	for i, selection := range selections {
		switch selection.SelectionMode {
		case db.DeploymentConnectionAppTargetsSelectionModeDeployment:
			deploymentIDs[i] = selection.TargetDeploymentID
		case db.DeploymentConnectionAppTargetsSelectionModeAutomatic:
			switch {
			case selection.CallerEnvironmentKind == mysqltype.EnvironmentKindProduction:
				deploymentIDs[i] = selection.LiveDeploymentID
			case selection.CallerSource == db.DeploymentsSourceGit && selection.CallerGitBranch != "":
				previews.add(i, db.FindPrivateNetworkPreviewDeploymentIdParams{
					AppID:       selection.AppID,
					GitBranch:   sql.NullString{String: selection.CallerGitBranch, Valid: true},
					ForkKey:     sql.NullString{String: selection.CallerForkKey, Valid: true},
					WorkspaceID: selection.WorkspaceID,
					ProjectID:   selection.ProjectID,
				})
			}
		case db.DeploymentConnectionAppTargetsSelectionModeEnvironment:
			switch {
			case selection.TargetEnvironmentID == "":
			case selection.TargetEnvironmentID == selection.LiveEnvironmentID:
				deploymentIDs[i] = selection.LiveDeploymentID
			default:
				environments.add(i, db.FindPrivateNetworkEnvironmentDeploymentIdParams{
					AppID:         selection.AppID,
					EnvironmentID: selection.TargetEnvironmentID,
					WorkspaceID:   selection.WorkspaceID,
					ProjectID:     selection.ProjectID,
				})
			}
		}
	}

	found, err := queries.FindPrivateNetworkPreviewDeploymentIds(ctx, previews.keys)
	if err != nil {
		return nil, err
	}
	previews.assign(deploymentIDs, found)
	found, err = queries.FindPrivateNetworkEnvironmentDeploymentIds(ctx, environments.keys)
	if err != nil {
		return nil, err
	}
	environments.assign(deploymentIDs, found)
	return deploymentIDs, nil
}

func readyTargets(ctx context.Context, queries *db.Queries, deploymentIDs []string, platform string) (map[string]db.ListPrivateNetworkTargetsRow, error) {
	ids := slices.DeleteFunc(slices.Compact(slices.Sorted(slices.Values(deploymentIDs))), func(id string) bool { return id == "" })
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := queries.ListPrivateNetworkTargets(ctx, db.ListPrivateNetworkTargetsParams{DeploymentIds: ids, Platform: platform})
	if err != nil {
		return nil, err
	}
	targets := make(map[string]db.ListPrivateNetworkTargetsRow, len(rows))
	for _, row := range rows {
		targets[row.ID] = row
	}
	return targets, nil
}

type lookups[K comparable] struct {
	keys       []K
	slotByKey  map[K]int
	selections []int
	slots      []int
}

func (l *lookups[K]) add(selection int, key K) {
	slot, ok := l.slotByKey[key]
	if !ok {
		if l.slotByKey == nil {
			l.slotByKey = make(map[K]int)
		}
		slot = len(l.keys)
		l.slotByKey[key] = slot
		l.keys = append(l.keys, key)
	}
	l.selections = append(l.selections, selection)
	l.slots = append(l.slots, slot)
}

func (l *lookups[K]) assign(deploymentIDs, found []string) {
	for i, selection := range l.selections {
		deploymentIDs[selection] = found[l.slots[i]]
	}
}
