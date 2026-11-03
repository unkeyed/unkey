package containers

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var loadContainerTracing = sync.OnceValue(newContainerTracing)

type containerTracing struct {
	provider      *sdktrace.TracerProvider
	tracer        trace.Tracer
	packageParent trace.SpanContext
	mu            sync.Mutex
	// Retain TB identities until process exit, including repeated test names.
	tests map[testing.TB]trace.SpanContext
}

func startContainerSpan(t testing.TB, name string, attributes ...attribute.KeyValue) trace.Span {
	t.Helper()

	tracing := loadContainerTracing()
	if tracing == nil {
		return trace.SpanFromContext(context.Background())
	}
	parent := tracing.testParent(t)
	if !parent.IsValid() {
		return trace.SpanFromContext(context.Background())
	}
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), parent)
	_, span := tracing.tracer.Start(ctx, name, trace.WithAttributes(attributes...))
	return span
}

func newContainerTracing() *containerTracing {
	ctx := propagation.TraceContext{}.Extract(context.Background(), propagation.MapCarrier{
		"traceparent": os.Getenv("RASK_TRACEPARENT"),
	})
	parent := trace.SpanContextFromContext(ctx)
	dataset := os.Getenv("RASK_AXIOM_DATASET")
	if dataset == "" {
		dataset = "rask-traces"
	}
	token := os.Getenv("RASK_AXIOM_TRACE_TOKEN")
	if token == "" {
		token = os.Getenv("RASK_AXIOM_TOKEN")
	}
	if !parent.IsValid() || token == "" {
		return nil
	}
	endpoint := os.Getenv("RASK_AXIOM_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://us-east-1.aws.edge.axiom.co/v1/traces"
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil || endpointURL.Host == "" || (endpointURL.Scheme != "https" && endpointURL.Scheme != "http") {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":   "Bearer " + token,
			"X-Axiom-Dataset": dataset,
		}),
		otlptracehttp.WithTimeout(500*time.Millisecond),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled: false, InitialInterval: 0, MaxInterval: 0, MaxElapsedTime: 0,
		}),
	)
	if err != nil {
		return nil
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "unkey-test-containers"))),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(256),
			sdktrace.WithMaxExportBatchSize(64),
			sdktrace.WithBatchTimeout(200*time.Millisecond),
			sdktrace.WithExportTimeout(500*time.Millisecond),
		),
	)
	return &containerTracing{
		provider:      provider,
		tracer:        provider.Tracer("github.com/unkeyed/unkey/pkg/testutil/containers"),
		packageParent: parent,
		mu:            sync.Mutex{},
		tests:         make(map[testing.TB]trace.SpanContext),
	}
}

func (c *containerTracing) testParent(t testing.TB) trace.SpanContext {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()
	if parent, ok := c.tests[t]; ok {
		return parent
	}
	var id trace.SpanID
	if _, err := rand.Read(id[:]); err != nil || !id.IsValid() {
		return trace.SpanContext{}
	}
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    c.packageParent.TraceID(),
		SpanID:     id,
		TraceFlags: trace.FlagsSampled,
		TraceState: c.packageParent.TraceState(),
		Remote:     true,
	})
	c.tests[t] = parent
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(trace.ContextWithRemoteSpanContext(context.Background(), parent), carrier)
	// Rask owns the recording test span; the child only claims its identity.
	t.Attr("rask.test.traceparent.v1", carrier.Get("traceparent"))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := c.provider.ForceFlush(ctx); err != nil {
			t.Log("containers tracing flush incomplete")
		}
	})
	return parent
}
