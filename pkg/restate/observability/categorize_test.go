package observability

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantOutcome  string
		wantCategory string
	}{
		{
			name:         "nil",
			err:          nil,
			wantOutcome:  OutcomeSuccess,
			wantCategory: CategoryNone,
		},
		{
			name:         "context canceled",
			err:          context.Canceled,
			wantOutcome:  OutcomeCancelled,
			wantCategory: CategoryCancelled,
		},
		{
			name:         "context deadline exceeded",
			err:          context.DeadlineExceeded,
			wantOutcome:  OutcomeCancelled,
			wantCategory: CategoryCancelled,
		},
		{
			name:         "wrapped context canceled",
			err:          fmt.Errorf("upstream: %w", context.Canceled),
			wantOutcome:  OutcomeCancelled,
			wantCategory: CategoryCancelled,
		},
		{
			name:         "untagged error defaults to infra",
			err:          stderrors.New("boom"),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryInfra,
		},
		{
			name:         "workflow app code",
			err:          fault.Wrap(stderrors.New("dockerfile parse"), fault.Code(codes.Workflow.App.BuildBroken.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryApp,
		},
		{
			name:         "workflow provider code (depot)",
			err:          fault.Wrap(stderrors.New("build failed"), fault.Code(codes.Workflow.Provider.DepotBuildFailed.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryProvider,
		},
		{
			name:         "workflow provider code (acme)",
			err:          fault.Wrap(stderrors.New("rate limited"), fault.Code(codes.Workflow.Provider.AcmeRateLimited.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryProvider,
		},
		{
			name:         "workflow infra code",
			err:          fault.Wrap(stderrors.New("krane timeout"), fault.Code(codes.Workflow.Infra.KraneTimeout.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryInfra,
		},
		{
			name:         "user system code",
			err:          fault.Wrap(stderrors.New("bad input"), fault.Code(codes.User.BadRequest.MissingRequiredHeader.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryUser,
		},
		{
			name:         "unkey system code falls back to infra",
			err:          fault.Wrap(stderrors.New("data error"), fault.Code(codes.Data.Workspace.NotFound.URN())),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryInfra,
		},
		{
			name: "fmt-wrapped fault still categorizes",
			err: fmt.Errorf("outer: %w",
				fault.Wrap(stderrors.New("inner"), fault.Code(codes.Workflow.Provider.DepotMachineUnavailable.URN())),
			),
			wantOutcome:  OutcomeFailed,
			wantCategory: CategoryProvider,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotOutcome, gotCategory := Classify(tc.err)
			if gotOutcome != tc.wantOutcome {
				t.Errorf("outcome: got %q, want %q", gotOutcome, tc.wantOutcome)
			}
			if gotCategory != tc.wantCategory {
				t.Errorf("category: got %q, want %q", gotCategory, tc.wantCategory)
			}
		})
	}
}
