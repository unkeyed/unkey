package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var ResourceCleanupRowsDeleted = lazy.NewCounterVec(prometheus.CounterOpts{
	Namespace: "unkey", Subsystem: "control", Name: "resource_cleanup_rows_deleted_total",
	Help: "Rows reclaimed by resource cleanup, including retries before journaling.",
}, []string{"table"})

var ResourceCleanupLastScanTimestamp = lazy.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "unkey", Subsystem: "control", Name: "resource_cleanup_last_scan_timestamp_seconds",
	Help: "Unix timestamp of the last completed scan, or the first attempt until a scan completes.",
}, []string{"table"})
