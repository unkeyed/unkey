package app

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// envSpec defines a default environment created with an app.
type envSpec struct {
	slug        string
	description string
	kind        dbtype.EnvironmentKind
}

// defaultEnvironments are the environments created automatically for every new app.
var defaultEnvironments = []envSpec{
	{slug: "production", description: "Production", kind: dbtype.EnvironmentKindProduction},
	{slug: "preview", description: "Preview", kind: dbtype.EnvironmentKindPreview},
}

type createAppSource struct {
	sourceType          db.AppsSourceType
	imageReference      string
	createBuildSettings bool
}

func parseCreateAppSource(req *ctrlv1.CreateAppRequest) (createAppSource, error) {
	switch source := req.GetSource().(type) {
	case nil:
		return createAppSource{
			sourceType:          db.AppsSourceTypeUnknown,
			imageReference:      "",
			createBuildSettings: true,
		}, nil
	case *ctrlv1.CreateAppRequest_Git:
		return createAppSource{
			sourceType:          db.AppsSourceTypeGit,
			imageReference:      "",
			createBuildSettings: true,
		}, nil
	case *ctrlv1.CreateAppRequest_Docker:
		imageReference, err := imageref.Normalize(source.Docker.GetImageReference())
		if err != nil {
			return createAppSource{}, err
		}
		return createAppSource{
			sourceType:          db.AppsSourceTypeDocker,
			imageReference:      imageReference,
			createBuildSettings: false,
		}, nil
	default:
		return createAppSource{}, fmt.Errorf("unsupported app source %T", source)
	}
}

// CreateApp creates an app with default environments and their
// build/runtime settings in a single transaction.
func (s *Service) CreateApp(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateAppRequest],
) (*connect.Response[ctrlv1.CreateAppResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"),
		assert.NotEmpty(req.Msg.GetName(), "name is required"),
		assert.NotEmpty(req.Msg.GetSlug(), "slug is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	source, err := parseCreateAppSource(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workspaceID := req.Msg.GetWorkspaceId()
	projectID := req.Msg.GetProjectId()
	appID := uid.New(uid.AppPrefix)
	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.NewQueries(tx).InsertApp(txCtx, db.InsertAppParams{
			ID:               appID,
			WorkspaceID:      workspaceID,
			ProjectID:        projectID,
			Name:             req.Msg.GetName(),
			Slug:             req.Msg.GetSlug(),
			SourceType:       source.sourceType,
			DefaultBranch:    "main",
			DeleteProtection: sql.NullBool{Valid: false},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{Valid: false},
		}); txErr != nil {
			return fmt.Errorf("insert app: %w", txErr)
		}

		if source.sourceType == db.AppsSourceTypeDocker {
			if txErr := db.NewQueries(tx).InsertAppDockerSource(txCtx, db.InsertAppDockerSourceParams{
				WorkspaceID:    workspaceID,
				AppID:          appID,
				ImageReference: source.imageReference,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Valid: false},
			}); txErr != nil {
				return fmt.Errorf("insert app docker source: %w", txErr)
			}
		}

		// Pick a default schedulable region to seed so a fresh environment is
		// deployable without a separate region step. Regions are infra-registered,
		// so if none are schedulable yet we skip; the deploy-time check reports it.
		regions, regErr := db.NewQueries(tx).ListRegions(txCtx)
		if regErr != nil {
			return fmt.Errorf("list regions: %w", regErr)
		}
		defaultRegionID, hasDefaultRegion := pickDefaultRegion(regions)

		for _, env := range defaultEnvironments {
			envID := uid.New(uid.EnvironmentPrefix)

			if txErr := db.NewQueries(tx).InsertEnvironment(txCtx, db.InsertEnvironmentParams{
				ID:          envID,
				WorkspaceID: workspaceID,
				ProjectID:   projectID,
				AppID:       appID,
				Slug:        env.slug,
				Description: env.description,
				Kind:        env.kind,
				CreatedAt:   now,
				UpdatedAt:   sql.NullInt64{Valid: false},
			}); txErr != nil {
				return fmt.Errorf("insert %s environment: %w", env.slug, txErr)
			}

			if source.createBuildSettings {
				if txErr := db.NewQueries(tx).UpsertAppBuildSettings(txCtx, db.UpsertAppBuildSettingsParams{
					WorkspaceID:   workspaceID,
					AppID:         appID,
					EnvironmentID: envID,
					Dockerfile:    sql.NullString{Valid: false, String: ""},
					DockerContext: "",
					BuildCommand:  sql.NullString{Valid: false, String: ""},
					WatchPaths:    nil,
					AutoDeploy:    true,
					CreatedAt:     now,
					UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
				}); txErr != nil {
					return fmt.Errorf("upsert %s build settings: %w", env.slug, txErr)
				}
			}

			if txErr := db.NewQueries(tx).UpsertAppRuntimeSettings(txCtx, db.UpsertAppRuntimeSettingsParams{
				WorkspaceID:      workspaceID,
				AppID:            appID,
				EnvironmentID:    envID,
				Port:             8080,
				CpuMillicores:    250,
				MemoryMib:        256,
				StorageMib:       0,
				Command:          dbtype.StringSlice{},
				Healthcheck:      dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
				ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
				UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
				SentinelConfig:   []byte("{}"),
				OpenapiSpecPath:  sql.NullString{Valid: false},
				CreatedAt:        now,
				UpdatedAt:        sql.NullInt64{Valid: true, Int64: now},
			}); txErr != nil {
				return fmt.Errorf("upsert %s runtime settings: %w", env.slug, txErr)
			}

			if hasDefaultRegion {
				if txErr := db.NewQueries(tx).UpsertAppRegionalSettings(txCtx, db.UpsertAppRegionalSettingsParams{
					WorkspaceID:   workspaceID,
					AppID:         appID,
					EnvironmentID: envID,
					RegionID:      defaultRegionID,
					Replicas:      1,
					CreatedAt:     now,
					UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
				}); txErr != nil {
					return fmt.Errorf("upsert %s regional settings: %w", env.slug, txErr)
				}
			}
		}

		a := req.Msg.GetActor()
		if txErr := s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   workspaceID,
				Event:         auditlog.AppCreateEvent,
				Display:       fmt.Sprintf("Created app %s", appID),
				ActorID:       a.GetId(),
				ActorName:     a.GetName(),
				ActorType:     actor.AuditType(a.GetType()),
				ActorMeta:     actor.Meta(a.GetMeta()),
				RemoteIP:      a.GetRemoteIp(),
				UserAgent:     a.GetUserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:   appID,
						Type: auditlog.AppResourceType,
						Meta: map[string]any{
							"name":       req.Msg.GetName(),
							"slug":       req.Msg.GetSlug(),
							"projectId":  projectID,
							"sourceType": string(source.sourceType),
						},
						Name:        req.Msg.GetName(),
						DisplayName: req.Msg.GetName(),
					},
				},
			},
		}); txErr != nil {
			return fmt.Errorf("insert audit log: %w", txErr)
		}

		return nil
	})
	if err != nil {
		if db.IsDuplicateKeyError(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("app with slug %q already exists in project", req.Msg.GetSlug()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create app: %w", err))
	}

	return connect.NewResponse(&ctrlv1.CreateAppResponse{
		Id: appID,
	}), nil
}

// pickDefaultRegion returns the id of the schedulable region whose name sorts
// first lexically, or ("", false) when none can be scheduled. Sorting by name
// keeps the choice deterministic without depending on a fixed region being up.
func pickDefaultRegion(regions []db.ListRegionsRow) (string, bool) {
	schedulable := make([]db.ListRegionsRow, 0, len(regions))
	for _, r := range regions {
		if r.CanSchedule {
			schedulable = append(schedulable, r)
		}
	}
	if len(schedulable) == 0 {
		return "", false
	}
	slices.SortFunc(schedulable, func(a, b db.ListRegionsRow) int {
		return strings.Compare(a.Name, b.Name)
	})
	return schedulable[0].ID, true
}
