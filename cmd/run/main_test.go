package run_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	run "github.com/unkeyed/unkey/cmd/run"
)

// captureStdout redirects os.Stdout to a pipe, runs fn, then restores stdout
// and returns the captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)

	return buf.String()
}

// subcommandNames extracts the Name field from each subcommand registered
// under run.Cmd.
func subcommandNames() []string {
	names := make([]string, len(run.Cmd.Commands))
	for i, sub := range run.Cmd.Commands {
		names[i] = sub.Name
	}
	return names
}

// TestCmd_Name verifies the top-level "run" command has the correct name.
func TestCmd_Name(t *testing.T) {
	require.Equal(t, "run", run.Cmd.Name)
}

// TestCmd_UsageIsSet verifies that a non-empty usage string is configured.
func TestCmd_UsageIsSet(t *testing.T) {
	require.NotEmpty(t, run.Cmd.Usage)
}

// TestCmd_HasNoFlags verifies that the "run" command itself has no flags
// (flags belong to individual subcommands).
func TestCmd_HasNoFlags(t *testing.T) {
	require.Empty(t, run.Cmd.Flags)
}

// TestCmd_HasAction verifies that the "run" command has an action registered.
func TestCmd_HasAction(t *testing.T) {
	require.NotNil(t, run.Cmd.Action)
}

// TestCmd_HasFrontlineSubcommand verifies that "frontline" is a registered
// subcommand after the refactor that moved the policy engine into frontline.
func TestCmd_HasFrontlineSubcommand(t *testing.T) {
	names := subcommandNames()
	require.Contains(t, names, "frontline", "expected frontline subcommand to be registered")
}

// TestCmd_DoesNotHaveHeimdallSubcommand verifies that "heimdall" is no longer
// a subcommand of "run" after it was removed in this refactor.
func TestCmd_DoesNotHaveHeimdallSubcommand(t *testing.T) {
	names := subcommandNames()
	require.NotContains(t, names, "heimdall", "heimdall should have been removed as a subcommand")
}

// TestCmd_HasExpectedSubcommands verifies all expected service subcommands are
// registered: api, ctrl, krane, frontline, sentinel, vault.
func TestCmd_HasExpectedSubcommands(t *testing.T) {
	expected := []string{"api", "ctrl", "krane", "frontline", "sentinel", "vault"}
	names := subcommandNames()
	for _, svc := range expected {
		require.Contains(t, names, svc, "expected subcommand %q to be registered", svc)
	}
}

// TestRunAction_ReturnsNil verifies that invoking the "run" command with no
// subcommand executes the action and returns nil.
func TestRunAction_ReturnsNil(t *testing.T) {
	captureStdout(t, func() {
		err := run.Cmd.Run(context.Background(), []string{"run"})
		require.NoError(t, err)
	})
}

// TestRunAction_PrintsFrontline verifies that the action output mentions
// "frontline", which was added in this PR as a replacement for heimdall.
func TestRunAction_PrintsFrontline(t *testing.T) {
	output := captureStdout(t, func() {
		_ = run.Cmd.Run(context.Background(), []string{"run"})
	})
	require.Contains(t, output, "frontline", "expected output to mention frontline service")
}

// TestRunAction_DoesNotPrintHeimdall verifies that the action output no longer
// mentions "heimdall", which was removed in this refactor.
func TestRunAction_DoesNotPrintHeimdall(t *testing.T) {
	output := captureStdout(t, func() {
		_ = run.Cmd.Run(context.Background(), []string{"run"})
	})
	require.NotContains(t, output, "heimdall", "heimdall should not appear in the run action output")
}

// TestRunAction_PrintsExpectedServices verifies the action lists all known
// services (api, ctrl, krane, vault, sentinel) in its output.
func TestRunAction_PrintsExpectedServices(t *testing.T) {
	services := []string{"api", "ctrl", "krane", "vault", "sentinel"}

	output := captureStdout(t, func() {
		_ = run.Cmd.Run(context.Background(), []string{"run"})
	})

	for _, svc := range services {
		require.Contains(t, output, svc, "expected output to mention service %q", svc)
	}
}

// TestRunAction_PrintsUsageHint verifies the action output includes guidance
// on how to use the run command.
func TestRunAction_PrintsUsageHint(t *testing.T) {
	output := captureStdout(t, func() {
		_ = run.Cmd.Run(context.Background(), []string{"run"})
	})
	require.Contains(t, output, "unkey run", "expected output to include 'unkey run' usage hint")
}

// TestCmd_SubcommandCount verifies the exact number of registered subcommands
// so regressions from accidental additions or removals are caught.
func TestCmd_SubcommandCount(t *testing.T) {
	// api, ctrl, krane, frontline, sentinel, vault
	require.Len(t, run.Cmd.Commands, 6, "expected exactly 6 subcommands registered under 'run'")
}
