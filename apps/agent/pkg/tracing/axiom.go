package tracing

import (
	"context"
	"fmt"

	axiom "github.com/axiomhq/axiom-go/axiom/otel"
	"go.opentelemetry.io/otel"
)

type Config struct {
	Dataset     string
	Application string
	Version     string
	AxiomToken  string
}

// Coser is a function that closes the global tracer.
type Closer func() error

func Init(ctx context.Context, config Config) (Closer, error) {

	close, err := axiom.InitTracing(
		ctx,
		config.Dataset,
		config.Application,
		config.Version,
		axiom.SetNoEnv(),
		axiom.SetToken(config.AxiomToken),
	)

	if err != nil {
		return nil, fmt.Errorf("unable to init tracing: %w", err)
	}

	globalTracer = otel.Tracer("main")
	return close, nil
}
