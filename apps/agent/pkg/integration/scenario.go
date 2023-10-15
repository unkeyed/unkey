package integration

import (
	"context"
	"fmt"
	"log"
)

type stepFn func(ctx context.Context, env Env)

type Scenario struct {
	Name   string
	StepFn stepFn
}

func newScenario(name string, stepFn stepFn) Scenario {
	return Scenario{
		Name:   name,
		StepFn: stepFn,
	}
}

func (s Scenario) Run(ctx context.Context, env Env) {
	log.SetPrefix(fmt.Sprintf("[%s] - ", s.Name))
	log.SetFlags(0) // disable timestamp
	s.StepFn(ctx, env)
}
