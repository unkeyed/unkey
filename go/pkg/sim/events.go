package sim

import (
	"errors"
	"math/rand"
)

type Idle struct {
}

var _ Event[any] = (*Idle)(nil)

func (i Idle) Name() string {
	return "Idle"
}

func (i Idle) Run(rng *rand.Rand, state *any) error {
	return nil
}

type Fail struct {
	Message string
}

var _ Event[any] = (*Fail)(nil)

func (f Fail) Name() string {
	return "Fail"
}

func (f Fail) Run(rng *rand.Rand, state *any) error {
	return errors.New(f.Message)
}
