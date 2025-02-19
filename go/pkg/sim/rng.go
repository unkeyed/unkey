package sim

import "math/rand/v2"

type source struct {
	seed uint64
}

var _ rand.Source = (*source)(nil)

func (s source) Uint64() uint64 {
	return s.seed
}
