package billingmeter

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Meter event names. These are the stable contract between this package and the
// Stripe Billing Meters configured in infra: we post events against these names
// and Stripe's meters (formula "last") aggregate them and map them to metered
// prices. We never reference prices or meter IDs.
const (
	eventCPU    = "cpu_seconds"
	eventMemory = "memory_gib_seconds"
	eventEgress = "egress_public_gib"
	eventDisk   = "disk_gib_seconds"
)

// payloadKeyCustomer and payloadKeyValue are the meter's
// customer_mapping / value_settings payload keys.
const (
	payloadKeyCustomer = "stripe_customer_id"
	payloadKeyValue    = "value"
)

// stripePusher reports usage via Stripe Billing Meter events.
type stripePusher struct {
	client *stripe.Client
}

var _ Pusher = (*stripePusher)(nil)

// sdkLogger routes stripe-go's internal request logging (retries, request
// errors) through our structured logger instead of the SDK's raw stderr
// printer. The SDK logs with printf formatting, so the formatted string
// becomes the message and the origin is tagged for filtering.
type sdkLogger struct{}

var _ stripe.LeveledLoggerInterface = sdkLogger{}

func (sdkLogger) Debugf(format string, v ...any) {
	logger.Debug(fmt.Sprintf(format, v...), "source", "stripe-go")
}

func (sdkLogger) Infof(format string, v ...any) {
	logger.Info(fmt.Sprintf(format, v...), "source", "stripe-go")
}

func (sdkLogger) Warnf(format string, v ...any) {
	logger.Warn(fmt.Sprintf(format, v...), "source", "stripe-go")
}

func (sdkLogger) Errorf(format string, v ...any) {
	logger.Error(fmt.Sprintf(format, v...), "source", "stripe-go")
}

// NewStripe builds a Stripe-backed Pusher from a secret key.
func NewStripe(secretKey string) Pusher {
	return &stripePusher{client: stripe.NewClient(secretKey, stripe.WithBackends(stripe.NewBackendsWithConfig(&stripe.BackendConfig{
		//nolint:exhaustruct // defaults are fine for everything but the logger
		LeveledLogger: sdkLogger{},
	})))}
}

// Push posts one meter event per non-zero meter, carrying the workspace's
// month-to-date total as the meter value.
//
// No idempotency identifier is set: the meters aggregate with formula "last",
// so the billed quantity is always the value of the most recent event, never a
// sum. Retries, replays, and manual re-triggers therefore converge on the
// latest month-to-date total by construction — sending the same or a newer
// value again is harmless. (A stable identifier would be worse here: Stripe
// rejects a duplicate identifier with a hard 400, turning a harmless re-run
// into a failure.)
func (p *stripePusher) Push(ctx context.Context, req PushRequest) (int, error) {
	meters := []struct {
		name  string
		value float64
	}{
		{eventCPU, req.Values.CPUSeconds},
		{eventMemory, req.Values.MemoryGiBSeconds},
		{eventEgress, req.Values.EgressGiB},
		{eventDisk, req.Values.DiskGiBSeconds},
	}

	pushed := 0
	for _, m := range meters {
		if m.value <= 0 {
			continue
		}
		_, err := p.client.V1BillingMeterEvents.Create(ctx, &stripe.BillingMeterEventCreateParams{
			EventName: stripe.String(m.name),
			Timestamp: stripe.Int64(req.Timestamp),
			Payload: map[string]string{
				payloadKeyCustomer: req.StripeCustomerID,
				payloadKeyValue:    formatMeterValue(m.value),
			},
		})
		if err != nil {
			return pushed, fault.Wrap(err, fault.Internal("failed to push stripe meter event"))
		}
		pushed++
	}
	return pushed, nil
}

// Stripe enforces two separate limits on a meter event payload value, and a
// value must satisfy both: at most stripeMaxValueDigits total digits (counting
// both sides of the point) and at most stripeMaxValueDecimals decimal places.
// Full float64 precision (~15-17 significant digits) overruns both.
const (
	stripeMaxValueDigits   = 15
	stripeMaxValueDecimals = 12
)

// formatMeterValue renders a meter value as a plain decimal string that fits
// both Stripe limits. The decimals kept are whatever the integer part leaves
// within the 15-digit budget, but never more than the 12-decimal cap: a small
// value like 0.000006944 is bounded by the 12-decimal cap, while a large
// month-to-date total like 1781907 GiB-seconds is bounded by the 15-digit
// budget (7 integer digits leaves 8). A fixed 12-decimal count fails the large
// case (7 + 12 = 19 > 15); dropping the 12-decimal cap fails the small case
// (2 + 13 decimals > 12). Trailing zeros (and a bare dot) are trimmed so clean
// values like 9000 or 1.5 stay clean. 'f' never uses scientific notation,
// which Stripe would reject.
func formatMeterValue(v float64) string {
	// Digits left of the point ("0" counts as one). int64 covers every
	// realistic meter total (max ~1e13 GiB-seconds, far below int64's range).
	intDigits := len(strconv.FormatInt(int64(math.Abs(v)), 10))
	decimals := min(stripeMaxValueDecimals, stripeMaxValueDigits-intDigits)
	if decimals < 0 {
		decimals = 0
	}

	s := strconv.FormatFloat(v, 'f', decimals, 64)
	if strings.ContainsRune(s, '.') {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
