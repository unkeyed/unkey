package otel

import (
	"context"
	"fmt"

	adapter "github.com/axiomhq/axiom-go/adapters/slog"

	axiomOtel "github.com/axiomhq/axiom-go/axiom/otel"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
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
