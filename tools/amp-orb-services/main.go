package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const commandTimeout = 30 * time.Second

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: amp-orb-services <proxy|frontline|deployment-portals> [arguments]")
	}

	var err error
	switch os.Args[1] {
	case "proxy":
		err = runTCPProxy(os.Args[2:])
	case "frontline":
		err = runFrontlineProxy(os.Args[2:])
	case "deployment-portals":
		err = runDeploymentPortals(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	return port, nil
}

func homePath(parts ...string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(parts...)
	}
	return filepath.Join(append([]string{home}, parts...)...)
}

func runKubectl(args ...string) (string, error) {
	commandArgs := append([]string{"exec", "--", "kubectl"}, args...)
	return runCommand(homePath(".local", "bin", "mise"), commandArgs...)
}

func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s failed: %s", filepath.Base(name), message)
	}
	return stdout.String(), nil
}
