package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

var errViolations = errors.New("verification found counterexamples")

type result struct {
	Name           string   `json:"name"`
	Cases          int      `json:"cases"`
	Violations     int      `json:"violations"`
	Counterexample []string `json:"counterexample,omitempty"`
}

func (r *result) observe(trace []string, violation string) {
	r.Cases++
	if violation == "" {
		return
	}
	r.Violations++
	if len(r.Counterexample) == 0 {
		r.Counterexample = append(append([]string{}, trace...), violation)
	}
}

type check struct {
	name string
	run  func(context.Context) (result, error)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errViolations) {
			os.Exit(1)
		}
		os.Exit(2)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("verify-formal", flag.ContinueOnError)
	flags.SetOutput(out)
	selected := flags.String("check", "all", "all, deploygate, circuitbreaker-limit, or circuitbreaker-recovery")
	asJSON := flags.Bool("json", false, "emit structured results")
	mutations := flags.Bool("self-test", false, "require real-source mutations to produce counterexamples")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	checks := []check{
		{name: "deploygate", run: checkDeploygate},
		{name: "circuitbreaker-limit", run: checkProbeLimit},
		{name: "circuitbreaker-recovery", run: checkRecovery},
	}
	var chosen []check
	for _, c := range checks {
		if *selected == "all" || *selected == c.name {
			chosen = append(chosen, c)
		}
	}
	if len(chosen) == 0 {
		return fmt.Errorf("unknown check %q", *selected)
	}
	if err := checkSources(); err != nil {
		return err
	}
	if *mutations {
		if *selected != "all" || *asJSON {
			return errors.New("--self-test cannot be combined with --check or --json")
		}
		return selfTest(ctx, out)
	}
	var results []result
	failed := false
	for _, c := range chosen {
		r, err := c.run(ctx)
		if err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
		if r.Cases == 0 {
			return fmt.Errorf("%s explored no cases", c.name)
		}
		results = append(results, r)
		failed = failed || r.Violations > 0
	}
	if *asJSON {
		if err := json.NewEncoder(out).Encode(results); err != nil {
			return err
		}
	} else {
		for _, r := range results {
			if _, err := fmt.Fprintf(out, "%s: %d cases, %d violations\n", r.Name, r.Cases, r.Violations); err != nil {
				return err
			}
			for _, step := range r.Counterexample {
				if _, err := fmt.Fprintf(out, "  %s\n", step); err != nil {
					return err
				}
			}
		}
	}
	if failed {
		return errViolations
	}
	return nil
}
