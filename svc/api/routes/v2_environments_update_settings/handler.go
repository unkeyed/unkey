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

// Per-instance resource ceilings applied when a workspace has no quota row.
// These mirror the dashboard fallbacks and the quota column defaults.
const (
	defaultMaxCPUMillicores = 2000
	defaultMaxMemoryMib     = 4096
	defaultMaxStorageMib    = 10240
)

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

	environment := env

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

	hasBuild := req.Dockerfile.IsSpecified() || req.RootDirectory != nil ||
		req.WatchPaths != nil || req.AutoDeploy != nil
	hasRuntime := req.Port != nil || req.CpuMillicores != nil || req.MemoryMib != nil ||
		req.StorageMib != nil || req.Command != nil || req.Healthcheck.IsSpecified() ||
		req.ShutdownSignal != nil || req.UpstreamProtocol != nil || req.OpenapiSpecPath.IsSpecified()

	if !hasBuild && !hasRuntime && req.Regions == nil {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.EmptyResponse{},
		})
	}

	if hasRuntime {
		if err := h.validateResourceQuota(ctx, principal.WorkspaceID, req); err != nil {
			return err
		}
	}

	var desired []resolvedRegion
	if req.Regions != nil {
		desired, err = h.resolveRegions(ctx, *req.Regions)
		if err != nil {
			return err
		}
	}

	now := time.Now().UnixMilli()
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Lock the environment row before region reconciliation, which reads then
		// replaces the regional set. Build/runtime UPDATEs skip the lock.
		if req.Regions != nil {
			if _, err := db.Query.LockEnvironmentForUpdate(ctx, tx, environment.ID); err != nil {
				if db.IsNotFound(err) {
					// Deleted between the read above and acquiring the lock.
					return fault.New(
						"environment not found",
						fault.Code(codes.Data.Environment.NotFound.URN()),
						fault.Internal("environment deleted before lock"),
						fault.Public("The requested environment does not exist."),
					)
				}
				return fault.Wrap(
					err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to lock environment"),
					fault.Public("We're unable to update the environment settings."),
				)
			}
		}

		if hasBuild {
			if err := h.applyBuildSettings(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, req, now); err != nil {
				return err
			}
		}

		if hasRuntime {
			if err := h.applyRuntimeSettings(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, req, now); err != nil {
				return err
			}
		}

		if req.Regions != nil {
			if err := h.applyRegions(ctx, tx, principal.WorkspaceID, environment.AppID, environment.ID, desired, now); err != nil {
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

func (h *Handler) applyBuildSettings(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, req Request, now int64) error {
	params := db.UpdateAppBuildSettingsParams{
		WorkspaceID:            workspaceID,
		AppID:                  appID,
		EnvironmentID:          environmentID,
		UpdatedAt:              sql.NullInt64{Valid: true, Int64: now},
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
			params.Dockerfile = sql.NullString{Valid: true, String: req.Dockerfile.MustGet()}
		}
	}
	if req.RootDirectory != nil {
		params.DockerContextSpecified = 1
		params.DockerContext = *req.RootDirectory
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

func (h *Handler) applyRuntimeSettings(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, req Request, now int64) error {
	params := db.UpdateAppRuntimeSettingsParams{
		WorkspaceID:               workspaceID,
		AppID:                     appID,
		EnvironmentID:             environmentID,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
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
			params.Healthcheck = dbtype.NullHealthcheck{Valid: true, Healthcheck: buildHealthcheck(req.Healthcheck.MustGet())}
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
			params.OpenapiSpecPath = sql.NullString{Valid: true, String: req.OpenapiSpecPath.MustGet()}
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

// validateResourceQuota rejects cpu/memory/storage requests that exceed the
// workspace's per-instance quota, mirroring the dashboard.
func (h *Handler) validateResourceQuota(ctx context.Context, workspaceID string, req Request) error {
	maxCPU := uint32(defaultMaxCPUMillicores)
	maxMemory := uint32(defaultMaxMemoryMib)
	maxStorage := uint32(defaultMaxStorageMib)

	quota, err := db.Query.FindQuotaByWorkspaceID(ctx, h.DB.RO(), workspaceID)
	if err != nil {
		if !db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to validate resource limits."),
			)
		}
	} else {
		maxCPU = quota.MaxCpuMillicoresPerInstance
		maxMemory = quota.MaxMemoryMibPerInstance
		maxStorage = quota.MaxStorageMibPerInstance
	}

	if req.CpuMillicores != nil && *req.CpuMillicores > int(maxCPU) {
		return quotaExceeded(fmt.Sprintf("CPU per instance cannot exceed %d millicores. Contact support@unkey.com to increase it.", maxCPU))
	}
	if req.MemoryMib != nil && *req.MemoryMib > int(maxMemory) {
		return quotaExceeded(fmt.Sprintf("Memory per instance cannot exceed %d MiB. Contact support@unkey.com to increase it.", maxMemory))
	}
	if req.StorageMib != nil && *req.StorageMib > int(maxStorage) {
		return quotaExceeded(fmt.Sprintf("Storage per instance cannot exceed %d MiB. Contact support@unkey.com to increase it.", maxStorage))
	}
	return nil
}

func quotaExceeded(public string) error {
	return fault.New(
		"quota exceeded",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("resource request exceeds workspace per-instance quota"),
		fault.Public(public),
	)
}

// resolveRegions validates the desired regions against the regions table and
// replica bounds, returning the resolved region ids. The regions table is loaded
// once and matched in memory.
func (h *Handler) resolveRegions(ctx context.Context, regions []openapi.RegionSetting) ([]resolvedRegion, error) {
	all, err := db.Query.ListRegions(ctx, h.DB.RO())
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to resolve regions."),
		)
	}
	byKey := make(map[string]db.ListRegionsRow, len(all))
	for _, region := range all {
		byKey[region.Platform+"/"+region.Name] = region
	}

	seen := make(map[string]struct{}, len(regions))
	resolved := make([]resolvedRegion, 0, len(regions))

	for _, r := range regions {
		platform := "aws"
		if r.Platform != nil {
			platform = *r.Platform
		}
		key := platform + "/" + r.Name
		rmin, rmax := int32(r.Replicas.Min), int32(r.Replicas.Max)

		if _, dup := seen[key]; dup {
			return nil, invalidRegion(fmt.Sprintf("Region '%s' on platform '%s' is listed more than once.", r.Name, platform))
		}
		seen[key] = struct{}{}

		if rmin > rmax {
			return nil, invalidRegion(fmt.Sprintf("Region '%s' min replicas cannot exceed max replicas.", r.Name))
		}

		region, ok := byKey[key]
		if !ok {
			return nil, invalidRegion(fmt.Sprintf("Region '%s' on platform '%s' does not exist.", r.Name, platform))
		}

		// The deploy worker silently skips regions with can_schedule=false, so
		// reject them here rather than store a set that never schedules.
		if !region.CanSchedule {
			return nil, invalidRegion(fmt.Sprintf("Region '%s' on platform '%s' is not available for scheduling.", r.Name, platform))
		}

		// applyRegions writes one shared policy from desired[0], so per-region
		// bounds must match. Infra supports per-region; this is a product choice
		// mirroring the dashboard's single env-level replica range.
		if len(resolved) > 0 && (rmin != resolved[0].min || rmax != resolved[0].max) {
			return nil, invalidRegion("All regions must specify the same replica bounds; per-region autoscaling is not supported yet.")
		}

		resolved = append(resolved, resolvedRegion{
			regionID: region.ID,
			min:      rmin,
			max:      rmax,
		})
	}

	return resolved, nil
}

// applyRegions reconciles the desired region set: reuse or create the env's
// shared autoscaling policy, point every desired region at it with replicas =
// max, and delete rows for regions no longer desired. The policy is never deleted.
func (h *Handler) applyRegions(ctx context.Context, tx db.DBTX, workspaceID, appID, environmentID string, desired []resolvedRegion, now int64) error {
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

	sharedPolicyID := existingPolicyID(current)

	if len(desired) > 0 {
		minReplicas, maxReplicas := desired[0].min, desired[0].max

		sharedPolicyID, err = h.ensureSharedPolicy(ctx, tx, workspaceID, sharedPolicyID, minReplicas, maxReplicas, now)
		if err != nil {
			return wrapRegionWriteErr(err)
		}

		for _, d := range desired {
			if err := db.Query.UpsertAppRegionalSettings(ctx, tx, db.UpsertAppRegionalSettingsParams{
				WorkspaceID:                   workspaceID,
				AppID:                         appID,
				EnvironmentID:                 environmentID,
				RegionID:                      d.regionID,
				Replicas:                      maxReplicas,
				HorizontalAutoscalingPolicyID: sql.NullString{Valid: true, String: sharedPolicyID},
				CreatedAt:                     now,
				UpdatedAt:                     sql.NullInt64{Valid: true, Int64: now},
			}); err != nil {
				return wrapRegionWriteErr(err)
			}
		}
	}

	desiredByRegion := make(map[string]struct{}, len(desired))
	for _, d := range desired {
		desiredByRegion[d.regionID] = struct{}{}
	}

	// Remove rows for regions no longer desired. The shared policy is left in place.
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
	}

	return nil
}

// existingPolicyID returns the shared autoscaling policy id from the current
// regional rows, or "" if none reference one.
func existingPolicyID(current []db.ListAppRegionalSettingsByAppEnvRow) string {
	for _, row := range current {
		if row.HorizontalAutoscalingPolicyID.Valid {
			return row.HorizontalAutoscalingPolicyID.String
		}
	}
	return ""
}

// ensureSharedPolicy updates the policy at policyID to the given bounds, or
// creates one when policyID is empty, returning the id to point regional rows at.
func (h *Handler) ensureSharedPolicy(ctx context.Context, tx db.DBTX, workspaceID, policyID string, minReplicas, maxReplicas int32, now int64) (string, error) {
	if policyID != "" {
		return policyID, db.Query.UpdateHorizontalAutoscalingPolicy(ctx, tx, db.UpdateHorizontalAutoscalingPolicyParams{
			ID:          policyID,
			WorkspaceID: workspaceID,
			ReplicasMin: minReplicas,
			ReplicasMax: maxReplicas,
			UpdatedAt:   sql.NullInt64{Valid: true, Int64: now},
		})
	}

	policyID = uid.New(uid.AutoscalingPolicyPrefix)
	return policyID, db.Query.InsertHorizontalAutoscalingPolicy(ctx, tx, db.InsertHorizontalAutoscalingPolicyParams{
		ID:           policyID,
		WorkspaceID:  workspaceID,
		ReplicasMin:  minReplicas,
		ReplicasMax:  maxReplicas,
		CpuThreshold: sql.NullInt16{Valid: true, Int16: cpuThreshold},
		CreatedAt:    now,
	})
}

func wrapRegionWriteErr(err error) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("unable to update regional settings"),
		fault.Public("We're unable to update the environment settings."),
	)
}

func invalidRegion(public string) error {
	return fault.New(
		"invalid region",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("invalid regions input"),
		fault.Public(public),
	)
}

// buildHealthcheck maps the request healthcheck to the stored type, applying the
// dashboard's default field values.
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
