package billingmeter

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Meter event names. These are the stable contract between this package and the
// Stripe Billing Meters defined in infra (terraform .../catalog/meters.tf): we
// post events against these names and Stripe's meters (formula "last")
// aggregate them and map them to metered prices. We never reference prices or
// meter IDs.
const (
	eventCPU    = "cpu_seconds"
	eventMemory = "memory_gib_seconds"
	eventEgress = "egress_public_gib"
	eventDisk   = "disk_gib_seconds"
	eventKeys   = "active_keys"
)

// payloadKeyCustomer and payloadKeyValue are the meter's
// customer_mapping / value_settings payload keys (defaults, set in meters.tf).
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

// SetMonthToDate posts one meter event per non-zero meter, carrying the
// workspace's month-to-date total as the meter value.
//
// No idempotency identifier is set: the meters aggregate with formula "last",
// so the billed quantity is always the value of the most recent event, never a
// sum. Retries, replays, and manual re-triggers therefore converge on the
// latest month-to-date total by construction — sending the same or a newer
// value again is harmless. (A stable identifier would be worse here: Stripe
// rejects a duplicate identifier with a hard 400, turning a harmless re-run
// into a failure.)
func (p *stripePusher) SetMonthToDate(ctx context.Context, req PushRequest) (int, error) {
	events := meterEventsFor(req)
	pushed := 0
	for _, e := range events {
		_, err := p.client.V1BillingMeterEvents.Create(ctx, &stripe.BillingMeterEventCreateParams{
			EventName: stripe.String(e.eventName),
			Timestamp: stripe.Int64(req.Timestamp),
			Payload: map[string]string{
				payloadKeyCustomer: req.StripeCustomerID,
				payloadKeyValue:    e.value,
			},
		})
		if err != nil {
			return pushed, fault.Wrap(err, fault.Internal("failed to push stripe meter event"))
		}
		pushed++
	}
	return pushed, nil
}

// meterEvent is one resolved meter event: which meter and the decimal value.
type meterEvent struct {
	eventName string
	value     string
}

// meterEventsFor builds the meter events for a push, one per meter with a
// positive value. Pure (no I/O) so the value formatting is unit tested without
// hitting Stripe.
func meterEventsFor(req PushRequest) []meterEvent {
	meters := []struct {
		name  string
		value float64
	}{
		{eventCPU, req.Values.CPUSeconds},
		{eventMemory, req.Values.MemoryGiBSeconds},
		{eventEgress, req.Values.EgressGiB},
		{eventDisk, req.Values.DiskGiBSeconds},
		{eventKeys, req.Values.ActiveKeys},
	}

	var out []meterEvent
	for _, m := range meters {
		if m.value <= 0 {
			continue
		}
		out = append(out, meterEvent{
			eventName: m.name,
			value:     formatMeterValue(m.value),
		})
	}
	return out
}

// stripeMaxValueDecimals is the most decimal places Stripe accepts in a meter
// event payload value. Full float64 precision (~15-17 significant digits)
// overruns this, so we round to it.
const stripeMaxValueDecimals = 12

// formatMeterValue renders a meter value as a plain decimal string within
// Stripe's 12-decimal-place limit. It rounds to 12 places, then trims trailing
// zeros (and a bare trailing dot) so clean values like 9000 or 1.5 stay clean.
// 'f' never uses scientific notation, which Stripe would reject.
func formatMeterValue(v float64) string {
	s := strconv.FormatFloat(v, 'f', stripeMaxValueDecimals, 64)
	if strings.ContainsRune(s, '.') {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
