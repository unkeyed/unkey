package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Response = openapi.V2WorkspaceGetLimitsResponseBody

const millicoresPerVCpu = 1000

type Handler struct {
	DB         db.Database
	ClickHouse clickhouse.ClickHouse
	Clock      clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/workspace.getLimits"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	workspaceID := principal.AuthorizedWorkspaceID
	err = principal.Authorize(rbac.Or(
		rbac.U(urn.New().Workspace(workspaceID).Limits(), permissions.Read),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Workspace,
			ResourceID:   "*",
			Action:       rbac.ReadLimits,
		}),
	))
	if err != nil {
		return err
	}

	limits, err := db.Query.FindLimitsWithUsageByWorkspaceID(ctx, h.DB.RO(), db.FindLimitsWithUsageByWorkspaceIDParams{
		WorkspaceID: workspaceID,
	})
	if db.IsNotFound(err) {
		return domaingate.LimitsNotConfigured(workspaceID)
	}
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to read the workspace's resource limits."),
		)
	}

	now := h.Clock.Now().UTC()
	verifications, err := h.ClickHouse.GetBillableVerifications(ctx, workspaceID, now.Year(), int(now.Month()))
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("clickhouse error"),
			fault.Public("Failed to read the workspace's billable operations."),
		)
	}
	ratelimits, err := h.ClickHouse.GetBillableRatelimits(ctx, workspaceID, now.Year(), int(now.Month()))
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("clickhouse error"),
			fault.Public("Failed to read the workspace's billable operations."),
		)
	}

	data := openapi.V2WorkspaceGetLimitsResponseData{
		Api: openapi.V2WorkspaceGetLimitsApi{
			BillableOperations: openapi.LimitMeter{
				Limit: int64(limits.ApiBillableOperationsCountMaxPerMonth),
				Used:  verifications + ratelimits,
			},
			RequestsPerMinute: nil,
		},
		Logs: openapi.V2WorkspaceGetLimitsLogs{
			RetentionDays:      int(limits.LogsRetentionDaysMax),
			AuditRetentionDays: int(limits.LogsAuditRetentionDaysMax),
			LogDrains: openapi.LimitMeter{
				Limit: int64(limits.LogdrainsMax),
				Used:  limits.LogdrainsCount,
			},
		},
		Compute: nil,
	}
	if limits.ApiRequestsCountMaxPerMinute.Valid {
		requestsPerMinute := int64(limits.ApiRequestsCountMaxPerMinute.Int32)
		data.Api.RequestsPerMinute = &requestsPerMinute
	}

	if deploygate.Entitled(limits.Plan, limits.PlanOverride) {
		data.Compute = &openapi.V2WorkspaceGetLimitsCompute{
			VCpus: openapi.V2WorkspaceGetLimitsVcpuMeter{
				Limit: float64(limits.CpuCoresMax),
				Used:  float64(limits.TotalCpuMillicores) / millicoresPerVCpu,
			},
			VCpusPerInstance: float64(limits.CpuCoresMaxPerInstance),
			MemoryMib: openapi.LimitMeter{
				Limit: int64(limits.MemoryMibMax),
				Used:  limits.TotalMemoryMib,
			},
			MemoryMibPerInstance: int(limits.MemoryMibMaxPerInstance),
			StorageMib: openapi.LimitMeter{
				Limit: int64(limits.StorageMibMax),
				Used:  limits.TotalStorageMib,
			},
			StorageMibPerInstance: int(limits.StorageMibMaxPerInstance),
			ConcurrentBuilds:      int(limits.BuildsConcurrentMax),
			ReplicasPerRegion:     int(limits.AutoscalingReplicasMax),
			CustomDomains: openapi.LimitMeter{
				Limit: int64(limits.CustomDomainsMax),
				Used:  limits.CustomDomainsCount,
			},
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
