package gdprerasure

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/workos"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// ClickHouse is the slice of the ClickHouse client the erasure needs. A noop
// client finds no rows.
type ClickHouse interface {
	Exec(ctx context.Context, sql string, args ...any) error
	QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error)
}

// Config holds the workflow's dependencies.
type Config struct {
	// DB is the control plane database. Must not be nil.
	DB db.Database

	// ClickHouse holds the analytics rows. Must not be nil; pass the noop
	// client when ClickHouse is not configured.
	ClickHouse ClickHouse

	// WorkOS deletes users and organizations. Must not be nil; pass
	// workos.NewNoop() when WorkOS is not configured.
	WorkOS workos.Eraser
}

// Workflow implements GdprErasureWorkflow.
type Workflow struct {
	hydrav1.UnimplementedGdprErasureWorkflowServer
	db         db.Database
	clickhouse ClickHouse
	workos     workos.Eraser
}

var _ hydrav1.GdprErasureWorkflowServer = (*Workflow)(nil)

// New constructs a Workflow.
func New(cfg Config) (*Workflow, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.ClickHouse, "ClickHouse must not be nil; use clickhouse.NewNoop()"),
		assert.NotNil(cfg.WorkOS, "WorkOS must not be nil; use workos.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Workflow{
		UnimplementedGdprErasureWorkflowServer: hydrav1.UnimplementedGdprErasureWorkflowServer{},
		db:                                     cfg.DB,
		clickhouse:                             cfg.ClickHouse,
		workos:                                 cfg.WorkOS,
	}, nil
}

// Run erases the request's subject by running its registry steps in order and
// reports each step's affected rows.
func (w *Workflow) Run(ctx restate.WorkflowContext, req *hydrav1.GdprErasureRequest) (*hydrav1.GdprErasureResponse, error) {
	s, steps, err := w.plan(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &hydrav1.GdprErasureResponse{
		Steps:            make([]*hydrav1.GdprErasureStepResult, 0, len(steps)),
		StripeCustomerId: s.stripeCustomerID,
	}
	for _, st := range steps {
		rows, err := st.execute(ctx, w, *s)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", st.name(), err)
		}
		res.Steps = append(res.Steps, &hydrav1.GdprErasureStepResult{Name: st.name(), RowsAffected: rows})
	}

	logger.Info("gdpr erasure complete", "request", restate.Key(ctx), "steps", len(res.Steps))
	return res, nil
}

// plan parses the request into the subject's bound values and its registry
// steps. Anything that must stop the erasure is checked here, before any step
// changes data.
func (w *Workflow) plan(ctx restate.WorkflowContext, req *hydrav1.GdprErasureRequest) (*subject, []step, error) {
	switch target := req.GetSubject().(type) {
	case *hydrav1.GdprErasureRequest_User:
		userID := target.User.GetWorkosUserId()
		if userID == "" {
			return nil, nil, restate.TerminalErrorf("user.workos_user_id is required")
		}
		return &subject{
			workspaceID:      "",
			externalID:       "",
			userID:           userID,
			identityIDs:      nil,
			orgID:            "",
			stripeCustomerID: "",
		}, userSteps, nil

	case *hydrav1.GdprErasureRequest_Workspace:
		workspaceID := target.Workspace.GetWorkspaceId()
		if workspaceID == "" {
			return nil, nil, restate.TerminalErrorf("workspace.workspace_id is required")
		}
		s, err := w.guardWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, nil, err
		}
		return s, workspaceSteps, nil

	case *hydrav1.GdprErasureRequest_EndUser:
		workspaceID := target.EndUser.GetWorkspaceId()
		externalID := target.EndUser.GetExternalId()
		if workspaceID == "" || externalID == "" {
			return nil, nil, restate.TerminalErrorf("end_user.workspace_id and end_user.external_id are required")
		}
		identityIDs, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
			return w.db.ListIdentityIDsByExternalID(rc, db.ListIdentityIDsByExternalIDParams{
				WorkspaceID: workspaceID,
				ExternalID:  externalID,
			})
		}, restate.WithName("resolve identities"))
		if err != nil {
			return nil, nil, fmt.Errorf("resolve identities: %w", err)
		}
		return &subject{
			workspaceID:      workspaceID,
			externalID:       externalID,
			userID:           "",
			identityIDs:      identityIDs,
			orgID:            "",
			stripeCustomerID: "",
		}, endUserSteps, nil

	default:
		return nil, nil, restate.TerminalErrorf("request has no subject")
	}
}

// workspaceRecord is what the workspace guard reads before any step runs. It
// is journaled, so its fields are exported.
type workspaceRecord struct {
	DeleteProtection bool   `json:"delete_protection"`
	OrgID            string `json:"org_id"`
}

// guardWorkspace refuses to erase a delete-protected workspace and reads the
// external ids the later steps delete. A missing workspaces row is not an
// error: an earlier run may have deleted it, and every step still converges.
func (w *Workflow) guardWorkspace(ctx restate.WorkflowContext, workspaceID string) (*subject, error) {
	record, err := restate.Run(ctx, func(rc restate.RunContext) (workspaceRecord, error) {
		workspace, err := w.db.FindWorkspaceByID(rc, workspaceID)
		if db.IsNotFound(err) {
			return workspaceRecord{DeleteProtection: false, OrgID: ""}, nil
		}
		if err != nil {
			return workspaceRecord{DeleteProtection: false, OrgID: ""}, err
		}
		return workspaceRecord{
			DeleteProtection: workspace.DeleteProtection.Valid && workspace.DeleteProtection.Bool,
			OrgID:            workspace.OrgID,
		}, nil
	}, restate.WithName("read workspace"))
	if err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}
	if record.DeleteProtection {
		return nil, restate.TerminalErrorf("workspace %s has delete protection; disable it before erasing", workspaceID)
	}
	return &subject{
		workspaceID:      workspaceID,
		externalID:       "",
		userID:           "",
		identityIDs:      nil,
		orgID:            record.OrgID,
		stripeCustomerID: "",
	}, nil
}
