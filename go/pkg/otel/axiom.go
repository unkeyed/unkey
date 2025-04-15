package otel

import (
	"context"
	"fmt"
	"runtime"
	"time"

	adapter "github.com/axiomhq/axiom-go/adapters/slog"

	axiomOtel "github.com/axiomhq/axiom-go/axiom/otel"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	"github.com/unkeyed/unkey/go/pkg/shutdown"

	"go.opentelemetry.io/otel"
)

// AxiomConfig defines the configuration settings for OpenTelemetry integration with Axiom.
// It specifies connection details and application metadata needed for proper telemetry.
type AxiomConfig struct {

	// Application is the name of your application, used to identify the source of telemetry data.
	// This appears in Axiom dashboards and alerts.
	Application string

	// Version is the current version of your application, allowing you to correlate
	// behavior changes with specific releases.
	Version string
}

// InitAxiom initializes the global tracer and logging providers for OpenTelemetry,
// configured to send telemetry data to Axiom.
//
// It sets up:
// - Distributed tracing using the OTLP HTTP exporter with Axiom endpoints
// - Logging via OTLP HTTP exporter to Axiom
//
// The function registers all necessary shutdown handlers with the provided shutdowns instance.
// These handlers will be called during application termination to ensure proper cleanup.
//
// Example:
//
//	shutdowns := shutdown.New()
//	err := otel.InitAxiom(ctx, otel.AxiomConfig{
//	    AxiomAPIToken: "your-axiom-api-token",
//	    AxiomURL:      "https://api.axiom.co",
//	    Application:   "unkey-api",
//	    Version:       version.Version,
//	}, shutdowns)
//
//	if err != nil {
//	    log.Fatalf("Failed to initialize telemetry: %v", err)
//	}
//
//	// Later during shutdown:
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	errs := shutdowns.Shutdown(ctx)
//	for _, err := range errs {
//	    log.Printf("Shutdown error: %v", err)
//	}
func InitAxiom(ctx context.Context, config AxiomConfig, shutdowns *shutdown.Shutdowns) error {

	logger, err := adapter.New(adapter.SetDataset("apiv2_logs"))
	if err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}
	shutdowns.Register(func() error {
		logger.Close()
		return nil
	})

	logging.AddHandler(logger)

	traceProvider, err := axiomOtel.TracerProvider(context.Background(), "apiv2_traces", config.Application, config.Version)
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	shutdowns.RegisterCtx(traceProvider.Shutdown)

	// Set the global trace provider
	otel.SetTracerProvider(traceProvider)
	tracing.SetGlobalTraceProvider(traceProvider)

	return nil
}

// temporary function to get some cpu and memory metrics into axiom until we have prometheus
// this already starts a new goroutine
func EmitSystemMetrics(logger logging.Logger) {

	repeat.Every(15*time.Second, func() {

		vm, err := mem.VirtualMemory()
		if err != nil {
			logger.Error("failed to get virtual memory metrics", "error", err)
			return
		}

		cpuPercentage, err := cpu.Percent(time.Second, false)
		if err != nil {
			logger.Error("failed to get cpu metrics", "error", err)
			return
		}
		if len(cpuPercentage) == 0 {
			logger.Error("cpu metrics returned empty")
			return
		}

		loadAvg, err := load.Avg()
		if err != nil {
			logger.Error("failed to get load avg", "error", err)
			return
		}

		logger.Info("system_metrics",
			"goroutines", runtime.NumGoroutine(),
			"memory_total", vm.Total,
			"memory_used", vm.Used,
			"memory_free", vm.Free,
			"memory_available", vm.Available,
			"memory_percent", vm.UsedPercent,
			"cpu_percent", cpuPercentage[0],
			"load_1", loadAvg.Load1,
			"load_5", loadAvg.Load5,
			"load_15", loadAvg.Load15,
		)
	})

}
