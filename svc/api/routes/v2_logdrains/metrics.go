package logdrains

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Metrics struct {
	DB         db.Database
	ClickHouse clickhouse.ClickHouse
	Clock      clock.Clock
}

func (h *Metrics) Method() string { return http.MethodPost }
func (h *Metrics) Path() string   { return "/v2/logdrains.getMetrics" }
func (h *Metrics) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.LogdrainMetricsRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/" + req.LogdrainId}, permissions.Read)); err != nil {
		return err
	}
	if _, err := db.Query.FindLogdrain(ctx, h.DB.RO(), db.FindLogdrainParams{WorkspaceID: principal.AuthorizedWorkspaceID, ID: req.LogdrainId}); err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(err, fault.Code(codes.Data.Logdrain.NotFound.URN()), fault.Public("Log drain not found."))
		}
		return err
	}
	bucketMinutes := 1
	switch req.Hours {
	case openapi.N1:
	case openapi.N24:
		bucketMinutes = 30
	case openapi.N168:
		bucketMinutes = 240
	default:
		return invalid("Hours must be 1, 24, or 168.")
	}
	endMs := h.Clock.Now().UnixMilli()
	startMs := endMs - int64(req.Hours)*3_600_000
	stepMs := int64(bucketMinutes) * 60_000
	rows, err := h.ClickHouse.Conn().Query(ctx, `
SELECT intDiv(time, ?) * ? AS ts,
  toInt64(countIf(outcome = 'success')),
  toInt64(countIf(outcome IN ('error', 'transient_error'))),
  toInt64(countIf(outcome = 'permanent_error')),
  sumIf(events, outcome = 'success'), avg(webhook_duration_ms), maxIf(time, outcome = 'success')
FROM default.logdrain_deliveries_raw_v1
PREWHERE workspace_id = ? AND drain_id = ? AND time >= ? AND time <= ?
GROUP BY ts ORDER BY ts ASC WITH FILL FROM ? TO ? STEP ?`, stepMs, stepMs, principal.AuthorizedWorkspaceID, req.LogdrainId, startMs, endMs, startMs/stepMs*stepMs, endMs/stepMs*stepMs+stepMs, stepMs)
	if err != nil {
		return err
	}
	defer rows.Close()
	var response openapi.LogdrainMetricsResponse
	response.Meta.RequestId = s.RequestID()
	response.Data.BucketMinutes = bucketMinutes
	response.Data.Series = []openapi.LogdrainMetric{}
	for rows.Next() {
		var point openapi.LogdrainMetric
		if err := rows.Scan(&point.Ts, &point.SuccessCount, &point.TransientErrorCount, &point.PermanentErrorCount, &point.EventsDelivered, &point.AvgDurationMs, &point.LastSuccessMs); err != nil {
			return err
		}
		response.Data.Series = append(response.Data.Series, point)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return s.JSON(http.StatusOK, response)
}
