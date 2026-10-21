package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkSources() error {
	manifest, err := os.ReadFile("tools/formalverify/sources.sha256")
	if err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("invalid source fingerprint: %q", line)
		}
		source, err := os.ReadFile(fields[1])
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(source)) != fields[0] {
			return fmt.Errorf("source drift: %s; review the models and their bounds before updating sources.sha256", fields[1])
		}
	}
	return nil
}

type mutation struct {
	check  string
	path   string
	before string
	after  string
}

func selfTest(ctx context.Context, out io.Writer) error {
	mutations := []mutation{
		{
			check: "deploygate", path: "pkg/deploy/deploygate/deploygate.go",
			before: "case status != dbtype.DeploymentsStatusReady:",
			after:  "case status == dbtype.DeploymentsStatusReady:",
		},
		{
			check: "circuitbreaker-recovery", path: "pkg/circuitbreaker/lib.go",
			before: "if cb.consecutiveSuccesses >= cb.config.maxRequests {",
			after:  "if cb.consecutiveSuccesses > cb.config.maxRequests {",
		},
	}
	for _, m := range mutations {
		baseline, err := subprocess(ctx, m.check, "")
		if err != nil {
			return err
		}
		if baseline.Violations != 0 {
			return fmt.Errorf("%s already fails; cannot establish a passing mutation control", m.check)
		}
		broken, err := checkMutation(ctx, m)
		if err != nil {
			return err
		}
		if broken.Cases != baseline.Cases || broken.Violations == 0 || len(broken.Counterexample) == 0 {
			return fmt.Errorf("%s mutation did not produce a semantic counterexample", m.check)
		}
		restored, err := subprocess(ctx, m.check, "")
		if err != nil {
			return err
		}
		if restored.Violations != 0 || restored.Cases != baseline.Cases {
			return fmt.Errorf("%s failed after removing the overlay", m.check)
		}
		if _, err := fmt.Fprintf(out, "%s mutation caught: %d/%d violations; unmodified source passes before and after\n", m.check, broken.Violations, broken.Cases); err != nil {
			return err
		}
	}
	return nil
}

func checkMutation(ctx context.Context, m mutation) (r result, err error) {
	path, err := filepath.Abs(m.path)
	if err != nil {
		return r, err
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if bytes.Count(source, []byte(m.before)) != 1 {
		return r, fmt.Errorf("%s mutation must match exactly once", m.path)
	}
	dir, err := os.MkdirTemp("", "unkey-formal-mutation-")
	if err != nil {
		return r, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	replacement := filepath.Join(dir, "mutated.go")
	if err := os.WriteFile(replacement, bytes.Replace(source, []byte(m.before), []byte(m.after), 1), 0o600); err != nil {
		return r, err
	}
	overlay, err := json.Marshal(struct {
		Replace map[string]string
	}{Replace: map[string]string{path: replacement}})
	if err != nil {
		return r, err
	}
	overlayPath := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0o600); err != nil {
		return r, err
	}
	return subprocess(ctx, m.check, overlayPath)
}

func subprocess(ctx context.Context, name, overlay string) (result, error) {
	args := []string{"run"}
	if overlay != "" {
		args = append(args, "-overlay="+overlay)
	}
	args = append(args, "./tools/formalverify", "--check", name, "--json")
	cmd := exec.CommandContext(ctx, "go", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, runErr := cmd.Output()
	var results []result
	if err := json.Unmarshal(output, &results); err != nil {
		return result{}, fmt.Errorf("%s produced no verification result: %w; %s", name, errors.Join(runErr, err), stderr.String())
	}
	if len(results) != 1 || results[0].Name != name || results[0].Cases == 0 {
		return result{}, fmt.Errorf("%s produced an invalid verification result", name)
	}
	r := results[0]
	var exitErr *exec.ExitError
	if runErr != nil && (r.Violations == 0 || !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 1) {
		return r, fmt.Errorf("%s failed to execute: %w; %s", name, runErr, stderr.String())
	}
	if runErr == nil && r.Violations != 0 {
		return r, fmt.Errorf("%s returned success with violations", name)
	}
	return r, nil
}
