package version

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/urfave/cli/v3"
)

// TODO: REMOVE THIS ENTIRE FILE - This is a temporary bootstrap helper
// Remove once we have proper UI for project management

var bootstrapProjectCmd = &cli.Command{
	Name:  "bootstrap-project",
	Usage: "TEMPORARY: Create a project for testing (remove once we have UI)",
	Description: `TEMPORARY BOOTSTRAP HELPER - REMOVE ONCE WE HAVE PROPER UI

This command directly creates a project in the database for testing purposes.
This bypasses proper API workflows and should be removed once we have a UI.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "workspace-id",
			Usage:    "Workspace ID",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "slug",
			Usage:    "Project slug",
			Value:    "my-api",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "db-url",
			Usage:    "Database connection string",
			Value:    "root:password@tcp(localhost:3306)/unkey",
			Required: false,
		},
	},
	Action: bootstrapProjectAction,
}

func bootstrapProjectAction(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	workspaceID := cmd.String("workspace-id")
	projectSlug := cmd.String("slug")
	dbURL := cmd.String("db-url")

	fmt.Printf("🚧 TEMPORARY BOOTSTRAP - Creating project...\n")
	fmt.Printf("   Workspace ID: %s\n", workspaceID)
	fmt.Printf("   Project Slug: %s\n", projectSlug)
	fmt.Println()

	// Connect to database (TEMPORARY - this should be done via API)
	sqlDB, err := sql.Open("mysql", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer sqlDB.Close()

	// Create workspace if it doesn't exist
	_, err = db.Query.FindWorkspaceByID(ctx, sqlDB, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			// Workspace doesn't exist, create it
			fmt.Printf("📁 Creating workspace: %s\n", workspaceID)
			now := time.Now().UnixMilli()
			err = db.Query.InsertWorkspace(ctx, sqlDB, db.InsertWorkspaceParams{
				ID:        workspaceID,
				OrgID:     "org_bootstrap", // hardcoded for bootstrap
				Name:      workspaceID,     // use ID as name for simplicity
				CreatedAt: now,
			})
			if err != nil {
				return fmt.Errorf("failed to create workspace: %w", err)
			}
			fmt.Printf("✅ Workspace created: %s\n", workspaceID)
		} else {
			return fmt.Errorf("failed to validate workspace: %w", err)
		}
	} else {
		fmt.Printf("📁 Using existing workspace: %s\n", workspaceID)
	}

	// Check if project already exists
	_, err = db.Query.FindProjectByWorkspaceSlug(ctx, sqlDB, db.FindProjectByWorkspaceSlugParams{
		WorkspaceID: workspaceID,
		Slug:        projectSlug,
	})
	if err == nil {
		return fmt.Errorf("project with slug '%s' already exists in workspace '%s'", projectSlug, workspaceID)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing project: %w", err)
	}

	// Generate project ID
	projectID := uid.New("proj")

	// Create project
	now := time.Now().UnixMilli()
	err = db.Query.InsertProject(ctx, sqlDB, db.InsertProjectParams{
		ID:               projectID,
		WorkspaceID:      workspaceID,
		PartitionID:      "part_default", // hardcoded for now
		Name:             projectSlug,    // use slug as name for simplicity
		Slug:             projectSlug,
		GitRepositoryUrl: sql.NullString{String: "", Valid: false},
		DefaultBranch:    sql.NullString{String: "main", Valid: true},
		DeleteProtection: sql.NullBool{Bool: false, Valid: true},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Printf("✅ Project created successfully!\n")
	fmt.Printf("   Project ID: %s\n", projectID)
	fmt.Printf("   Workspace ID: %s\n", workspaceID)
	fmt.Printf("   Project Slug: %s\n", projectSlug)
	fmt.Println()

	fmt.Printf("📋 Use these values for deployment:\n")
	fmt.Printf("   unkey-cli create --workspace-id=%s --project-id=%s\n", workspaceID, projectID)
	fmt.Printf("\n")
	fmt.Printf("🗑️  Remember to remove this bootstrap command once we have proper UI!\n")

	logger.Info("bootstrap project created",
		"project_id", projectID,
		"workspace_id", workspaceID,
		"project_slug", projectSlug)

	return nil
}
