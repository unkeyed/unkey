package deploybilling

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// pushRetryDuration bounds the per-workspace push retries. Transient provider
// errors retry within this window; on exhaustion the Run returns a
// TerminalError, which the handler surfaces so the invocation fails rather
// than pausing and blocking an awaiting caller (the month-end close).
const pushRetryDuration = 2 * time.Minute

// PushHandler implements DeployBillingPushService: it pushes one workspace's
// month-to-date usage to the billing provider. It is fanned out to by the
// CronService orchestrators, one invocation per workspace, keyed by workspace
// id so a customer's pushes serialize and a broken workspace fails in
// isolation.
type PushHandler struct {
	pusher billingmeter.Pusher
}

// NewPushHandler constructs a PushHandler.
func NewPushHandler(pusher billingmeter.Pusher) (*PushHandler, error) {
	if err := assert.NotNil(pusher, "Pusher must not be nil; use billingmeter.NewNoop()"); err != nil {
		return nil, err
	}
	return &PushHandler{pusher: pusher}, nil
}

// PushWorkspaceUsage pushes the workspace's month-to-date meter values. The
// workspace id is the VO key; the customer, values, and event timestamp come
// in the request. A push failure is returned as an error so this workspace's
// invocation surfaces it; the next orchestrator tick re-sends the absolute
// total, so a failure self-heals once its cause clears.
func (h *PushHandler) PushWorkspaceUsage(
	ctx restate.ObjectContext,
	req *hydrav1.PushWorkspaceUsageRequest,
) (*hydrav1.PushWorkspaceUsageResponse, error) {
	workspaceID := restate.Key(ctx)

	pushReq := billingmeter.PushRequest{
		StripeCustomerID: req.GetStripeCustomerId(),
		Values: billingmeter.MeterValues{
			CPUSeconds:       req.GetCpuSeconds(),
			MemoryGiBSeconds: req.GetMemoryGibSeconds(),
			EgressGiB:        req.GetEgressGib(),
			DiskGiBSeconds:   req.GetDiskGibSeconds(),
			ActiveKeys:       req.GetActiveKeys(),
		},
		Timestamp: req.GetEventTimestamp(),
	}

	n, err := restate.Run(ctx, func(rc restate.RunContext) (int, error) {
		return h.pusher.Push(rc, pushReq)
	}, restate.WithName("push to provider"), restate.WithMaxRetryDuration(pushRetryDuration))
	if err != nil {
		logger.Error("deploy billing push failed",
			"workspace_id", workspaceID,
			"stripe_customer_id", req.GetStripeCustomerId(),
			"error", err,
		)
		return nil, err
	}

	// Shadow numbers, logged even when the noop pusher sent nothing.
	logger.Info("deploy billing push",
		"workspace_id", workspaceID,
		"stripe_customer_id", req.GetStripeCustomerId(),
		"cpu_seconds", req.GetCpuSeconds(),
		"memory_gib_seconds", req.GetMemoryGibSeconds(),
		"egress_gib", req.GetEgressGib(),
		"disk_gib_seconds", req.GetDiskGibSeconds(),
		"active_keys", req.GetActiveKeys(),
		"meters_pushed", n,
	)
	return &hydrav1.PushWorkspaceUsageResponse{MetersPushed: int32(n)}, nil
}
