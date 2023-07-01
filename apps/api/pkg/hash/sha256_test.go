package hash

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSha256(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := uuid.NewString()
		h := Sha256(s)
		require.Greater(t, len(h), 10)

		// check if it's consistent
		require.Equal(t, h, Sha256(s))
	}
}
