package environment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// AppScope identifies the app that owns newly created environments. Callers
// derive it from an app they just inserted or loaded from the database.
type AppScope struct {
	WorkspaceID string
	ProjectID   string
	AppID       string
}

// CreateSpec describes one environment to create.
type CreateSpec struct {
	ID          string
	Slug        string
	Description string
}

// CreateManyParams contains the complete environment creation batch.
type CreateManyParams struct {
	App          AppScope
	Environments []CreateSpec
	Now          int64
}

// CreateMany inserts environments and all required settings using the caller's
// transaction. It performs one region read and a fixed number of bulk writes,
// regardless of the number of environments or regions.
func CreateMany(ctx context.Context, tx db.DBTX, params CreateManyParams) error {
	availableRegions, err := db.NewQueries(tx).ListRegions(ctx)
	if err != nil {
		return fmt.Errorf("list regions: %w", err)
	}

	schedulableRegionIDs := make([]string, 0, len(availableRegions))
	for _, region := range availableRegions {
		if region.CanSchedule {
			schedulableRegionIDs = append(schedulableRegionIDs, region.ID)
		}
	}
	if len(schedulableRegionIDs) == 0 {
		return errors.New("no schedulable regions available")
	}

	writePlan := planWrites(params, schedulableRegionIDs)
	bulk := db.NewBulkQueries(tx)
	if err = bulk.InsertEnvironments(ctx, writePlan.environments); err != nil {
		return fmt.Errorf("insert environments: %w", err)
	}
	if err = bulk.UpsertAppBuildSettings(ctx, writePlan.buildSettings); err != nil {
		return fmt.Errorf("upsert build settings: %w", err)
	}
	if err = bulk.UpsertAppRuntimeSettings(ctx, writePlan.runtimeSettings); err != nil {
		return fmt.Errorf("upsert runtime settings: %w", err)
	}
	if err = bulk.UpsertAppRegionalSettings(ctx, writePlan.regionalSettings); err != nil {
		return fmt.Errorf("upsert regional settings: %w", err)
	}

	return nil
}

type writes struct {
	environments     []db.InsertEnvironmentParams
	buildSettings    []db.UpsertAppBuildSettingsParams
	runtimeSettings  []db.UpsertAppRuntimeSettingsParams
	regionalSettings []db.UpsertAppRegionalSettingsParams
}

func planWrites(params CreateManyParams, schedulableRegionIDs []string) writes {
	planned := writes{
		environments:     make([]db.InsertEnvironmentParams, 0, len(params.Environments)),
		buildSettings:    make([]db.UpsertAppBuildSettingsParams, 0, len(params.Environments)),
		runtimeSettings:  make([]db.UpsertAppRuntimeSettingsParams, 0, len(params.Environments)),
		regionalSettings: make([]db.UpsertAppRegionalSettingsParams, 0, len(params.Environments)*len(schedulableRegionIDs)),
	}

	for _, environment := range params.Environments {
		planned.environments = append(planned.environments, db.InsertEnvironmentParams{
			ID:          environment.ID,
			WorkspaceID: params.App.WorkspaceID,
			ProjectID:   params.App.ProjectID,
			AppID:       params.App.AppID,
			Slug:        environment.Slug,
			Description: environment.Description,
			CreatedAt:   params.Now,
			UpdatedAt:   sql.NullInt64{Valid: false},
		})

		planned.buildSettings = append(planned.buildSettings, db.UpsertAppBuildSettingsParams{
			WorkspaceID:   params.App.WorkspaceID,
			AppID:         params.App.AppID,
			EnvironmentID: environment.ID,
			Dockerfile:    sql.NullString{String: "", Valid: false},
			DockerContext: ".",
			BuildCommand:  sql.NullString{String: "", Valid: false},
			WatchPaths:    nil,
			AutoDeploy:    true,
			CreatedAt:     params.Now,
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: params.Now},
		})

		planned.runtimeSettings = append(planned.runtimeSettings, db.UpsertAppRuntimeSettingsParams{
			WorkspaceID:      params.App.WorkspaceID,
			AppID:            params.App.AppID,
			EnvironmentID:    environment.ID,
			Port:             8080,
			CpuMillicores:    250,
			MemoryMib:        256,
			StorageMib:       0,
			Command:          dbtype.StringSlice{},
			Healthcheck:      dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
			ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
			UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
			SentinelConfig:   []byte("{}"),
			OpenapiSpecPath:  sql.NullString{String: "", Valid: false},
			CreatedAt:        params.Now,
			UpdatedAt:        sql.NullInt64{Valid: true, Int64: params.Now},
		})

		for _, regionID := range schedulableRegionIDs {
			planned.regionalSettings = append(planned.regionalSettings, db.UpsertAppRegionalSettingsParams{
				WorkspaceID:   params.App.WorkspaceID,
				AppID:         params.App.AppID,
				EnvironmentID: environment.ID,
				RegionID:      regionID,
				Replicas:      1,
				CreatedAt:     params.Now,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: params.Now},
			})
		}
	}

	return planned
}
