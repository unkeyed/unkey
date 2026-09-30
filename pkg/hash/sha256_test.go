package hash

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSha256(t *testing.T) {
	for range 100 {
		b := []byte{32}
		_, err := rand.Read(b)
		require.NoError(t, err)
		s := string(b)
		h := Sha256(s)
		require.Greater(t, len(h), 10)

		// check if it's consistent
		require.Equal(t, h, Sha256(s))
	}
}
