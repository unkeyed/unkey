package sim

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
)

type Seed [32]byte

func (s Seed) String() string {
	return base64.StdEncoding.EncodeToString(s[:])
}

// NewSeed generates a cryptographically secure random 32-byte seed
// suitable for simulations requiring high entropy and unpredictability.
func NewSeed() Seed {
	var seed [32]byte
	_, err := rand.Read(seed[:])
	if err != nil {
		panic(err)
	}
	return seed
}

func NewSeedFromInt(i int) Seed {
	// Create a seed from the integer value

	seed := [32]byte{}
	binary.BigEndian.PutUint32(seed[:], uint32(i)) //nolint:gosec

	return seed
}
