//go:build verifyratelimit

package ratelimit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
)

const (
	formalWorkspace = "ws_formal"
	formalNamespace = "requests"
	formalDuration  = time.Second
)

// formalOrigin is an exact one-second window boundary in 2033. NewLocal's
// janitor reads the wall clock, so these cells stay put for the whole run.
var formalOrigin = time.Unix(2_000_000_000, 0).UTC()

func TestFormalCommittedCount(t *testing.T) {
	checked, err := formalCheck()
	require.NoError(t, err)
	t.Logf("pass: %d cases", checked)
}

func formalCheck() (int, error) {
	checked := 0
	limits := []int64{1, 2, 3, 4}
	costs := []int64{0, 1, 2}
	for _, limit := range limits {
		for length := 1; length <= 4; length++ {
			seq := make([]int64, length)
			var walk func(int) error
			walk = func(i int) error {
				if i == length {
					checked++
					return formalRunSingle(limit, formalOrigin, seq)
				}
				for _, cost := range costs {
					seq[i] = cost
					if err := walk(i + 1); err != nil {
						return err
					}
				}
				return nil
			}
			if err := walk(0); err != nil {
				return checked, err
			}
		}
	}

	offsets := []time.Duration{
		0,
		500 * time.Millisecond,
		999 * time.Millisecond,
		time.Second,
		1500 * time.Millisecond,
		2 * time.Second,
	}
	for _, limit := range []int64{1, 3} {
		for i, first := range offsets {
			for j := i; j < len(offsets); j++ {
				for k := j; k < len(offsets); k++ {
					for _, c1 := range costs {
						for _, c2 := range costs {
							for _, c3 := range costs {
								checked++
								err := formalRunSteps(limit, []formalStep{
									{formalOrigin.Add(first), c1},
									{formalOrigin.Add(offsets[j]), c2},
									{formalOrigin.Add(offsets[k]), c3},
								})
								if err != nil {
									return checked, err
								}
							}
						}
					}
				}
			}
		}
	}

	for _, limit := range []int64{1, 2, 3} {
		for _, left := range []int64{0, 1, 2, 3} {
			for _, right := range []int64{0, 1, 2, 3} {
				checked++
				if err := formalRunBatch(limit, "same", "same", left, right); err != nil {
					return checked, err
				}
				checked++
				if err := formalRunBatch(limit, "left", "right", left, right); err != nil {
					return checked, err
				}
			}
		}
	}

	for _, limit := range limits {
		checked++
		if err := formalRunBoundary(limit); err != nil {
			return checked, err
		}
	}
	checked++
	if err := formalRunWorkspaceIsolation(); err != nil {
		return checked, err
	}
	return checked, nil
}

type formalStep struct {
	at   time.Time
	cost int64
}

func formalRunSingle(limit int64, at time.Time, costs []int64) error {
	steps := make([]formalStep, len(costs))
	for i, cost := range costs {
		steps[i] = formalStep{at: at, cost: cost}
	}
	return formalRunSteps(limit, steps)
}

func formalRunSteps(limit int64, steps []formalStep) error {
	svc := ratelimit.NewLocal(nil)
	defer svc.Close()
	taken := make([]formalStep, 0, len(steps))
	allowed := make([]bool, 0, len(steps))
	for _, next := range steps {
		before, err := formalProbe(svc, "user", limit, next.at)
		if err != nil {
			return formalStory(limit, taken, allowed, err)
		}
		resp, err := formalTake(svc, formalWorkspace, "user", limit, next.cost, next.at)
		if err != nil {
			return formalStory(limit, taken, allowed, err)
		}
		after, err := formalProbe(svc, "user", limit, next.at)
		if err != nil {
			return formalStory(limit, append(taken, next), append(allowed, resp.Success), err)
		}
		fits := before+next.cost <= limit
		switch {
		case fits && (!resp.Success || after != before+next.cost || resp.Current != after || resp.Remaining != limit-after):
			err = fmt.Errorf("cost %d fit under limit %d with committed current %d, but Ratelimit returned success=%v current=%d remaining=%d and the next zero-cost read saw %d", next.cost, limit, before, resp.Success, resp.Current, resp.Remaining, after)
		case !fits && (resp.Success || after != before || resp.Remaining != 0 || resp.Current != before+next.cost):
			err = fmt.Errorf("cost %d did not fit under limit %d with committed current %d, but Ratelimit returned success=%v current=%d remaining=%d and the next zero-cost read saw %d", next.cost, limit, before, resp.Success, resp.Current, resp.Remaining, after)
		case !resp.Reset.Equal(formalWindowEnd(next.at)):
			err = fmt.Errorf("reset %s is not the end of the fixed window containing %s", resp.Reset.UTC().Format(time.RFC3339Nano), next.at.UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			return formalStory(limit, append(taken, next), append(allowed, resp.Success), err)
		}
		taken = append(taken, next)
		allowed = append(allowed, resp.Success)
	}
	return nil
}

func formalRunBatch(limit int64, leftID, rightID string, leftCost, rightCost int64) error {
	svc := ratelimit.NewLocal(nil)
	defer svc.Close()
	leftBefore, err := formalProbe(svc, leftID, limit, formalOrigin)
	if err != nil {
		return err
	}
	rightBefore := leftBefore
	if leftID != rightID {
		rightBefore, err = formalProbe(svc, rightID, limit, formalOrigin)
		if err != nil {
			return err
		}
	}
	responses, err := svc.RatelimitMany(context.Background(), []ratelimit.RatelimitRequest{
		formalRequest(formalWorkspace, leftID, limit, leftCost, formalOrigin),
		formalRequest(formalWorkspace, rightID, limit, rightCost, formalOrigin),
	})
	if err != nil {
		return err
	}
	if len(responses) != 2 {
		return fmt.Errorf("RatelimitMany returned %d responses", len(responses))
	}
	leftFits := leftBefore+leftCost <= limit
	rightFits := rightBefore+rightCost <= limit
	if leftID == rightID {
		rightFits = leftBefore+leftCost+rightCost <= limit
	}
	if responses[0].Success != leftFits || responses[1].Success != rightFits {
		return fmt.Errorf("batch costs %d and %d on %s and %s, limit %d: individual fit %v and %v, Success %v and %v", leftCost, rightCost, leftID, rightID, limit, leftFits, rightFits, responses[0].Success, responses[1].Success)
	}
	allFit := leftFits && rightFits
	leftAfter, err := formalProbe(svc, leftID, limit, formalOrigin)
	if err != nil {
		return err
	}
	wantLeft := leftBefore
	if allFit {
		wantLeft += leftCost
		if leftID == rightID {
			wantLeft += rightCost
		}
	}
	if leftAfter != wantLeft {
		return fmt.Errorf("batch on %s committed current went from %d to %d, want %d (costs %d and %d, limit %d, all fit %v)", leftID, leftBefore, leftAfter, wantLeft, leftCost, rightCost, limit, allFit)
	}
	if leftID == rightID {
		return nil
	}
	rightAfter, err := formalProbe(svc, rightID, limit, formalOrigin)
	if err != nil {
		return err
	}
	wantRight := rightBefore
	if allFit {
		wantRight += rightCost
	}
	if rightAfter != wantRight {
		return fmt.Errorf("batch on %s committed current went from %d to %d, want %d (cost %d, limit %d, all fit %v)", rightID, rightBefore, rightAfter, wantRight, rightCost, limit, allFit)
	}
	return nil
}

func formalRunBoundary(limit int64) error {
	svc := ratelimit.NewLocal(nil)
	defer svc.Close()
	var committed int64
	for committed+1 <= limit {
		resp, err := formalTake(svc, formalWorkspace, "user", limit, 1, formalOrigin)
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("filling the window rejected cost 1 at committed current %d, limit %d", committed, limit)
		}
		committed++
	}
	next := formalOrigin.Add(formalDuration)
	current, err := formalProbe(svc, "user", limit, next)
	if err != nil {
		return err
	}
	if current != committed {
		return fmt.Errorf("at the first instant of the next window the committed current was %d, want the whole previous window %d", current, committed)
	}
	resp, err := formalTake(svc, formalWorkspace, "user", limit, 1, next)
	if err != nil {
		return err
	}
	if resp.Success {
		return fmt.Errorf("cost 1 was allowed at the start of the next window while the previous window still held %d of limit %d", committed, limit)
	}
	after, err := formalProbe(svc, "user", limit, next)
	if err != nil {
		return err
	}
	if after != committed {
		return fmt.Errorf("the denied request at the next window changed the committed current from %d to %d", committed, after)
	}
	empty, err := formalProbe(svc, "user", limit, formalOrigin.Add(2*formalDuration))
	if err != nil {
		return err
	}
	if empty != 0 {
		return fmt.Errorf("two windows later, with the previous window empty, the committed current was %d, want 0", empty)
	}
	return nil
}

func formalRunWorkspaceIsolation() error {
	svc := ratelimit.NewLocal(nil)
	defer svc.Close()
	resp, err := formalTake(svc, formalWorkspace, "user", 1, 1, formalOrigin)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("workspace %s rejected the first cost", formalWorkspace)
	}
	other, err := formalProbeWorkspace(svc, "ws_other", "user", 1, formalOrigin)
	if err != nil {
		return err
	}
	if other != 0 {
		return fmt.Errorf("workspace ws_other saw committed current %d after %s took 1", other, formalWorkspace)
	}
	return nil
}

func formalProbe(svc ratelimit.Service, identifier string, limit int64, at time.Time) (int64, error) {
	return formalProbeWorkspace(svc, formalWorkspace, identifier, limit, at)
}

func formalProbeWorkspace(svc ratelimit.Service, workspace, identifier string, limit int64, at time.Time) (int64, error) {
	resp, err := formalTake(svc, workspace, identifier, limit, 0, at)
	if err != nil {
		return 0, err
	}
	if !resp.Success {
		return resp.Current, fmt.Errorf("a zero-cost read was denied, so the committed current %d is already above limit %d", resp.Current, limit)
	}
	return resp.Current, nil
}

func formalTake(svc ratelimit.Service, workspace, identifier string, limit, cost int64, at time.Time) (ratelimit.RatelimitResponse, error) {
	return svc.Ratelimit(context.Background(), formalRequest(workspace, identifier, limit, cost, at))
}

func formalRequest(workspace, identifier string, limit, cost int64, at time.Time) ratelimit.RatelimitRequest {
	return ratelimit.RatelimitRequest{
		WorkspaceID: workspace,
		Namespace:   formalNamespace,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    formalDuration,
		Cost:        cost,
		Time:        at,
	}
}

func formalWindowEnd(at time.Time) time.Time {
	ms := formalDuration.Milliseconds()
	start := at.UnixMilli() / ms * ms
	return time.UnixMilli(start).Add(formalDuration).UTC()
}

func formalStory(limit int64, steps []formalStep, allowed []bool, err error) error {
	text := fmt.Sprintf("identifier user, limit %d, duration 1s\n", limit)
	for i, next := range steps {
		decision := "denied"
		if i < len(allowed) && allowed[i] {
			decision = "allowed"
		}
		text += fmt.Sprintf("%d. Ratelimit cost %d at %s -> %s\n", i+1, next.cost, next.at.UTC().Format(time.RFC3339Nano), decision)
	}
	return fmt.Errorf("%s%w", text, err)
}
