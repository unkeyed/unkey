package seed

import (
	"context"
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

// EnvironmentTopology identifies a project, app and environment in one workspace.
type EnvironmentTopology struct {
	WorkspaceID   string
	ProjectID     string
	ProjectSlug   string
	AppID         string
	AppSlug       string
	EnvironmentID string
}

// CreateEnvironmentTopology creates a project, app, environment and default settings.
// Workspace and kind are required; credentials and billing entitlements are not created.
func (s *Seeder) CreateEnvironmentTopology(ctx context.Context, workspaceID string, kind mysqltype.EnvironmentKind) EnvironmentTopology {
	s.t.Helper()
	require.NotEmpty(s.t, workspaceID)
	require.Contains(s.t, []mysqltype.EnvironmentKind{
		mysqltype.EnvironmentKindProduction,
		mysqltype.EnvironmentKindPreview,
	}, kind)

	project := s.CreateProject(ctx, CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspaceID,
		Name:             "Payments Service",
		Slug:             strings.ToLower(strings.ReplaceAll(uid.New(uid.TestPrefix), "_", "-")),
		DeleteProtection: false,
	})

	app := s.CreateApp(ctx, CreateAppRequest{
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		Name:             "Payments API",
		Slug:             strings.ToLower(strings.ReplaceAll(uid.New(uid.TestPrefix), "_", "-")),
		SourceType:       db.AppsSourceTypeUnknown,
		ImageReference:   "",
		DeleteProtection: false,
	})

	description := "Preview environment"
	if kind == mysqltype.EnvironmentKindProduction {
		description = "Production environment"
	}
	environment := s.CreateEnvironment(ctx, CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             string(kind),
		Kind:             kind,
		Description:      description,
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	return EnvironmentTopology{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		ProjectSlug:   project.Slug,
		AppID:         app.ID,
		AppSlug:       app.Slug,
		EnvironmentID: environment.ID,
	}
}
