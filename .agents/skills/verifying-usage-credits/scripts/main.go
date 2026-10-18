package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/uid"
)

type operation struct {
	name  string
	cost  int64
	reply chan int64
}

type caller struct {
	operations chan operation
	done       chan struct{}
	response   usagelimiter.UsageResponse
	err        error
}

func main() {
	if err := checkSources(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	found, err := explore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if found {
		os.Exit(1)
	}
}

func checkSources() error {
	manifest, err := os.ReadFile(".agents/skills/verifying-usage-credits/sources.sha256")
	if err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("invalid source manifest line %q", line)
		}
		source, err := os.ReadFile(fields[1])
		if err != nil {
			return fmt.Errorf("read modeled source: %w", err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(source)) != fields[0] {
			return fmt.Errorf("STALE: review the model mapping before updating the hash for %s", fields[1])
		}
	}
	return nil
}

func explore(ctx context.Context) (bool, error) {
	found := false
	for count := 1; count <= 3; count++ {
		orders := schedules(make([]int, count), nil)
		cases, violations := 0, 0
		for initial := int64(0); initial <= 3; initial++ {
			for _, costs := range inputs(count) {
				for _, order := range orders {
					spent, trace, err := execute(ctx, initial, costs, order)
					if err != nil {
						return false, err
					}
					cases++
					if spent <= initial {
						continue
					}
					violations++
					if violations == 1 {
						fmt.Printf("COUNTEREXAMPLE callers=%d initial=%d costs=%v accepted_cost=%d\n%s\n", count, initial, costs, spent, strings.Join(trace, "\n"))
					}
				}
			}
		}
		fmt.Printf("callers=%d schedules=%d cases=%d overspend=%d\n", count, len(orders), cases, violations)
		found = found || violations > 0
	}
	return found, nil
}

func schedules(steps []int, prefix []int) [][]int {
	if len(prefix) == 2*len(steps) {
		return [][]int{append([]int(nil), prefix...)}
	}
	var result [][]int
	for actor := range steps {
		if steps[actor] == 2 {
			continue
		}
		steps[actor]++
		result = append(result, schedules(steps, append(prefix, actor))...)
		steps[actor]--
	}
	return result
}

func inputs(count int) [][]int64 {
	if count == 0 {
		return [][]int64{nil}
	}
	var result [][]int64
	for _, prefix := range inputs(count - 1) {
		for cost := int64(0); cost <= 3; cost++ {
			result = append(result, append(append([]int64(nil), prefix...), cost))
		}
	}
	return result
}

func exchange(ctx context.Context, operations chan<- operation, name string, cost int64) (int64, error) {
	op := operation{name: name, cost: cost, reply: make(chan int64)}
	select {
	case operations <- op:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	select {
	case value := <-op.reply:
		return value, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func launch(ctx context.Context, workers *sync.WaitGroup, keyID string, cost int64) (*caller, error) {
	c := &caller{
		operations: make(chan operation),
		done:       make(chan struct{}),
		response:   usagelimiter.UsageResponse{Valid: false, Remaining: 0},
		err:        nil,
	}
	service, err := usagelimiter.New(usagelimiter.Config{
		FindKeyCredits: func(ctx context.Context, _ string) (int64, bool, error) {
			value, err := exchange(ctx, c.operations, "FindKeyCredits", 0)
			return value, true, err
		},
		DecrementKeyCredits: func(ctx context.Context, _ string, cost int64) error {
			_, err := exchange(ctx, c.operations, "UpdateKeyCreditsDecrement", cost)
			return err
		},
	})
	if err != nil {
		return nil, err
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		defer close(c.done)
		c.response, c.err = service.Limit(ctx, usagelimiter.UsageRequest{KeyID: keyID, Cost: cost})
	}()
	return c, nil
}

func execute(ctx context.Context, initial int64, costs []int64, order []int) (int64, []string, error) {
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()
	keyID := uid.New(uid.KeyPrefix)
	callers := make([]*caller, len(costs))
	for actor, cost := range costs {
		c, err := launch(ctx, &workers, keyID, cost)
		if err != nil {
			return 0, nil, err
		}
		callers[actor] = c
	}
	balance := initial
	var trace []string
	for _, actor := range order {
		select {
		case op := <-callers[actor].operations:
			before := balance
			switch op.name {
			case "FindKeyCredits":
			case "UpdateKeyCreditsDecrement":
				balance = max(0, balance-op.cost)
			default:
				return 0, nil, fmt.Errorf("unmodeled operation %q", op.name)
			}
			trace = append(trace, fmt.Sprintf("  caller=%d %s cost=%d balance=%d->%d", actor, op.name, op.cost, before, balance))
			select {
			case op.reply <- balance:
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			}
		case <-callers[actor].done:
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		}
	}
	return collect(ctx, callers, costs, trace)
}

func collect(ctx context.Context, callers []*caller, costs []int64, trace []string) (int64, []string, error) {
	var spent int64
	for actor, c := range callers {
		select {
		case <-c.done:
		case op := <-c.operations:
			return 0, nil, fmt.Errorf("caller %d exceeded two-operation bound with %s", actor, op.name)
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		}
		if c.err != nil {
			return 0, nil, fmt.Errorf("caller %d: %w", actor, c.err)
		}
		if c.response.Valid {
			spent += costs[actor]
		}
		trace = append(trace, fmt.Sprintf("  caller=%d result=%+v", actor, c.response))
	}
	return spent, trace, nil
}
