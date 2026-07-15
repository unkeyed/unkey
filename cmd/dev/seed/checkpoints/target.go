package checkpoints

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/prompt"
	"github.com/unkeyed/unkey/pkg/uid"
	"golang.org/x/term"
)

// target is the resolved scope every checkpoint is attributed to.
type target struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	resourceID    string // deployment id
	// declared allocation from the deployment, if one was found.
	deployCPUMilli int32
	deployMemBytes int64
}

func (t target) cpuAllocMilli(vcpu float64) int32 {
	if t.deployCPUMilli > 0 {
		return t.deployCPUMilli
	}
	return int32(math.Ceil(vcpu * 1000))
}

func (t target) memAllocBytes(usage int64) int64 {
	if t.deployMemBytes > 0 {
		return t.deployMemBytes
	}
	return usage
}

// resolveTarget validates the workspace exists, then resolves the
// project/app/environment (interactively when not given as flags) and picks a
// deployment to attribute usage to.
func resolveTarget(ctx context.Context, database db.Database, cmd *cli.Command) (target, error) {
	workspaceID := cmd.RequireString("workspace")
	if _, err := db.Query.FindWorkspaceByID(ctx, database.RO(), workspaceID); err != nil {
		return target{}, fmt.Errorf("workspace %q not found: %w", workspaceID, err)
	}

	projectID, err := resolveProject(ctx, database, workspaceID, cmd.String("project"))
	if err != nil {
		return target{}, err
	}
	appID, err := resolveApp(ctx, database, projectID, cmd.String("app"))
	if err != nil {
		return target{}, err
	}
	envID, err := resolveEnvironment(ctx, database, appID, cmd.String("environment"))
	if err != nil {
		return target{}, err
	}

	t := target{workspaceID: workspaceID, projectID: projectID, appID: appID, environmentID: envID} //nolint:exhaustruct

	if deploymentID := cmd.String("deployment"); deploymentID != "" {
		t.resourceID = deploymentID
		return t, nil
	}

	dep, err := latestDeployment(ctx, database, envID)
	if err != nil {
		return target{}, err
	}
	if dep == nil {
		t.resourceID = uid.New(uid.DeploymentPrefix)
		logger.Info("no deployment found in environment; using synthetic resource id", "resource_id", t.resourceID)
		return t, nil
	}

	t.resourceID = dep.id
	t.deployCPUMilli = dep.cpuMilli
	t.deployMemBytes = int64(dep.memMib) * 1024 * 1024
	logger.Info("using existing deployment", "deployment_id", dep.id, "cpu_millicores", dep.cpuMilli, "memory_mib", dep.memMib)
	return t, nil
}

func resolveProject(ctx context.Context, database db.Database, workspaceID, input string) (string, error) {
	rows, err := queryChoices(ctx, database,
		"SELECT id, slug FROM projects WHERE workspace_id = ? ORDER BY created_at DESC", workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no projects found in workspace %s", workspaceID)
	}
	return pick("project", rows, input)
}

func resolveApp(ctx context.Context, database db.Database, projectID, input string) (string, error) {
	rows, err := queryChoices(ctx, database,
		"SELECT id, slug FROM apps WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return "", fmt.Errorf("failed to list apps: %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no apps found in project %s", projectID)
	}
	return pick("app", rows, input)
}

func resolveEnvironment(ctx context.Context, database db.Database, appID, input string) (string, error) {
	rows, err := queryChoices(ctx, database,
		"SELECT id, slug FROM environments WHERE app_id = ? ORDER BY created_at DESC", appID)
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no environments found in app %s", appID)
	}
	return pick("environment", rows, input)
}

// deploymentInfo holds the few deployment columns this seeder uses for
// allocation defaults.
type deploymentInfo struct {
	id       string
	cpuMilli int32
	memMib   int32
}

func latestDeployment(ctx context.Context, database db.Database, envID string) (*deploymentInfo, error) {
	row := database.RO().QueryRowContext(ctx,
		"SELECT id, cpu_millicores, memory_mib FROM deployments WHERE environment_id = ? ORDER BY created_at DESC LIMIT 1", envID)
	var dep deploymentInfo
	err := row.Scan(&dep.id, &dep.cpuMilli, &dep.memMib)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read latest deployment: %w", err)
	}
	return &dep, nil
}

type choice struct {
	id    string
	label string
}

func queryChoices(ctx context.Context, database db.Database, query string, args ...any) ([]choice, error) {
	rows, err := database.RO().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []choice
	for rows.Next() {
		var c choice
		if err := rows.Scan(&c.id, &c.label); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// pick resolves a choice from explicit user input (id or slug), or prompts
// interactively when input is empty. On a non-interactive stdin it falls back
// to the most recent entry so scripted runs still work.
func pick(kind string, choices []choice, input string) (string, error) {
	if input != "" {
		for _, c := range choices {
			if c.id == input || c.label == input {
				return c.id, nil
			}
		}
		return "", fmt.Errorf("%s %q not found among: %s", kind, input, joinChoices(choices))
	}
	if len(choices) == 1 {
		logger.Info("using only "+kind, "slug", choices[0].label, "id", choices[0].id)
		return choices[0].id, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		logger.Info("non-interactive: defaulting "+kind+" to most recent", "slug", choices[0].label, "id", choices[0].id)
		return choices[0].id, nil
	}

	opts := make([]prompt.SelectOption, len(choices))
	for i, c := range choices {
		opts[i] = prompt.SelectOption{Key: c.id, Label: fmt.Sprintf("%s (%s)", c.label, c.id)}
	}
	id, err := prompt.New().SelectOrdered("Select "+kind, opts, choices[0].id)
	if err != nil {
		return "", fmt.Errorf("failed to select %s: %w", kind, err)
	}
	for _, c := range choices {
		if c.id == id {
			// The picker leaves no record of the choice on screen, so log it
			// for the scrollback.
			logger.Info("selected "+kind, "slug", c.label, "id", c.id)
			break
		}
	}
	return id, nil
}

func joinChoices(choices []choice) string {
	parts := make([]string, len(choices))
	for i, c := range choices {
		parts[i] = fmt.Sprintf("%s (%s)", c.label, c.id)
	}
	return strings.Join(parts, ", ")
}
