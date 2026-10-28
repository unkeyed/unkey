// Command helm-resource installs Helm releases and returns their live objects to Tilt.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type helmRelease struct {
	Manifest string `json:"manifest"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute+30*time.Second)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		// There is no other diagnostic channel if stderr itself fails.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (runErr error) {
	if len(args) < 3 || (args[0] != "apply" && args[0] != "delete") {
		return errors.New("usage: helm-resource apply RELEASE NAMESPACE CHART [FLAGS] | delete RELEASE NAMESPACE [FLAGS]")
	}
	count := 3
	if args[0] == "apply" {
		count = 4
	}
	if len(args) < count {
		return errors.New("apply requires a release, namespace, and chart")
	}
	for _, arg := range args[1:count] {
		if strings.TrimSpace(arg) == "" || strings.HasPrefix(arg, "-") {
			return errors.New("release, namespace, and chart must be nonempty positional arguments")
		}
	}
	if args[0] == "delete" {
		cmdArgs := append([]string{"uninstall", args[1], "--namespace", args[2], "--ignore-not-found"}, args[3:]...)
		cmd := exec.CommandContext(ctx, "helm", cmdArgs...)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("helm uninstall: %w", err)
		}
		return nil
	}

	helmArgs := append([]string{"upgrade", "--install", args[1], args[3], "--namespace", args[2]}, args[4:]...)
	helmArgs = append(helmArgs, "--output", "json")
	releaseJSON, err := output(ctx, nil, "helm", helmArgs...)
	if err != nil {
		return fmt.Errorf("helm upgrade: %w", err)
	}
	var release helmRelease
	if err := json.Unmarshal(releaseJSON, &release); err != nil {
		return fmt.Errorf("decode Helm release JSON: %w", err)
	}
	if strings.TrimSpace(release.Manifest) == "" {
		return errors.New("Helm release JSON has no nonempty manifest")
	}

	configJSON, err := output(ctx, nil, "kubectl", "config", "view", "--raw", "--flatten", "--minify", "-o", "json")
	if err != nil {
		return fmt.Errorf("read minified kubeconfig: %w", err)
	}
	// Keep unknown fields, including authentication configuration, intact.
	var config map[string]json.RawMessage
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return errors.New("kubectl returned invalid kubeconfig JSON")
	}
	var contexts []map[string]json.RawMessage
	if err := json.Unmarshal(config["contexts"], &contexts); err != nil || len(contexts) != 1 {
		return errors.New("minified kubeconfig must contain exactly one context")
	}
	var current string
	var name string
	var contextFields map[string]json.RawMessage
	if json.Unmarshal(config["current-context"], &current) != nil ||
		json.Unmarshal(contexts[0]["name"], &name) != nil || current == "" || current != name ||
		json.Unmarshal(contexts[0]["context"], &contextFields) != nil || contextFields == nil {
		return errors.New("minified kubeconfig has no valid current context")
	}
	contextFields["namespace"], err = json.Marshal(args[2])
	if err != nil {
		return fmt.Errorf("encode namespace: %w", err)
	}
	contexts[0]["context"], err = json.Marshal(contextFields)
	if err != nil {
		return fmt.Errorf("encode context: %w", err)
	}
	config["contexts"], err = json.Marshal(contexts)
	if err != nil {
		return fmt.Errorf("encode contexts: %w", err)
	}
	configJSON, err = json.Marshal(config)
	if err != nil {
		return errors.New("cannot encode temporary kubeconfig")
	}
	f, err := os.CreateTemp("", "helm-resource-kubeconfig-*")
	if err != nil {
		return fmt.Errorf("create temporary kubeconfig: %w", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("remove temporary kubeconfig: %w", err))
		}
	}()
	// CreateTemp uses mode 0600. Never print this file or its contents.
	if _, err := f.Write(configJSON); err != nil {
		return errors.Join(fmt.Errorf("write temporary kubeconfig: %w", err), f.Close())
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temporary kubeconfig: %w", err)
	}
	objects, err := output(ctx, []byte(release.Manifest), "kubectl", "--kubeconfig", f.Name(), "get", "-oyaml", "-f", "-")
	if err != nil {
		return fmt.Errorf("get Helm release objects: %w", err)
	}
	if _, err := os.Stdout.Write(objects); err != nil {
		return fmt.Errorf("write live objects: %w", err)
	}
	return nil
}

func output(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
