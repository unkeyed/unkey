package keys

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	for i := 0; i < 100; i++ {

		key, err := NewV1Key("prefix", i)
		require.NoError(t, err)

		decodedKey := keyV1{}
		err = decodedKey.Unmarshal(key)
		require.NoError(t, err)

		require.NoError(t, err)
		require.Equal(t, "prefix", decodedKey.prefix)
		require.GreaterOrEqual(t, len(decodedKey.random), i)

	}
}
