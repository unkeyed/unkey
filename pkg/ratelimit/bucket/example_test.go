package bucket_test

import (
	"fmt"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/ratelimit/bucket"
)

// noonNs is 12:00:00 in the examples. Any positive time works.
var noonNs = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()

// clock formats a time of the examples as hh:mm:ss.
func clock(ns int64) string {
	return time.Unix(0, ns).UTC().Format("15:04:05")
}

// ExampleSessions replays the single-machine table of the package
// documentation: 10 units per 10 seconds on one pod.
func ExampleSessions() {
	rate := bucket.Rate{Limit: 10, Duration: 10 * time.Second}
	requests := []struct {
		at    time.Duration
		units int64
	}{
		{0, 1},
		{0, 9},
		{0, 1},
		{time.Second, 1},
		{30 * time.Second, 1},
	}

	var sessions bucket.Sessions
	for index, request := range requests {
		nowNs := noonNs + request.at.Nanoseconds()
		decision, err := sessions.Check(nowNs, rate, request.units)
		if err != nil {
			panic(err)
		}
		err = sessions.Spend(decision, "session_"+strconv.Itoa(index))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s  %d unit(s)  allowed=%-5v  reset_at=%s  remaining=%d\n",
			clock(nowNs), request.units, decision.Allowed, clock(decision.ResetAtNs), decision.Remaining)
	}
	// Output:
	// 12:00:00  1 unit(s)  allowed=true   reset_at=12:00:01  remaining=9
	// 12:00:00  9 unit(s)  allowed=true   reset_at=12:00:10  remaining=0
	// 12:00:00  1 unit(s)  allowed=false  reset_at=12:00:10  remaining=0
	// 12:00:01  1 unit(s)  allowed=true   reset_at=12:00:11  remaining=0
	// 12:00:30  1 unit(s)  allowed=true   reset_at=12:00:31  remaining=9
}

// ExampleSessions_Check_newerPeerSession shows why a request sometimes starts a
// new session. A peer started a session after this pod's live session and
// holds `reset_at`. Adding the request to the live session would not move
// `reset_at`, so the request goes into a new session instead.
func ExampleSessions_Check_newerPeerSession() {
	rate := bucket.Rate{Limit: 60, Duration: time.Minute}
	var sessions bucket.Sessions

	// This pod admits 1 unit at 12:00:00 into its live session.
	decision, err := sessions.Check(noonNs, rate, 1)
	if err != nil {
		panic(err)
	}
	err = sessions.Spend(decision, "own")
	if err != nil {
		panic(err)
	}

	// A peer started a session at 12:00:50 and used the whole limit.
	err = sessions.MergeRows([]bucket.Session{
		{ID: "peer", AnchorAtNs: noonNs + 50*time.Second.Nanoseconds(), ConsumedNs: time.Minute.Nanoseconds()},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("reset_at before:", clock(sessions.ResetAtNs()))

	nowNs := noonNs + 55*time.Second.Nanoseconds()
	decision, err = sessions.Check(nowNs, rate, 1)
	if err != nil {
		panic(err)
	}
	fmt.Println("allowed:", decision.Allowed)
	fmt.Println("reset_at after:", clock(decision.ResetAtNs))
	fmt.Println("new session anchored at:", clock(decision.CreateNewSessionAtNs))
	// Output:
	// reset_at before: 12:01:50
	// allowed: true
	// reset_at after: 12:01:51
	// new session anchored at: 12:00:55
}

// ExampleSessions_overAdmissionIsPaidBack shows two pods that each admitted
// the full limit before they saw each other. Once both rows are visible, the
// extra units are paid back before anything else fits.
func ExampleSessions_overAdmissionIsPaidBack() {
	rate := bucket.Rate{Limit: 10, Duration: 10 * time.Second}
	var sessions bucket.Sessions

	// Pod A admits all 10 units at 12:00:00.
	decision, err := sessions.Check(noonNs, rate, 10)
	if err != nil {
		panic(err)
	}
	err = sessions.Spend(decision, "pod_a")
	if err != nil {
		panic(err)
	}

	// Pod B did the same before either saw the other. Its row arrives.
	err = sessions.MergeRows([]bucket.Session{
		{ID: "pod_b", AnchorAtNs: noonNs, ConsumedNs: (10 * time.Second).Nanoseconds()},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("reset_at:", clock(sessions.ResetAtNs()))

	for index, step := range []struct {
		at    time.Duration
		units int64
	}{
		{10 * time.Second, 1},
		{10 * time.Second, 0},
		{11 * time.Second, 1},
	} {
		nowNs := noonNs + step.at.Nanoseconds()
		decision, err := sessions.Check(nowNs, rate, step.units)
		if err != nil {
			panic(err)
		}
		err = sessions.Spend(decision, "pod_a_"+strconv.Itoa(index))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s  %d unit(s)  allowed=%v\n", clock(nowNs), step.units, decision.Allowed)
	}
	// Output:
	// reset_at: 12:00:20
	// 12:00:10  1 unit(s)  allowed=false
	// 12:00:10  0 unit(s)  allowed=true
	// 12:00:11  1 unit(s)  allowed=true
}
