package ui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}

func openTilt() {
	openURL("http://localhost:10350")
}

// runInTerminal launches argv in a new terminal window from the current
// directory. Used for tasks that need an interactive tty (sudo password
// prompts), which a detached managed process cannot provide.
func runInTerminal(argv []string) error {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	line := terminalCommandLine(dir, argv)
	switch runtime.GOOS {
	case "darwin":
		script := "tell application \"Terminal\"\n\tactivate\n\tdo script \"" + escapeAppleScript(line) + "\"\nend tell"
		return exec.Command("osascript", "-e", script).Start()
	case "linux":
		return exec.Command("x-terminal-emulator", "-e", "bash", "-lc", line).Start()
	default:
		return exec.Command("false").Run()
	}
}

// terminalCommandLine builds the shell line a new terminal runs: cd into the
// working dir, then the quoted command.
func terminalCommandLine(dir string, argv []string) string {
	parts := make([]string, 0, len(argv)+2)
	parts = append(parts, "cd", shellQuote(dir), "&&")
	for _, a := range argv {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

func unkeyBin() string {
	if _, err := exec.LookPath("./bin/unkey"); err == nil {
		return "./bin/unkey"
	}
	bin, err := exec.LookPath("unkey")
	if err == nil {
		return bin
	}
	return "go"
}
