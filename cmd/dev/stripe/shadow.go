package stripe

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/tui"
)

// Worker unit conversions (svc/ctrl/worker/cron/deploybilling/billing.go): the
// ClickHouse query reports memory/disk in GiB-hours and egress in bytes, while
// the meters bill in GiB-seconds and GiB. Kept in sync by hand; the fixture
// test TestMeterUsageHandComputedBill guards the same factors.
const (
	secondsPerHour = 3600.0
	bytesPerGiB    = float64(int64(1024 * 1024 * 1024))
)

// relativeTolerance is the drift a meter may show before shadow flags it. The
// hourly push can lag the live query by up to an hour, so a small difference
// is usually that lag rather than a real problem. Set above the lag, below
// any difference that would indicate the pipeline is dropping or double-counting.
const relativeTolerance = 0.001 // 0.1%

// shadowCmd recomputes a workspace's month-to-date Deploy usage from ClickHouse
// and diffs it against the values Stripe holds for the customer's meters. Use it
// before enabling billing to check the query, hourly push, and meters agree.
var shadowCmd = &cli.Command{
	Name:  "shadow",
	Usage: "Diff a workspace's ClickHouse usage against Stripe's meter values",
	Flags: []cli.Flag{
		keyFlag(),
		cli.String("clickhouse-url", "ClickHouse DSN", cli.EnvVar("CLICKHOUSE_URL"), cli.Required()),
		cli.String("workspace", "Workspace id (for the ClickHouse query)", cli.Required()),
		cli.String("customer", "Stripe customer id (cus_...)", cli.Required()),
		cli.String("month", "Billing month YYYY-MM (default: current)", cli.Default("")),
	},
	Action: shadow,
}

// meterValues holds one workspace's usage in the exact units each Stripe meter
// expects, keyed by the meter's event_name so it lines up with the summaries.
type meterValues map[string]float64

func shadow(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}
	out := tui.New(os.Stdout)

	period, err := resolvePeriod(ctx, sc, cmd.String("month"))
	if err != nil {
		return err
	}
	start := period.Start()
	end := period.End()

	out.Printf("Shadow %s  workspace=%s  customer=%s\n",
		out.Bold(start.Format("2006-01")), cmd.RequireString("workspace"), cmd.RequireString("customer"))
	out.Println(out.Dim(fmt.Sprintf("  window %s -> %s", start.Format("2006-01-02"), end.Format("2006-01-02"))))

	chValues, err := clickhouseUsage(ctx, cmd, start.UnixMilli(), end.UnixMilli())
	if err != nil {
		return err
	}

	stripeValues, err := stripeMeterValues(ctx, sc, cmd.RequireString("customer"), start.Unix(), end.Unix())
	if err != nil {
		return err
	}

	return report(out, chValues, stripeValues)
}

// resolvePeriod picks the billing period: the --month flag if given, else the
// current calendar month from the local clock. For clocked test customers
// whose "now" is not wall-clock time, pass --month explicitly.
func resolvePeriod(_ context.Context, _ *stripesdk.Client, month string) (billingperiod.Period, error) {
	if month != "" {
		return billingperiod.Parse(month)
	}
	now := time.Now().UTC()
	return billingperiod.Period{Year: now.Year(), Month: now.Month()}, nil
}

// clickhouseUsage runs the production billing queries for the window and folds
// them into meter units, exactly as the worker does before pushing.
func clickhouseUsage(ctx context.Context, cmd *cli.Command, startMillis, endMillis int64) (meterValues, error) {
	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.RequireString("clickhouse-url")})
	if err != nil {
		return nil, fmt.Errorf("connect clickhouse: %w", err)
	}
	defer func() { _ = ch.Close() }()

	workspace := cmd.RequireString("workspace")

	usage, err := ch.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
		WorkspaceID: workspace,
		Start:       startMillis,
		End:         endMillis,
	})
	if err != nil {
		return nil, fmt.Errorf("query instance usage: %w", err)
	}

	var cpuSeconds, memoryGiBHours, diskGiBHours float64
	var egressBytes int64
	for _, r := range usage {
		cpuSeconds += r.CPUSeconds
		memoryGiBHours += r.MemoryGiBHours
		diskGiBHours += r.DiskGiBHours
		egressBytes += r.EgressBytes
	}

	keys, err := ch.GetActiveKeysUsage(ctx, clickhouse.GetActiveKeysUsageRequest{
		WorkspaceID: workspace,
		Month:       startMillis,
	})
	if err != nil {
		return nil, fmt.Errorf("query active keys: %w", err)
	}
	var activeKeys float64
	for _, k := range keys {
		activeKeys += float64(k.ActiveKeys)
	}

	return meterValues{
		"cpu_seconds":        cpuSeconds,
		"memory_gib_seconds": memoryGiBHours * secondsPerHour,
		"egress_public_gib":  float64(egressBytes) / bytesPerGiB,
		"disk_gib_seconds":   diskGiBHours * secondsPerHour,
		"active_keys":        activeKeys,
	}, nil
}

// stripeMeterValues reads the value Stripe currently holds for each meter: the
// "last"-aggregated summary over the window is the most recent month-to-date
// total the worker pushed.
func stripeMeterValues(ctx context.Context, sc *stripesdk.Client, customer string, startUnix, endUnix int64) (meterValues, error) {
	values := meterValues{}

	meters := sc.V1BillingMeters.List(ctx, &stripesdk.BillingMeterListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(100)},
	})
	for meter, err := range meters.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list meters: %w", err)
		}

		summaries := sc.V1BillingMeterEventSummaries.List(ctx, &stripesdk.BillingMeterEventSummaryListParams{
			ID:        stripesdk.String(meter.ID),
			Customer:  stripesdk.String(customer),
			StartTime: stripesdk.Int64(startUnix),
			EndTime:   stripesdk.Int64(endUnix),
		})
		var value float64
		for summary, sErr := range summaries.All(ctx) {
			if sErr != nil {
				return nil, fmt.Errorf("meter %s summaries: %w", meter.EventName, sErr)
			}
			// One summary for the whole window (no grouping requested); for a
			// "last" meter its aggregated value is the latest MTD push.
			value = summary.AggregatedValue
		}
		values[meter.EventName] = value
	}

	return values, nil
}

// report prints a per-meter ClickHouse-vs-Stripe table and returns an error if
// any meter drifts beyond the tolerance, so scripts can gate on the exit code.
func report(out *tui.Renderer, ch, stripe meterValues) error {
	order := []string{"cpu_seconds", "memory_gib_seconds", "egress_public_gib", "disk_gib_seconds", "active_keys"}

	drifted := 0
	out.Blank()
	for _, meter := range order {
		chVal := ch[meter]
		stripeVal := stripe[meter]
		diff := chVal - stripeVal

		ok := withinTolerance(chVal, stripeVal)
		verdict := out.Green("ok")
		if !ok {
			verdict = out.Red("DRIFT")
			drifted++
		}
		out.Printf("  %-20s clickhouse=%-16.6f stripe=%-16.6f diff=%-14.6f %s\n",
			meter, chVal, stripeVal, diff, verdict)
	}
	out.Blank()

	if drifted > 0 {
		return fmt.Errorf("%d meter(s) drifted beyond %.3f%%; a gap this large is more than the hourly push lag can explain",
			drifted, relativeTolerance*100)
	}
	out.Println(out.Green("All meters agree within tolerance."))
	return nil
}

// withinTolerance compares two meter values with a relative tolerance, treating
// two zeros as agreement and any zero-vs-nonzero as drift.
func withinTolerance(a, b float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	return math.Abs(a-b)/scale <= relativeTolerance
}
