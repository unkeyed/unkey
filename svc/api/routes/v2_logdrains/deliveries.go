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

type Deliveries struct {
	DB         db.Database
	ClickHouse clickhouse.ClickHouse
	Clock      clock.Clock
}

func (h *Deliveries) Method() string { return http.MethodPost }
func (h *Deliveries) Path() string   { return "/v2/logdrains.getRecentDeliveries" }
func (h *Deliveries) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.LogdrainIdRequest](s)
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
	rows, err := h.ClickHouse.Conn().Query(ctx, `SELECT time, outcome, events, webhook_duration_ms, response_status, response_body, error
FROM default.logdrain_deliveries_raw_v1
PREWHERE workspace_id = ? AND drain_id = ? AND time >= ?
ORDER BY time DESC LIMIT 20`, principal.AuthorizedWorkspaceID, req.LogdrainId, h.Clock.Now().UnixMilli()-86_400_000)
	if err != nil {
		return err
	}
	defer rows.Close()
	response := openapi.LogdrainDeliveriesResponse{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: []openapi.LogdrainDelivery{}}
	for rows.Next() {
		var delivery openapi.LogdrainDelivery
		var status int32
		var outcome string
		if err := rows.Scan(&delivery.Time, &outcome, &delivery.Events, &delivery.DurationMs, &status, &delivery.ResponseBody, &delivery.Error); err != nil {
			return err
		}
		delivery.Outcome = openapi.LogdrainDeliveryOutcome(outcome)
		delivery.ResponseStatus = int(status)
		response.Data = append(response.Data, delivery)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return s.JSON(http.StatusOK, response)
}
