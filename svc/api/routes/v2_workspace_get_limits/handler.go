package handler

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/nullable"
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

const millicoresPerCpuCore = 1000

// Plans store this custom domain limit to mean unlimited, the same value as
// CUSTOM_DOMAINS_UNLIMITED in the dashboard
const unlimitedCustomDomains = 1_000_000

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
	err = principal.Authorize(rbac.U(urn.New().Workspace(workspaceID).Limits(), permissions.Read))
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
		ApiBillableOperationsCountMaxPerMonth: meteredLimit(int64(limits.ApiBillableOperationsCountMaxPerMonth), float64(verifications+ratelimits)),
		ApiRequestsCountMaxPerMinute:          limit(int64(limits.ApiRequestsCountMaxPerMinute.Int32)),
		LogsRetentionDaysMax:                  limit(int64(limits.LogsRetentionDaysMax)),
		LogsAuditRetentionDaysMax:             limit(int64(limits.LogsAuditRetentionDaysMax)),
		LogdrainsMax:                          meteredLimit(int64(limits.LogdrainsMax), float64(limits.LogdrainsCount)),
		CpuCoresMax:                           nil,
		CpuCoresMaxPerInstance:                nil,
		MemoryMibMax:                          nil,
		MemoryMibMaxPerInstance:               nil,
		StorageMibMax:                         nil,
		StorageMibMaxPerInstance:              nil,
		BuildsConcurrentMax:                   nil,
		AutoscalingReplicasMax:                nil,
		CustomDomainsMax:                      nil,
	}
	if !limits.ApiRequestsCountMaxPerMinute.Valid {
		data.ApiRequestsCountMaxPerMinute.Limit = nullable.NewNullNullable[int64]()
	}

	if deploygate.Entitled(limits.Plan, limits.PlanOverride) {
		data.CpuCoresMax = new(meteredLimit(int64(limits.CpuCoresMax), float64(limits.TotalCpuMillicores)/millicoresPerCpuCore))
		data.CpuCoresMaxPerInstance = new(limit(int64(limits.CpuCoresMaxPerInstance)))
		data.MemoryMibMax = new(meteredLimit(int64(limits.MemoryMibMax), float64(limits.TotalMemoryMib)))
		data.MemoryMibMaxPerInstance = new(limit(int64(limits.MemoryMibMaxPerInstance)))
		data.StorageMibMax = new(meteredLimit(int64(limits.StorageMibMax), float64(limits.TotalStorageMib)))
		data.StorageMibMaxPerInstance = new(limit(int64(limits.StorageMibMaxPerInstance)))
		data.BuildsConcurrentMax = new(limit(int64(limits.BuildsConcurrentMax)))
		data.AutoscalingReplicasMax = new(limit(int64(limits.AutoscalingReplicasMax)))

		customDomains := meteredLimit(int64(limits.CustomDomainsMax), float64(limits.CustomDomainsCount))
		if limits.CustomDomainsMax >= unlimitedCustomDomains {
			customDomains.Limit = nullable.NewNullNullable[int64]()
		}
		data.CustomDomainsMax = &customDomains
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}

func limit(maximum int64) openapi.V2WorkspaceGetLimitsLimit {
	return openapi.V2WorkspaceGetLimitsLimit{
		Limit:   nullable.NewNullableWithValue(maximum),
		Current: nil,
	}
}

func meteredLimit(maximum int64, current float64) openapi.V2WorkspaceGetLimitsLimit {
	return openapi.V2WorkspaceGetLimitsLimit{
		Limit:   nullable.NewNullableWithValue(maximum),
		Current: &current,
	}
}
