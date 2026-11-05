package privatenetwork

import (
	"context"
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
	previews := make(map[previewKey][]int)
	environments := make(map[environmentKey][]int)
	var previewCallers []string
	for i, selection := range selections {
		switch selection.SelectionMode {
		case db.DeploymentConnectionAppTargetsSelectionModeDeployment:
			deploymentIDs[i] = selection.TargetDeploymentID
		case db.DeploymentConnectionAppTargetsSelectionModeAutomatic:
			switch {
			case selection.CallerEnvironmentKind == mysqltype.EnvironmentKindProduction:
				deploymentIDs[i] = selection.LiveDeploymentID
			case selection.CallerSource == db.DeploymentsSourceGit && selection.CallerGitBranch != "":
				key := previewKey{
					workspaceID: selection.WorkspaceID,
					projectID:   selection.ProjectID,
					appID:       selection.AppID,
					gitBranch:   selection.CallerGitBranch,
					forkKey:     selection.CallerForkKey,
				}
				previews[key] = append(previews[key], i)
				previewCallers = append(previewCallers, selection.CallerDeploymentID)
			}
		case db.DeploymentConnectionAppTargetsSelectionModeEnvironment:
			switch {
			case selection.TargetEnvironmentID == "":
			case selection.TargetEnvironmentID == selection.LiveEnvironmentID:
				deploymentIDs[i] = selection.LiveDeploymentID
			default:
				key := environmentKey{
					workspaceID:   selection.WorkspaceID,
					projectID:     selection.ProjectID,
					appID:         selection.AppID,
					environmentID: selection.TargetEnvironmentID,
				}
				environments[key] = append(environments[key], i)
			}
		}
	}

	if len(previews) > 0 {
		rows, err := queries.ListPrivateNetworkPreviewDeploymentIds(ctx, db.ListPrivateNetworkPreviewDeploymentIdsParams{
			CallerDeploymentIds: slices.Compact(slices.Sorted(slices.Values(previewCallers))),
			AppIds:              distinct(previews, func(key previewKey) string { return key.appID }),
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			for _, i := range previews[previewKey{
				workspaceID: row.WorkspaceID,
				projectID:   row.ProjectID,
				appID:       row.AppID,
				gitBranch:   row.GitBranch,
				forkKey:     row.ForkKey,
			}] {
				deploymentIDs[i] = row.ID
			}
		}
	}
	if len(environments) > 0 {
		rows, err := queries.ListPrivateNetworkEnvironmentDeploymentIds(ctx,
			distinct(environments, func(key environmentKey) string { return key.environmentID }))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			for _, i := range environments[environmentKey{
				workspaceID:   row.WorkspaceID,
				projectID:     row.ProjectID,
				appID:         row.AppID,
				environmentID: row.EnvironmentID,
			}] {
				deploymentIDs[i] = row.ID
			}
		}
	}
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

type previewKey struct {
	workspaceID, projectID, appID, gitBranch, forkKey string
}

type environmentKey struct {
	workspaceID, projectID, appID, environmentID string
}

func distinct[K comparable](keys map[K][]int, field func(K) string) []string {
	values := make([]string, 0, len(keys))
	for key := range keys {
		values = append(values, field(key))
	}
	slices.Sort(values)
	return slices.Compact(values)
}
