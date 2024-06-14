package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	axiom "github.com/axiomhq/axiom-go/axiom/otel"
)

type Tracer trace.Tracer

type Config struct {
	Dataset     string
	Application string
	Version     string
	AxiomToken  string
}

func New(ctx context.Context, config Config) (Tracer, func() error, error) {

	close, err := axiom.InitTracing(
		ctx,
		config.Dataset,
		config.Application,
		config.Version,
		axiom.SetNoEnv(),
		axiom.SetToken(config.AxiomToken),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("unable to init tracing: %w", err)
	}

	return otel.Tracer("main"), close, nil
}

func NewNoop() Tracer {
	return trace.NewNoopTracerProvider().Tracer("noop")
}
