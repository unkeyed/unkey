package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type testEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

type runResult struct {
	Command  []string            `json:"command"`
	ExitCode int                 `json:"exit_code"`
	Tests    map[string][]string `json:"tests"`
	Packages map[string]string   `json:"packages"`
}

type receipt struct {
	Revision   string      `json:"revision"`
	DiffSHA256 string      `json:"diff_sha256"`
	GoVersion  string      `json:"go_version"`
	Witness    string      `json:"witness"`
	Expected   string      `json:"expected"`
	Runs       []runResult `json:"runs"`
	Error      string      `json:"error,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("test-evidence", flag.ContinueOnError)
	flags.SetOutput(stderr)
	base := flags.String("base", "", "test packages containing tracked Go changes since this revision")
	witness := flags.String("witness", "", "exact TestName[/subtest] in one package")
	expected := flags.String("expect", "pass", "required witness outcome: pass or fail")
	seeds := flags.String("seeds", "1,2", "comma-separated replayable shuffle seeds")
	repeat := flags.Int("repeat", 1, "executions per test per seed (not retries)")
	race := flags.Bool("race", false, "enable Go race detection")
	path := flags.String("receipt", "", "also write the generated JSON receipt to this file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *repeat < 1 || (*expected != "pass" && *expected != "fail") {
		return errors.New("repeat must be positive and expect must be pass or fail")
	}
	if *expected == "fail" && *witness == "" {
		return errors.New("red evidence requires an exact witness")
	}
	if *witness != "" && !strings.HasPrefix(*witness, "Test") {
		return errors.New("witness must name a Test function")
	}

	seedValues := strings.Split(*seeds, ",")
	for _, seed := range seedValues {
		if _, err := strconv.ParseInt(seed, 10, 64); err != nil {
			return fmt.Errorf("invalid shuffle seed %q: %w", seed, err)
		}
	}

	patterns := flags.Args()
	if *base != "" {
		if len(patterns) != 0 || *witness != "" {
			return errors.New("base selects changed packages; do not combine it with patterns or a witness")
		}
		var err error
		patterns, err = changedPackages(*base)
		if err != nil {
			return err
		}
		if len(patterns) == 0 {
			_, err = fmt.Fprintln(stderr, "No changed Go packages with tests; no test evidence produced.")
			return err
		}
	}
	if len(patterns) == 0 {
		return errors.New("provide package patterns or --base")
	}

	packages, err := listPackages(patterns, stderr)
	if err != nil {
		return err
	}
	if len(packages) == 0 || (*witness != "" && len(packages) != 1) {
		return errors.New("selection must contain packages; a witness requires exactly one package")
	}

	var evidence receipt
	evidence.Witness, evidence.Expected = *witness, *expected
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("read revision: %w", err)
	}
	evidence.Revision = strings.TrimSpace(string(revision))
	diff, err := exec.Command("git", "diff", "HEAD", "--").Output()
	if err != nil {
		return fmt.Errorf("read tracked diff: %w", err)
	}
	evidence.DiffSHA256 = fmt.Sprintf("%x", sha256.Sum256(diff))
	version, err := exec.Command("go", "version").Output()
	if err != nil {
		return fmt.Errorf("read Go version: %w", err)
	}
	evidence.GoVersion = strings.TrimSpace(string(version))

	for _, seed := range seedValues {
		command := []string{
			"test",
			"-json",
			"-count=" + strconv.Itoa(*repeat),
			"-shuffle=" + seed,
		}
		if *race {
			command = append(command, "-race")
		}
		if *witness != "" {
			parts := strings.Split(*witness, "/")
			for i, part := range parts {
				parts[i] = "^" + regexp.QuoteMeta(part) + "$"
			}
			command = append(command, "-run="+strings.Join(parts, "/"))
		}
		command = append(command, packages...)
		result, runErr := execute(command, stderr)
		evidence.Runs = append(evidence.Runs, result)
		if runErr == nil {
			runErr = validate(result, packages, *witness, *expected, *repeat)
		}
		if runErr != nil {
			evidence.Error = runErr.Error()
			break
		}
	}

	encoded, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("encode receipt: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := stdout.Write(encoded); err != nil {
		return fmt.Errorf("write receipt: %w", err)
	}
	if *path != "" {
		if err := os.WriteFile(*path, encoded, 0o600); err != nil {
			return fmt.Errorf("save receipt: %w", err)
		}
	}

	if evidence.Error != "" {
		return errors.New(evidence.Error)
	}
	return nil
}

func execute(args []string, stderr io.Writer) (runResult, error) {
	result := runResult{
		Command:  append([]string{"go"}, args...),
		ExitCode: -1,
		Tests:    make(map[string][]string),
		Packages: make(map[string]string),
	}

	cmd := exec.Command("go", args...)
	var processStderr bytes.Buffer
	cmd.Stderr = &processStderr
	stream, err := cmd.StdoutPipe()
	if err != nil {
		return result, err
	}
	if err := cmd.Start(); err != nil {
		return result, err
	}

	decoder := json.NewDecoder(stream)
	var decodeErr error
	for {
		var event testEvent
		if err := decoder.Decode(&event); err != nil {
			if !errors.Is(err, io.EOF) {
				decodeErr = fmt.Errorf("decode test events: %w", err)
			}
			break
		}
		if event.Output != "" {
			if _, err := io.WriteString(stderr, event.Output); err != nil {
				decodeErr = err
				break
			}
		}
		if event.Action != "pass" && event.Action != "fail" && event.Action != "skip" {
			continue
		}
		if event.Test == "" {
			result.Packages[event.Package] = event.Action
		} else {
			key := event.Package + "/" + event.Test
			result.Tests[key] = append(result.Tests[key], event.Action)
		}
	}

	if _, err := io.Copy(io.Discard, stream); err != nil && decodeErr == nil {
		decodeErr = err
	}
	waitErr := cmd.Wait()
	if _, err := io.Copy(stderr, &processStderr); err != nil && decodeErr == nil {
		decodeErr = err
	}
	if waitErr != nil {
		var exit *exec.ExitError
		if !errors.As(waitErr, &exit) {
			return result, waitErr
		}
	}
	result.ExitCode = cmd.ProcessState.ExitCode()
	return result, decodeErr
}

func validate(result runResult, packages []string, witness, expected string, repeat int) error {
	if expected == "pass" && result.ExitCode != 0 {
		return fmt.Errorf("go test failed with exit code %d", result.ExitCode)
	}
	if expected == "pass" {
		for name, outcomes := range result.Tests {
			if slices.Contains(outcomes, "fail") {
				return fmt.Errorf("test %s failed even though go test exited successfully", name)
			}
		}
	}
	if expected == "fail" && result.ExitCode != 1 {
		return fmt.Errorf("red witness requires go test exit code 1, got %d", result.ExitCode)
	}

	for _, pkg := range packages {
		if result.Packages[pkg] != expected {
			return fmt.Errorf("package %s: expected %s, got %q", pkg, expected, result.Packages[pkg])
		}
		if witness != "" {
			outcomes := result.Tests[pkg+"/"+witness]
			if len(outcomes) != repeat {
				return fmt.Errorf("witness %s: expected %d results, got %d", witness, repeat, len(outcomes))
			}
			for _, outcome := range outcomes {
				if outcome != expected {
					return fmt.Errorf("witness %s: expected %s, got %s", witness, expected, outcome)
				}
			}
			continue
		}
		passed := false
		for name, outcomes := range result.Tests {
			if strings.HasPrefix(name, pkg+"/") && slices.Contains(outcomes, "pass") {
				passed = true
			}
		}
		if !passed {
			return fmt.Errorf("package %s: no passing tests executed", pkg)
		}
	}
	return nil
}

func listPackages(patterns []string, stderr io.Writer) ([]string, error) {
	args := append([]string{"list", "-f", "{{.ImportPath}}"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Stderr = stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	return strings.Fields(string(output)), nil
}

func changedPackages(base string) ([]string, error) {
	output, err := exec.Command("git", "diff", "--name-only", "-z", base, "--", "*.go").Output()
	if err != nil {
		return nil, fmt.Errorf("list changed Go files: %w", err)
	}

	var patterns []string
	for _, path := range bytes.Split(output, []byte{0}) {
		if len(path) == 0 {
			continue
		}
		if slices.Contains(strings.Split(filepath.ToSlash(string(path)), "/"), "testdata") {
			continue
		}
		dir := filepath.Dir(string(path))
		tests, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			return nil, err
		}
		pattern := "./" + filepath.ToSlash(dir)
		if len(tests) > 0 && !slices.Contains(patterns, pattern) {
			patterns = append(patterns, pattern)
		}
	}
	slices.Sort(patterns)
	return patterns, nil
}
