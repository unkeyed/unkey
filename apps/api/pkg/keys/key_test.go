package keys

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	for i := 0; i < 100; i++ {

		key, err := NewKey("prefix", i)
		require.NoError(t, err)

		prefix, version, rest, err := DecodeKey(key)
		require.NoError(t, err)
		require.Equal(t, "prefix", prefix)
		require.Equal(t, uint8(0), version)
		require.GreaterOrEqual(t, len(rest), i)

	}
}
