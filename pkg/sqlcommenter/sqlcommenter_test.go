package sqlcommenter

import (
	"context"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestFormat(t *testing.T) {
	t.Run("empty returns empty string", func(t *testing.T) {
		require.Equal(t, "", Format(nil))
		require.Equal(t, "", Format(map[string]string{}))
	})

	t.Run("keys are sorted alphabetically", func(t *testing.T) {
		got := Format(map[string]string{
			"z": "1",
			"a": "2",
			"m": "3",
		})
		require.Equal(t, "/*a='2',m='3',z='1'*/", got)
	})

	t.Run("values are percent-encoded", func(t *testing.T) {
		got := Format(map[string]string{
			"route": "POST /v1/keys.createKey",
		})
		require.Equal(t, "/*route='POST%20%2Fv1%2Fkeys.createKey'*/", got)
	})

	t.Run("single quotes in values are escaped", func(t *testing.T) {
		got := Format(map[string]string{"k": "can't"})
		require.Equal(t, "/*k='can%27t'*/", got)
	})

	t.Run("equals sign in value is encoded", func(t *testing.T) {
		got := Format(map[string]string{"k": "a=b"})
		require.Equal(t, "/*k='a%3Db'*/", got)
	})

	t.Run("plus sign is encoded as %2B not retained as +", func(t *testing.T) {
		got := Format(map[string]string{"k": "1+1"})
		require.Equal(t, "/*k='1%2B1'*/", got)
	})
}

func TestAppend(t *testing.T) {
	t.Run("empty comment is a no-op", func(t *testing.T) {
		require.Equal(t, "SELECT 1", Append("SELECT 1", ""))
	})

	t.Run("no trailing semicolon", func(t *testing.T) {
		require.Equal(t, "SELECT 1 /*a='b'*/", Append("SELECT 1", "/*a='b'*/"))
	})

	t.Run("trailing semicolon: comment goes before it", func(t *testing.T) {
		require.Equal(t, "SELECT 1 /*a='b'*/;", Append("SELECT 1;", "/*a='b'*/"))
	})

	t.Run("trailing whitespace then semicolon", func(t *testing.T) {
		require.Equal(t, "SELECT 1 /*a='b'*/;", Append("SELECT 1;  \n", "/*a='b'*/"))
	})
}

func TestFromContext(t *testing.T) {
	t.Run("no request metadata, no span", func(t *testing.T) {
		got := FromContext(context.Background(), "api")
		require.Equal(t, "/*application='api',db_driver='go-database-sql'*/", got)
	})

	t.Run("empty app omits the application tag", func(t *testing.T) {
		got := FromContext(context.Background(), "")
		require.Equal(t, "/*db_driver='go-database-sql'*/", got)
	})

	t.Run("with request metadata", func(t *testing.T) {
		ctx := WithRequest(context.Background(), "POST /v1/keys.createKey", "req_abc123")
		got := FromContext(ctx, "api")
		require.Equal(t,
			"/*application='api',db_driver='go-database-sql',request_id='req_abc123',route='POST%20%2Fv1%2Fkeys.createKey'*/",
			got,
		)
	})

	t.Run("with valid sampled span yields traceparent", func(t *testing.T) {
		traceID, err := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
		require.NoError(t, err)
		spanID, err := trace.SpanIDFromHex("b7ad6b7169203331")
		require.NoError(t, err)

		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)
		got := FromContext(ctx, "api")

		require.Contains(t, got, "traceparent='00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01'")
	})

	t.Run("unsampled span uses 00 trace flags", func(t *testing.T) {
		traceID, err := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
		require.NoError(t, err)
		spanID, err := trace.SpanIDFromHex("b7ad6b7169203331")
		require.NoError(t, err)
		sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)

		require.Contains(t, FromContext(ctx, "api"), "-00'")
	})
}

// TestRoundTrip parses the formatted comment back out and confirms every tag
// survives encoding intact. Guards against the encoder and decoder drifting
// apart over time.
func TestRoundTrip(t *testing.T) {
	tags := map[string]string{
		"application": "api",
		"route":       "POST /v1/keys.createKey?foo=bar",
		"request_id":  "req_abc123",
	}
	comment := Format(tags)

	pairRe := regexp.MustCompile(`([a-z_]+)='([^']*)'`)
	matches := pairRe.FindAllStringSubmatch(comment, -1)
	require.Len(t, matches, len(tags))

	for _, m := range matches {
		key, encoded := m[1], m[2]
		want, ok := tags[key]
		require.True(t, ok, "unexpected key %q", key)
		decoded, err := url.QueryUnescape(encoded)
		require.NoError(t, err)
		require.Equal(t, want, decoded)
	}
}
