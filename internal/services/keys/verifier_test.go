package keys_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestKeyVerifier_RejectsKeyspaceOutsideAllowlist(t *testing.T) {
	t.Parallel()

	verifier := &keys.KeyVerifier{
		Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
		Status: keys.StatusValid,
	}

	err := verifier.Verify(t.Context(), keys.WithKeyspaces("ks_other", "ks_another"))
	require.NoError(t, err)
	require.Equal(t, keys.StatusNotFound, verifier.Status)
}

func TestKeyVerifier_KeyspaceAllowlist(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		opts   []keys.VerifyOption
		status keys.KeyStatus
	}{
		{name: "omitted allows any keyspace", status: keys.StatusValid},
		{name: "matches second keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces("ks_other", "ks_actual")}, status: keys.StatusValid},
		{name: "empty rejects every keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces()}, status: keys.StatusNotFound},
		{name: "nil slice rejects every keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces([]string(nil)...)}, status: keys.StatusNotFound},
		{name: "prefix does not match", opts: []keys.VerifyOption{keys.WithKeyspaces("ks_actual_suffix")}, status: keys.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			verifier := &keys.KeyVerifier{
				Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
				Status: keys.StatusValid,
			}

			err := verifier.Verify(t.Context(), tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.status, verifier.Status)
		})
	}
}

func TestKeyVerifier_KeyspaceRejectionRecordsRejectContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := tracing.GetGlobalTraceProvider()
	tracing.SetGlobalTraceProvider(provider)
	t.Cleanup(func() {
		tracing.SetGlobalTraceProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	ctx, span := tracing.Start(t.Context(), "keys.verify")

	verifier := &keys.KeyVerifier{
		Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
		Status: keys.StatusValid,
	}

	require.NoError(t, verifier.Verify(ctx, keys.WithKeyspaces("ks_other", "ks_another")))
	require.Equal(t, keys.StatusNotFound, verifier.Status)
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)

	attrs := map[string]any{}
	for _, kv := range spans[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}
	require.Equal(t, "ks_actual", attrs["key_space_id"])
	require.Equal(t, []string{"ks_other", "ks_another"}, attrs["allowed_key_space_ids"])
}
