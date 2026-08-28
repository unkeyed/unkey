package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig_RejectsMalformedWireFormat(t *testing.T) {
	_, err := parseConfig([]byte{0xff})
	require.ErrorContains(t, err, "decode logdrain config")
}
