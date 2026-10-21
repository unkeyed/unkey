package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
)

type probeStep struct {
	client int
	finish bool
}

type probe struct {
	release  chan struct{}
	done     chan error
	admitted bool
}

func checkProbeLimit(ctx context.Context) (result, error) {
	r := result{Name: "circuitbreaker-limit", Cases: 0, Violations: 0, Counterexample: nil}
	for _, clients := range []int{2, 3} {
		for _, limit := range []int{1, 2} {
			err := schedules(make([]int, clients), nil, func(steps []probeStep) error {
				trace, violation, err := replayProbes(ctx, clients, limit, steps)
				if err != nil {
					return err
				}
				r.observe(trace, violation)
				return nil
			})
			if err != nil {
				return r, err
			}
		}
	}
	return r, nil
}

func schedules(phases []int, steps []probeStep, visit func([]probeStep) error) error {
	if len(steps) == 2*len(phases) {
		return visit(steps)
	}
	for client, phase := range phases {
		if phase == 2 {
			continue
		}
		phases[client]++
		err := schedules(phases, append(steps, probeStep{client: client, finish: phase == 1}), visit)
		phases[client]--
		if err != nil {
			return err
		}
	}
	return nil
}

func replayProbes(ctx context.Context, clients, limit int, steps []probeStep) (trace []string, violation string, err error) {
	cb, err := recoveringBreaker(ctx, limit)
	if err != nil {
		return nil, "", err
	}
	probes := make([]probe, clients)
	defer func() {
		for _, p := range probes {
			if p.release != nil {
				close(p.release)
				err = errors.Join(err, <-p.done)
			}
		}
	}()
	trace = []string{fmt.Sprintf("trip breaker; advance beyond cooldown and counter reset; probe limit=%d", limit)}
	active, successes := 0, 0
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return trace, "", err
		}
		p := &probes[step.client]
		if step.finish {
			if p.admitted {
				close(p.release)
				p.release = nil
				if err := <-p.done; err != nil {
					return trace, "", err
				}
				active--
				successes++
			}
			trace = append(trace, fmt.Sprintf("finish client %d", step.client))
			continue
		}
		admission := make(chan error, 1)
		release := make(chan struct{})
		done := make(chan error, 1)
		p.release, p.done = release, done
		go func() {
			entered := false
			_, callErr := cb.Do(ctx, func(ctx context.Context) (struct{}, error) {
				entered = true
				admission <- nil
				select {
				case <-release:
					return struct{}{}, nil
				case <-ctx.Done():
					return struct{}{}, ctx.Err()
				}
			})
			if !entered {
				admission <- callErr
			}
			done <- callErr
		}()
		select {
		case err := <-admission:
			p.admitted = err == nil
			if err != nil {
				p.release = nil
				if doneErr := <-p.done; !errors.Is(doneErr, circuitbreaker.ErrTooManyRequests) {
					return trace, "", fmt.Errorf("unexpected admission error: %w", doneErr)
				}
			}
		case <-ctx.Done():
			return trace, "", ctx.Err()
		}
		trace = append(trace, fmt.Sprintf("start client %d: admitted=%t", step.client, p.admitted))
		if p.admitted {
			active++
		}
		if successes < limit && active > limit {
			return trace, fmt.Sprintf("%d probes in flight before recovery, maximum is %d", active, limit), nil
		}
	}
	return trace, "", nil
}

func recoveringBreaker(ctx context.Context, limit int) (*circuitbreaker.CB[struct{}], error) {
	clk := clock.NewTestClock(time.Unix(0, 0))
	cb := circuitbreaker.New[struct{}]("formalverify",
		circuitbreaker.WithClock(clk), circuitbreaker.WithTripThreshold(1),
		circuitbreaker.WithMaxRequests(limit), circuitbreaker.WithTimeout(time.Second),
		circuitbreaker.WithCyclicPeriod(time.Second))
	downstreamErr := errors.New("modeled downstream failure")
	_, err := cb.Do(ctx, func(context.Context) (struct{}, error) { return struct{}{}, downstreamErr })
	if !errors.Is(err, downstreamErr) {
		return nil, fmt.Errorf("trip setup returned %v instead of downstream failure", err)
	}
	called := false
	_, err = cb.Do(ctx, func(context.Context) (struct{}, error) {
		called = true
		return struct{}{}, nil
	})
	if called || !errors.Is(err, circuitbreaker.ErrTripped) {
		return nil, fmt.Errorf("trip setup did not block the next request: called=%t error=%v", called, err)
	}
	clk.Tick(2 * time.Second)
	return cb, nil
}

func checkRecovery(ctx context.Context) (result, error) {
	r := result{Name: "circuitbreaker-recovery", Cases: 0, Violations: 0, Counterexample: nil}
	for limit := 1; limit <= 3; limit++ {
		cb, err := recoveringBreaker(ctx, limit)
		if err != nil {
			return r, err
		}
		violation := ""
		for request := 0; request <= limit; request++ {
			called := false
			_, err := cb.Do(ctx, func(context.Context) (struct{}, error) {
				called = true
				return struct{}{}, nil
			})
			if !called || err != nil {
				violation = fmt.Sprintf("request %d rejected after %d successful probes: %v", request+1, request, err)
				break
			}
		}
		r.observe([]string{fmt.Sprintf("trip; wait; complete %d successful probes; request again", limit)}, violation)
	}
	return r, nil
}
