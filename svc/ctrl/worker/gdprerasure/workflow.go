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

	// Stripe deletes a workspace's Stripe customer. Nil when no Stripe key is
	// configured, and then erasing a workspace that has a Stripe customer
	// fails before anything is deleted.
	Stripe StripeCustomers

	// Resend deletes a dashboard user's Resend contact. Nil when no Resend key
	// is configured, and then every user erasure fails before anything is
	// deleted.
	Resend ResendContacts
}

// Workflow implements GdprErasureWorkflow.
type Workflow struct {
	hydrav1.UnimplementedGdprErasureWorkflowServer
	db         db.Database
	clickhouse ClickHouse
	workos     workos.Eraser
	stripe     StripeCustomers
	resend     ResendContacts
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
		stripe:                                 cfg.Stripe,
		resend:                                 cfg.Resend,
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
		if w.resend == nil {
			return nil, nil, restate.TerminalErrorf("no resend key is configured, so user %s would stay a resend contact; refusing to erase", userID)
		}
		email, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
			return w.workos.UserEmail(rc, userID)
		}, restate.WithName("read workos user email"))
		if err != nil {
			return nil, nil, fmt.Errorf("read workos user email: %w", err)
		}
		return &subject{
			workspaceID:      "",
			externalID:       "",
			userID:           userID,
			email:            email,
			identityIDs:      nil,
			orgID:            "",
			stripeCustomerID: "",
		}, userSteps, nil

	case *hydrav1.GdprErasureRequest_Workspace:
		workspaceID := target.Workspace.GetWorkspaceId()
		workspaceSlug := target.Workspace.GetWorkspaceSlug()
		if workspaceID == "" || workspaceSlug == "" {
			return nil, nil, restate.TerminalErrorf("workspace.workspace_id and workspace.workspace_slug are required")
		}
		s, err := w.guardWorkspace(ctx, workspaceID, workspaceSlug)
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
			email:            "",
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
	Found            bool   `json:"found"`
	Slug             string `json:"slug"`
	OrgID            string `json:"org_id"`
	StripeCustomerID string `json:"stripe_customer_id"`

	// StripeCustomerShares counts the other workspaces billed to the same
	// Stripe customer.
	StripeCustomerShares int64 `json:"stripe_customer_shares"`
}

// guardWorkspace refuses an erasure it cannot finish or must not start, and
// reads the external ids the later steps delete. A missing workspaces row is
// not an error: an earlier run may have deleted it, and every step still
// converges.
func (w *Workflow) guardWorkspace(ctx restate.WorkflowContext, workspaceID, workspaceSlug string) (*subject, error) {
	record, err := restate.Run(ctx, func(rc restate.RunContext) (workspaceRecord, error) {
		return w.readWorkspace(rc, workspaceID)
	}, restate.WithName("read workspace"))
	if err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}
	if record.Found && record.Slug != workspaceSlug {
		return nil, restate.TerminalErrorf("workspace %s has slug %q, not %q; refusing to erase it", workspaceID, record.Slug, workspaceSlug)
	}
	if record.StripeCustomerID != "" && record.StripeCustomerShares > 0 {
		return nil, restate.TerminalErrorf("stripe customer %s also bills %d other workspaces; resolve that before erasing workspace %s",
			record.StripeCustomerID, record.StripeCustomerShares, workspaceID)
	}
	if record.StripeCustomerID != "" && w.stripe == nil {
		return nil, restate.TerminalErrorf("workspace %s has stripe customer %s but no stripe key is configured", workspaceID, record.StripeCustomerID)
	}
	return &subject{
		workspaceID:      workspaceID,
		externalID:       "",
		userID:           "",
		email:            "",
		identityIDs:      nil,
		orgID:            record.OrgID,
		stripeCustomerID: record.StripeCustomerID,
	}, nil
}

func (w *Workflow) readWorkspace(ctx context.Context, workspaceID string) (workspaceRecord, error) {
	var record workspaceRecord

	workspace, err := w.db.FindWorkspaceByID(ctx, workspaceID)
	if err != nil && !db.IsNotFound(err) {
		return record, err
	}
	if err == nil {
		record.Found = true
		record.Slug = workspace.Slug
		record.OrgID = workspace.OrgID
	}

	billing, err := w.db.FindWorkspaceBillingByWorkspaceID(ctx, workspaceID)
	if err != nil && !db.IsNotFound(err) {
		return record, err
	}
	if err != nil || !billing.StripeCustomerID.Valid || billing.StripeCustomerID.String == "" {
		return record, nil
	}
	record.StripeCustomerID = billing.StripeCustomerID.String

	record.StripeCustomerShares, err = w.db.CountOtherWorkspaceBillingsByStripeCustomerID(ctx, db.CountOtherWorkspaceBillingsByStripeCustomerIDParams{
		StripeCustomerID: billing.StripeCustomerID,
		WorkspaceID:      workspaceID,
	})
	return record, err
}
