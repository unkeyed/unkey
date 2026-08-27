package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/svc/logdrain/sink/httpdrain"
)

func TestHTTPFormat_MapsStoredValues(t *testing.T) {
	tests := []struct {
		name     string
		stored   logdrainv1.HttpBodyFormat
		expected string
	}{
		{name: "unspecified defaults to JSON", stored: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_UNSPECIFIED, expected: httpdrain.FormatJSON},
		{name: "JSON", stored: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON, expected: httpdrain.FormatJSON},
		{name: "NDJSON", stored: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON, expected: httpdrain.FormatNDJSON},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := httpFormat(test.stored)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestHTTPFormat_RejectsUnknownValue(t *testing.T) {
	_, err := httpFormat(logdrainv1.HttpBodyFormat(99))
	require.EqualError(t, err, "unknown HTTP body format 99")
}

func TestParseConfig_RejectsMalformedWireFormat(t *testing.T) {
	_, err := parseConfig([]byte{0xff})
	require.ErrorContains(t, err, "decode logdrain config")
}
