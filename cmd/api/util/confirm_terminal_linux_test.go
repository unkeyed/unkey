package util

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/sys/unix"
)

func TestConfirmCallbackPreservesSafetyChecks(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		stdinFile  bool
		stderrFile bool
		wantError  string
		wantCalls  int
	}{
		{name: "yesStillReviewsGuidedInput", args: []string{"--yes"}, wantCalls: 1},
		{name: "yesDoesNotBypassGuidedSafeguards", args: []string{"--yes", "--no-input"}, stdinFile: true, stderrFile: true, wantError: "confirmation is disabled by --no-input"},
		{name: "noInput", args: []string{"--no-input"}, wantError: "confirmation is disabled by --no-input"},
		{name: "stdinNotTerminal", stdinFile: true, wantError: "confirmation requires a terminal"},
		{name: "stderrNotTerminal", stderrFile: true, wantError: "confirmation requires a terminal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			confirmationTerminal(t)
			file, err := os.CreateTemp(t.TempDir(), "nonterminal")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, file.Close()) })
			if test.stdinFile {
				os.Stdin = file
			}
			if test.stderrFile {
				os.Stderr = file
			}
			calls := 0
			ctx := WithConfirmation(t.Context(), func(string) error {
				calls++
				return nil
			})
			err = runConfirmation(ctx, t, "Delete\n\x1b[2J?", test.args...)
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
				require.ErrorContains(t, err, `Delete\n\x1b[2J?`)
			}
			require.Equal(t, test.wantCalls, calls)
		})
	}
}

func TestConfirmCallbackReceivesSanitizedWarningAndReturnsDecision(t *testing.T) {
	edit := errors.New("edit requested")
	for _, test := range []struct {
		name     string
		decision error
	}{
		{name: "confirmed"},
		{name: "edit", decision: edit},
		{name: "cancelled", decision: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			confirmationTerminal(t)
			var questions []string
			ctx := WithConfirmation(t.Context(), func(question string) error {
				questions = append(questions, question)
				return test.decision
			})
			err := runConfirmation(ctx, t, "Permanently delete\n\x1b[2J? This cannot be undone.")
			require.ErrorIs(t, err, test.decision)
			require.Equal(t, []string{`Permanently delete\n\x1b[2J? This cannot be undone.`}, questions)
		})
	}
}

func TestConfirmWithoutCallbackRequiresExplicitConsent(t *testing.T) {
	for _, test := range []struct {
		name   string
		answer string
		allow  bool
	}{
		{name: "yes", answer: "yes\n", allow: true},
		{name: "uppercase", answer: "Y\n", allow: true},
		{name: "default", answer: "\n"},
		{name: "no", answer: "no\n"},
		{name: "EOF", answer: "\x04"},
	} {
		t.Run(test.name, func(t *testing.T) {
			master := confirmationTerminal(t)
			_, err := master.WriteString(test.answer)
			require.NoError(t, err)
			err = runConfirmation(t.Context(), t, "Delete resource?")
			if test.allow {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, "cancelled")
			}
		})
	}
}

func runConfirmation(ctx context.Context, t *testing.T, question string, args ...string) error {
	t.Helper()
	cmd := &cli.Command{
		Name: "test", Flags: []cli.Flag{YesFlag(), cli.Bool("no-input", "Disable prompts")},
		Action: func(ctx context.Context, cmd *cli.Command) error { return Confirm(ctx, cmd, question) },
	}
	return cmd.Run(ctx, append([]string{"test"}, args...))
}

func confirmationTerminal(t *testing.T) *os.File {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, master.Close()) })
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(number), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, slave.Close()) })
	stdin, stderr := os.Stdin, os.Stderr
	os.Stdin, os.Stderr = slave, slave
	t.Cleanup(func() { os.Stdin, os.Stderr = stdin, stderr })
	return master
}
