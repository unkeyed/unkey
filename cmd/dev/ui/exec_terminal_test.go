package ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTerminalCommandLine(t *testing.T) {
	got := terminalCommandLine("/repo/unkey", []string{"mise", "run", "tunnel"})
	require.Equal(t, "cd '/repo/unkey' && 'mise' 'run' 'tunnel'", got)
}

func TestShellQuoteEscapesQuotes(t *testing.T) {
	require.Equal(t, `'it'\''s'`, shellQuote("it's"))
}

func TestEscapeAppleScript(t *testing.T) {
	require.Equal(t, `a\\b\"c`, escapeAppleScript(`a\b"c`))
}

func TestTunnelNeedsTTYOthersDoNot(t *testing.T) {
	p := newStackPane()
	for _, task := range p.tasks {
		if task.task == "tunnel" {
			require.True(t, task.needsTTY, "tunnel must launch in a terminal for sudo")
		} else {
			require.False(t, task.needsTTY, "%s should run as a managed proc", task.task)
		}
	}
}

func TestDropToolNoise(t *testing.T) {
	in := []string{
		"mise WARN  Failed to resolve tool version list for ruby",
		"hint: Run `mise install` without --locked to update the lockfile",
		"[tunnel] $ ~/dev/.mise/tasks/tunnel",
		"sudo: a password is required",
	}
	got := dropToolNoise(in)
	require.Equal(t, []string{
		"[tunnel] $ ~/dev/.mise/tasks/tunnel",
		"sudo: a password is required",
	}, got)
}
