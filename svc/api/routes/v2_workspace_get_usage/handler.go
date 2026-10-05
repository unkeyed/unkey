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

	compute := openapi.V2WorkspaceGetUsageCompute{
		CpuSeconds:     0,
		MemoryGiBHours: 0,
		DiskGiBHours:   0,
		EgressGiB:      0,
		ActiveKeys:     0,
		Environments:   make([]openapi.V2WorkspaceGetUsageEnvironment, 0, len(computeByEnvironment)),
		Apps:           make([]openapi.V2WorkspaceGetUsageApp, 0, len(keysByApp)),
	}
	for _, row := range computeByEnvironment {
		// A container that started before the collector recorded app ids keeps an
		// empty app id. Its environment still has the app id
		appID := row.AppID
		if appID == "" {
			appID = names.environments[row.EnvironmentID].ParentID
		}
		compute.CpuSeconds += row.CPUSeconds
		compute.MemoryGiBHours += row.MemoryGiBHours
		compute.DiskGiBHours += row.DiskGiBHours
		compute.EgressGiB += row.EgressGiB
		compute.Environments = append(compute.Environments, openapi.V2WorkspaceGetUsageEnvironment{
			ProjectId:       row.ProjectID,
			ProjectName:     nameOf(names.projects, row.ProjectID),
			AppId:           appID,
			AppName:         nameOf(names.apps, appID),
			EnvironmentId:   row.EnvironmentID,
			EnvironmentSlug: nameOf(names.environments, row.EnvironmentID),
			CpuSeconds:      row.CPUSeconds,
			MemoryGiBHours:  row.MemoryGiBHours,
			DiskGiBHours:    row.DiskGiBHours,
			EgressGiB:       row.EgressGiB,
		})
	}
	for _, row := range keysByApp {
		compute.ActiveKeys += row.ActiveKeys
		appRow := openapi.V2WorkspaceGetUsageApp{
			AppId:       row.AppID,
			AppName:     nil,
			ProjectId:   nil,
			ProjectName: nil,
			ActiveKeys:  row.ActiveKeys,
		}
		if app, ok := names.apps[row.AppID]; ok {
			appRow.AppName = &app.Name
			appRow.ProjectId = &app.ParentID
			appRow.ProjectName = nameOf(names.projects, app.ParentID)
		}
		compute.Apps = append(compute.Apps, appRow)
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
			Api: openapi.V2WorkspaceGetUsageApi{
				Verifications: verifications,
				Ratelimits:    ratelimits,
			},
			Compute: compute,
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
