package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2WorkspaceGetUsageRequestBody
	Response = openapi.V2WorkspaceGetUsageResponseBody
)

type Handler struct {
	DB         db.Database
	ClickHouse clickhouse.ClickHouse
	Clock      clock.Clock
}

// resourceNames maps ids to their names rows. A missing id means the resource
// was deleted. ParentID is the project id of an app and the app id of an
// environment
type resourceNames struct {
	projects     map[string]db.ListResourceNamesByIDsRow
	apps         map[string]db.ListResourceNamesByIDsRow
	environments map[string]db.ListResourceNamesByIDsRow
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/workspace.getUsage"
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

	workspaceID := principal.AuthorizedWorkspaceID
	err = principal.Authorize(rbac.Or(
		rbac.U(urn.New().Workspace(workspaceID).Usage(), permissions.Read),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Workspace,
			ResourceID:   "*",
			Action:       rbac.ReadUsage,
		}),
	))
	if err != nil {
		return err
	}

	now := h.Clock.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	start, end := monthStart, now
	if ptr.SafeDeref(req.Period, openapi.UsagePeriodCurrent) == openapi.UsagePeriodPrevious {
		start, end = monthStart.AddDate(0, -1, 0), monthStart
	}

	year, month := start.Year(), int(start.Month())

	var (
		verifications        int64
		ratelimits           int64
		computeByEnvironment []clickhouse.ComputeUsageByEnvironment
		keysByApp            []clickhouse.ActiveKeysByApp
	)
	err = conc.All(ctx,
		func(ctx context.Context) (err error) {
			verifications, err = h.ClickHouse.GetBillableVerifications(ctx, workspaceID, year, month)
			return err
		},
		func(ctx context.Context) (err error) {
			ratelimits, err = h.ClickHouse.GetBillableRatelimits(ctx, workspaceID, year, month)
			return err
		},
		func(ctx context.Context) (err error) {
			computeByEnvironment, err = h.ClickHouse.GetComputeUsageByEnvironment(ctx, workspaceID, start, end)
			return err
		},
		func(ctx context.Context) (err error) {
			keysByApp, err = h.ClickHouse.GetActiveKeysByApp(ctx, workspaceID, year, month)
			return err
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("clickhouse error"),
			fault.Public("Failed to read the workspace's usage."),
		)
	}

	names, err := h.findResourceNames(ctx, workspaceID, computeByEnvironment, keysByApp)
	if err != nil {
		return err
	}

	totals := openapi.V2WorkspaceGetUsageTotals{
		Api: openapi.V2WorkspaceGetUsageApi{
			Verifications: verifications,
			Ratelimits:    ratelimits,
		},
		Compute: openapi.V2WorkspaceGetUsageCompute{
			CpuSeconds:      0,
			MemoryGiBHours:  0,
			StorageGiBHours: 0,
			EgressGiB:       0,
		},
		Gateway: openapi.V2WorkspaceGetUsageGateway{ActiveKeys: 0},
	}
	breakdowns := openapi.V2WorkspaceGetUsageBreakdowns{
		ByEnvironment: make([]openapi.V2WorkspaceGetUsageByEnvironmentRow, 0, len(computeByEnvironment)),
		ByApp:         make([]openapi.V2WorkspaceGetUsageByAppRow, 0, len(keysByApp)),
	}

	for _, row := range computeByEnvironment {
		// A container that started before the collector recorded app ids keeps an
		// empty app id. Its environment still has the app id
		appID := row.AppID
		if appID == "" {
			appID = names.environments[row.EnvironmentID].ParentID
		}

		totals.Compute.CpuSeconds += row.CPUSeconds
		totals.Compute.MemoryGiBHours += row.MemoryGiBHours
		totals.Compute.StorageGiBHours += row.DiskGiBHours
		totals.Compute.EgressGiB += row.EgressGiB

		environmentRow := openapi.V2WorkspaceGetUsageByEnvironmentRow{
			Project: openapi.V2WorkspaceGetUsageResource{
				Id:   row.ProjectID,
				Name: nameOf(names.projects, row.ProjectID),
			},
			App: nil,
			Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{
				Id:   row.EnvironmentID,
				Slug: nameOf(names.environments, row.EnvironmentID),
			},
			Compute: openapi.V2WorkspaceGetUsageCompute{
				CpuSeconds:      row.CPUSeconds,
				MemoryGiBHours:  row.MemoryGiBHours,
				StorageGiBHours: row.DiskGiBHours,
				EgressGiB:       row.EgressGiB,
			},
		}
		if appID != "" {
			environmentRow.App = &openapi.V2WorkspaceGetUsageResource{
				Id:   appID,
				Name: nameOf(names.apps, appID),
			}
		}
		breakdowns.ByEnvironment = append(breakdowns.ByEnvironment, environmentRow)
	}

	for _, row := range keysByApp {
		totals.Gateway.ActiveKeys += row.ActiveKeys

		appRow := openapi.V2WorkspaceGetUsageByAppRow{
			Project: nil,
			App:     nil,
			Gateway: openapi.V2WorkspaceGetUsageGateway{ActiveKeys: row.ActiveKeys},
		}
		if row.AppID != "" {
			appRow.App = &openapi.V2WorkspaceGetUsageResource{Id: row.AppID, Name: nil}
			if app, ok := names.apps[row.AppID]; ok {
				appRow.App.Name = &app.Name
				appRow.Project = &openapi.V2WorkspaceGetUsageResource{
					Id:   app.ParentID,
					Name: nameOf(names.projects, app.ParentID),
				}
			}
		}
		breakdowns.ByApp = append(breakdowns.ByApp, appRow)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2WorkspaceGetUsageResponseData{
			Period: openapi.V2WorkspaceGetUsagePeriod{
				Start: start.UnixMilli(),
				End:   end.UnixMilli(),
			},
			Totals:     totals,
			Breakdowns: breakdowns,
		},
	})
}

// findResourceNames reads the names of the projects, apps, and environments in
// the usage rows in one query
func (h *Handler) findResourceNames(
	ctx context.Context,
	workspaceID string,
	computeByEnvironment []clickhouse.ComputeUsageByEnvironment,
	keysByApp []clickhouse.ActiveKeysByApp,
) (resourceNames, error) {
	names := resourceNames{
		projects:     map[string]db.ListResourceNamesByIDsRow{},
		apps:         map[string]db.ListResourceNamesByIDsRow{},
		environments: map[string]db.ListResourceNamesByIDsRow{},
	}
	if len(computeByEnvironment) == 0 && len(keysByApp) == 0 {
		return names, nil
	}

	params := db.ListResourceNamesByIDsParams{
		WorkspaceID:         workspaceID,
		ProjectIds:          make([]string, 0, len(computeByEnvironment)),
		ProjectOfAppIds:     make([]string, 0, len(keysByApp)),
		AppIds:              make([]string, 0, len(computeByEnvironment)+len(keysByApp)),
		AppOfEnvironmentIds: nil,
		EnvironmentIds:      make([]string, 0, len(computeByEnvironment)),
	}
	for _, row := range computeByEnvironment {
		params.ProjectIds = append(params.ProjectIds, row.ProjectID)
		params.EnvironmentIds = append(params.EnvironmentIds, row.EnvironmentID)
		if row.AppID == "" {
			params.AppOfEnvironmentIds = append(params.AppOfEnvironmentIds, row.EnvironmentID)
			continue
		}
		params.AppIds = append(params.AppIds, row.AppID)
	}
	for _, row := range keysByApp {
		params.ProjectOfAppIds = append(params.ProjectOfAppIds, row.AppID)
		params.AppIds = append(params.AppIds, row.AppID)
	}

	rows, err := db.Query.ListResourceNamesByIDs(ctx, h.DB.RO(), params)
	if err != nil {
		return resourceNames{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to read the names of the workspace's projects, apps, and environments."),
		)
	}
	for _, row := range rows {
		switch row.Kind {
		case "project":
			names.projects[row.ID] = row
		case "app":
			names.apps[row.ID] = row
		case "environment":
			names.environments[row.ID] = row
		default:
			return resourceNames{}, fault.New("unknown resource kind",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal(fmt.Sprintf("ListResourceNamesByIDs returned kind %q", row.Kind)),
				fault.Public("An unexpected error occurred while processing your request."),
			)
		}
	}
	return names, nil
}

func nameOf(rows map[string]db.ListResourceNamesByIDsRow, id string) *string {
	row, ok := rows[id]
	if !ok {
		return nil
	}
	return &row.Name
}
