package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRedirectedProgressUsesPlainStderr(t *testing.T) {
	stdout := captureFile(t, &os.Stdout)
	stderr := captureFile(t, &os.Stderr)
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")

	synctest.Test(t, func(t *testing.T) {
		terminal := NewUI()
		terminal.StartSpinner("Creating deployment")
		time.Sleep(350 * time.Millisecond)
		terminal.StopSpinner("Deployment created")
	})

	require.Empty(t, stdout())
	require.Equal(t, "Creating deployment\nDeployment created\n", stderr())
}

func TestTerminalModes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		terminal bool
		term     string
		noColor  string
		animate  bool
		color    bool
	}{
		{name: "terminal", terminal: true, term: "xterm-256color", animate: true, color: true},
		{name: "no color", terminal: true, term: "xterm-256color", noColor: "1", animate: true},
		{name: "no color zero", terminal: true, term: "xterm-256color", noColor: "0", animate: true},
		{name: "dumb terminal", terminal: true, term: "dumb"},
		{name: "redirected", term: "xterm-256color"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM", tt.term)
			t.Setenv("NO_COLOR", tt.noColor)
			synctest.Test(t, func(t *testing.T) {
				var output bytes.Buffer
				terminal := newUI(&output, tt.terminal)
				terminal.StartSpinner("Creating deployment")
				time.Sleep(350 * time.Millisecond)
				terminal.StopSpinner("Deployment created")
				finished := output.String()
				time.Sleep(time.Second)
				terminal.Close()
				terminal.Close()
				require.Equal(t, finished, output.String())
				require.True(t, strings.HasSuffix(finished, "Deployment created\n"))
				if tt.animate {
					require.Contains(t, finished, "\r⠙ Creating deployment")
					require.Contains(t, finished, "\r\x1b[K")
				} else {
					require.Equal(t, "Creating deployment\nDeployment created\n", finished)
				}
				if tt.color {
					require.Contains(t, finished, "\x1b[32m✔\x1b[0m")
				} else {
					require.NotContains(t, finished, "\x1b[32m")
					require.NotContains(t, finished, "\x1b[0m")
				}
			})
		})
	}
}

func TestSpinnerReplacementAndCloseJoinBeforeReturning(t *testing.T) {
	t.Setenv("TERM", "xterm")
	t.Setenv("NO_COLOR", "1")
	synctest.Test(t, func(t *testing.T) {
		var output bytes.Buffer
		terminal := newUI(&output, true)
		terminal.StartSpinner("First phase")
		terminal.StartSpinner("Second phase")
		time.Sleep(350 * time.Millisecond)
		terminal.Close()
		finished := output.String()
		time.Sleep(time.Second)
		require.Equal(t, finished, output.String())
		require.Equal(t, 1, strings.Count(finished, "First phase"))
		require.Contains(t, finished, "\r⠙ Second phase")
		require.True(t, strings.HasSuffix(finished, "\r\x1b[K"))
	})
}

func TestStopSpinnerWaitsForInFlightFrame(t *testing.T) {
	t.Setenv("TERM", "xterm")
	synctest.Test(t, func(t *testing.T) {
		output := &frameWriter{started: make(chan struct{}), release: make(chan struct{})}
		terminal := newUI(output, true)
		terminal.StartSpinner("Creating deployment")
		<-output.started
		stopped := make(chan struct{})
		go func() {
			terminal.StopSpinner("Deployment created")
			close(stopped)
		}()
		synctest.Wait()
		select {
		case <-stopped:
			t.Error("StopSpinner returned while a frame was still being written")
		default:
		}
		close(output.release)
		<-stopped
		require.True(t, strings.HasSuffix(output.String(), "Deployment created\n"))
	})
}

func TestTextEscapesControlsAndPreservesReadableValues(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "readable", input: "café preview-eu / : ' \\", want: "café preview-eu / : ' \\"},
		{name: "terminal escape", input: "\x1b[2J\x1b]52;c;clipboard\a", want: `\x1b[2J\x1b]52;c;clipboard\a`},
		{name: "line controls", input: "\r\n\t\b\x00\x7f", want: `\r\n\t\b\x00\x7f`},
		{name: "unicode controls", input: "\u009b\u202e\u200b", want: `\u009b\u202e\u200b`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Text(tt.input))
		})
	}
}

func TestSpinnerEscapesMessages(t *testing.T) {
	var output bytes.Buffer
	terminal := newUI(&output, false)
	terminal.StartSpinner("Creating\x1b[2J\nforged")
	terminal.StopSpinner("Ready\rhidden\u202e")

	require.Equal(t, "Creating\\x1b[2J\\nforged\nReady\\rhidden\\u202e\n", output.String())
}

type frameWriter struct {
	bytes.Buffer
	started chan struct{}
	release chan struct{}
}

func (w *frameWriter) Write(data []byte) (int, error) {
	if bytes.Contains(data, []byte("⠙")) {
		close(w.started)
		<-w.release
	}
	return w.Buffer.Write(data)
}

func captureFile(t *testing.T, target **os.File) func() string {
	t.Helper()
	original := *target
	file, err := os.Create(filepath.Join(t.TempDir(), "output"))
	require.NoError(t, err)
	*target = file
	t.Cleanup(func() {
		*target = original
		require.NoError(t, file.Close())
	})
	return func() string {
		t.Helper()
		data, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		return string(data)
	}
}
