package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2EnvironmentsUpdateSettingsRequestBody
	Response = openapi.V2EnvironmentsUpdateSettingsResponseBody
)

// maxReplicas mirrors REPLICAS_MAX_BETA in the dashboard.
const maxReplicas = 4

// cpuThreshold is the fixed autoscaling CPU threshold; the API does not expose it.
const cpuThreshold = 80

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.updateSettings"
}

// resolvedRegion is a desired region after validation against the regions table.
type resolvedRegion struct {
	regionID string
	min      int32
	max      int32
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	env, err := db.Query.FindEnvironmentByAppAndIdOrSlug(ctx, h.DB.RO(), db.FindEnvironmentByAppAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	environment := env.Environment

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.UpdateEnvironment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   environment.ID,
			Action:       rbac.UpdateEnvironment,
		}),
	))
	if err != nil {
		return err
	}

	hasBuild := req.Dockerfile.IsSpecified() || req.DockerContext != nil ||
		req.WatchPaths != nil || req.AutoDeploy != nil
	hasRuntime := req.Port != nil || req.CpuMillicores != nil || req.MemoryMib != nil ||
		req.StorageMib != nil || req.Command != nil || req.Healthcheck.IsSpecified() ||
		req.ShutdownSignal != nil || req.UpstreamProtocol != nil || req.OpenapiSpecPath.IsSpecified()

	// Nothing to do: skip the write transaction and audit log entirely.
	if !hasBuild && !hasRuntime && req.Regions == nil {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.EmptyResponse{},
		})
	}

	// Resolve and validate regions before opening the write transaction so we
	// never leave a partial write behind on bad input.
	var desired []resolvedRegion
	if req.Regions != nil {
		desired, err = h.resolveRegions(ctx, *req.Regions)
		if err != nil {
			return err
		}
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		if hasBuild {
			if err := h.applyBuildSettings(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, req); err != nil {
				return err
			}
		}

		if hasRuntime {
			if err := h.applyRuntimeSettings(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, req); err != nil {
				return err
			}
		}

		if req.Regions != nil {
			if err := h.applyRegions(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, desired); err != nil {
				return err
			}
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.EnvironmentUpdateEvent,
				Display:       fmt.Sprintf("Updated settings for environment %s", environment.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          environment.ID,
						Type:        auditlog.EnvironmentResourceType,
						Meta:        map[string]any{},
						Name:        environment.Slug,
						DisplayName: environment.Slug,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func (h *Handler) applyBuildSettings(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, req Request) error {
	params := db.UpdateAppBuildSettingsParams{
		WorkspaceID:            workspaceID,
		AppID:                  appID,
		EnvironmentID:          environmentID,
		UpdatedAt:              sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		DockerfileSpecified:    0,
		Dockerfile:             sql.NullString{Valid: false, String: ""},
		DockerContextSpecified: 0,
		DockerContext:          "",
		WatchPathsSpecified:    0,
		WatchPaths:             nil,
		AutoDeploySpecified:    0,
		AutoDeploy:             false,
	}

	if req.Dockerfile.IsSpecified() {
		params.DockerfileSpecified = 1
		if !req.Dockerfile.IsNull() {
			v, _ := req.Dockerfile.Get()
			params.Dockerfile = sql.NullString{Valid: true, String: v}
		}
	}
	if req.DockerContext != nil {
		params.DockerContextSpecified = 1
		params.DockerContext = *req.DockerContext
	}
	if req.WatchPaths != nil {
		params.WatchPathsSpecified = 1
		params.WatchPaths = dbtype.StringSlice(*req.WatchPaths)
	}
	if req.AutoDeploy != nil {
		params.AutoDeploySpecified = 1
		params.AutoDeploy = *req.AutoDeploy
	}

	if err := db.Query.UpdateAppBuildSettings(ctx, tx, params); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("unable to update build settings"),
			fault.Public("We're unable to update the environment settings."),
		)
	}
	return nil
}

func (h *Handler) applyRuntimeSettings(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, req Request) error {
	params := db.UpdateAppRuntimeSettingsParams{
		WorkspaceID:               workspaceID,
		AppID:                     appID,
		EnvironmentID:             environmentID,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		PortSpecified:             0,
		Port:                      0,
		CpuMillicoresSpecified:    0,
		CpuMillicores:             0,
		MemoryMibSpecified:        0,
		MemoryMib:                 0,
		StorageMibSpecified:       0,
		StorageMib:                0,
		CommandSpecified:          0,
		Command:                   nil,
		HealthcheckSpecified:      0,
		Healthcheck:               dbtype.NullHealthcheck{Valid: false, Healthcheck: nil},
		ShutdownSignalSpecified:   0,
		ShutdownSignal:            "",
		UpstreamProtocolSpecified: 0,
		UpstreamProtocol:          "",
		OpenapiSpecPathSpecified:  0,
		OpenapiSpecPath:           sql.NullString{Valid: false, String: ""},
	}

	if req.Port != nil {
		params.PortSpecified = 1
		params.Port = int32(*req.Port)
	}
	if req.CpuMillicores != nil {
		params.CpuMillicoresSpecified = 1
		params.CpuMillicores = int32(*req.CpuMillicores)
	}
	if req.MemoryMib != nil {
		params.MemoryMibSpecified = 1
		params.MemoryMib = int32(*req.MemoryMib)
	}
	if req.StorageMib != nil {
		params.StorageMibSpecified = 1
		params.StorageMib = uint32(*req.StorageMib)
	}
	if req.Command != nil {
		params.CommandSpecified = 1
		params.Command = dbtype.StringSlice(*req.Command)
	}
	if req.Healthcheck.IsSpecified() {
		params.HealthcheckSpecified = 1
		if !req.Healthcheck.IsNull() {
			hc, _ := req.Healthcheck.Get()
			params.Healthcheck = dbtype.NullHealthcheck{Valid: true, Healthcheck: buildHealthcheck(hc)}
		}
	}
	if req.ShutdownSignal != nil {
		params.ShutdownSignalSpecified = 1
		params.ShutdownSignal = db.AppRuntimeSettingsShutdownSignal(*req.ShutdownSignal)
	}
	if req.UpstreamProtocol != nil {
		params.UpstreamProtocolSpecified = 1
		params.UpstreamProtocol = db.AppRuntimeSettingsUpstreamProtocol(*req.UpstreamProtocol)
	}
	if req.OpenapiSpecPath.IsSpecified() {
		params.OpenapiSpecPathSpecified = 1
		if !req.OpenapiSpecPath.IsNull() {
			v, _ := req.OpenapiSpecPath.Get()
			params.OpenapiSpecPath = sql.NullString{Valid: true, String: v}
		}
	}

	if err := db.Query.UpdateAppRuntimeSettings(ctx, tx, params); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("unable to update runtime settings"),
			fault.Public("We're unable to update the environment settings."),
		)
	}
	return nil
}

// resolveRegions validates the desired region list against the regions table and
// the replica bounds, returning the resolved region ids. Validation happens before
// the write transaction so bad input never leaves a partial write.
func (h *Handler) resolveRegions(ctx context.Context, regions []openapi.RegionSetting) ([]resolvedRegion, error) {
	seen := make(map[string]struct{}, len(regions))
	resolved := make([]resolvedRegion, 0, len(regions))

	for _, r := range regions {
		platform := "aws"
		if r.Platform != nil && *r.Platform != "" {
			platform = *r.Platform
		}

		key := platform + "/" + r.Name
		if _, dup := seen[key]; dup {
			return nil, fault.New(
				"duplicate region",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("duplicate region in request"),
				fault.Public(fmt.Sprintf("Region '%s' on platform '%s' is listed more than once.", r.Name, platform)),
			)
		}
		seen[key] = struct{}{}

		minReplicas := int32(r.Replicas.Min)
		maxReplicasReq := int32(r.Replicas.Max)
		if minReplicas < 1 || maxReplicasReq > maxReplicas || minReplicas > maxReplicasReq {
			return nil, fault.New(
				"invalid replicas",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("invalid replica bounds"),
				fault.Public(fmt.Sprintf("Region '%s' replicas must satisfy 1 <= min <= max <= %d.", r.Name, maxReplicas)),
			)
		}

		region, err := db.Query.FindRegionByPlatformAndName(ctx, h.DB.RO(), db.FindRegionByPlatformAndNameParams{
			Platform: platform,
			Name:     r.Name,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return nil, fault.New(
					"region not found",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("region not found"),
					fault.Public(fmt.Sprintf("Region '%s' on platform '%s' does not exist.", r.Name, platform)),
				)
			}
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to resolve regions."),
			)
		}

		if !region.CanSchedule {
			return nil, fault.New(
				"region not schedulable",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("region cannot be scheduled to"),
				fault.Public(fmt.Sprintf("Region '%s' on platform '%s' is not available for scheduling.", r.Name, platform)),
			)
		}

		resolved = append(resolved, resolvedRegion{
			regionID: region.ID,
			min:      minReplicas,
			max:      maxReplicasReq,
		})
	}

	return resolved, nil
}

// applyRegions reconciles the desired regions against the current rows: it updates
// or creates one autoscaling policy per region, sets replicas to max, and removes
// regions (and their orphaned policies) no longer desired.
func (h *Handler) applyRegions(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, desired []resolvedRegion) error {
	current, err := db.Query.ListAppRegionalSettingsByAppEnv(ctx, tx, db.ListAppRegionalSettingsByAppEnvParams{
		AppID:         appID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("unable to list regional settings"),
			fault.Public("We're unable to update the environment settings."),
		)
	}

	currentByRegion := make(map[string]db.ListAppRegionalSettingsByAppEnvRow, len(current))
	for _, row := range current {
		currentByRegion[row.RegionID] = row
	}
	desiredByRegion := make(map[string]struct{}, len(desired))

	now := time.Now().UnixMilli()

	for _, d := range desired {
		desiredByRegion[d.regionID] = struct{}{}

		policyID := ""
		if existing, ok := currentByRegion[d.regionID]; ok && existing.HorizontalAutoscalingPolicyID.Valid {
			policyID = existing.HorizontalAutoscalingPolicyID.String
			if err := db.Query.UpdateHorizontalAutoscalingPolicy(ctx, tx, db.UpdateHorizontalAutoscalingPolicyParams{
				ID:          policyID,
				WorkspaceID: workspaceID,
				ReplicasMin: d.min,
				ReplicasMax: d.max,
				UpdatedAt:   sql.NullInt64{Valid: true, Int64: now},
			}); err != nil {
				return wrapRegionWriteErr(err)
			}
		} else {
			policyID = uid.New(uid.AutoscalingPolicyPrefix)
			if err := db.Query.InsertHorizontalAutoscalingPolicy(ctx, tx, db.InsertHorizontalAutoscalingPolicyParams{
				ID:           policyID,
				WorkspaceID:  workspaceID,
				ReplicasMin:  d.min,
				ReplicasMax:  d.max,
				CpuThreshold: sql.NullInt16{Valid: true, Int16: cpuThreshold},
				CreatedAt:    now,
			}); err != nil {
				return wrapRegionWriteErr(err)
			}
		}

		if err := db.Query.UpsertAppRegionalSettings(ctx, tx, db.UpsertAppRegionalSettingsParams{
			WorkspaceID:                   workspaceID,
			AppID:                         appID,
			EnvironmentID:                 environmentID,
			RegionID:                      d.regionID,
			Replicas:                      d.max,
			HorizontalAutoscalingPolicyID: sql.NullString{Valid: true, String: policyID},
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return wrapRegionWriteErr(err)
		}
	}

	for _, row := range current {
		if _, ok := desiredByRegion[row.RegionID]; ok {
			continue
		}

		if err := db.Query.DeleteAppRegionalSettingByAppEnvRegion(ctx, tx, db.DeleteAppRegionalSettingByAppEnvRegionParams{
			AppID:         appID,
			EnvironmentID: environmentID,
			RegionID:      row.RegionID,
		}); err != nil {
			return wrapRegionWriteErr(err)
		}

		if row.HorizontalAutoscalingPolicyID.Valid {
			if err := db.Query.DeleteHorizontalAutoscalingPolicy(ctx, tx, db.DeleteHorizontalAutoscalingPolicyParams{
				ID:          row.HorizontalAutoscalingPolicyID.String,
				WorkspaceID: workspaceID,
			}); err != nil {
				return wrapRegionWriteErr(err)
			}
		}
	}

	return nil
}

func wrapRegionWriteErr(err error) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("unable to update regional settings"),
		fault.Public("We're unable to update the environment settings."),
	)
}

// buildHealthcheck maps the request healthcheck to the stored type, applying the
// same field defaults the dashboard zod schema uses (OpenAPI ints carry no default).
func buildHealthcheck(hc openapi.Healthcheck) *dbtype.Healthcheck {
	intervalSeconds := 10
	if hc.IntervalSeconds != nil {
		intervalSeconds = *hc.IntervalSeconds
	}
	timeoutSeconds := 5
	if hc.TimeoutSeconds != nil {
		timeoutSeconds = *hc.TimeoutSeconds
	}
	failureThreshold := 3
	if hc.FailureThreshold != nil {
		failureThreshold = *hc.FailureThreshold
	}
	initialDelaySeconds := 0
	if hc.InitialDelaySeconds != nil {
		initialDelaySeconds = *hc.InitialDelaySeconds
	}

	return &dbtype.Healthcheck{
		Method:              string(hc.Method),
		Path:                hc.Path,
		IntervalSeconds:     intervalSeconds,
		TimeoutSeconds:      timeoutSeconds,
		FailureThreshold:    failureThreshold,
		InitialDelaySeconds: initialDelaySeconds,
	}
}
