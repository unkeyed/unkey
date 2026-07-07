package ui

import (
	"os"
	"os/exec"
	"strings"
)

func execUnkey(args ...string) (string, error) {
	bin := unkeyBin()
	var cmd *exec.Cmd
	if bin == "go" {
		cmd = exec.Command("go", append([]string{"run", "."}, args...)...)
	} else {
		cmd = exec.Command(bin, args...)
	}
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), "UNKEY_LOG_LEVEL=error")
	out, err := cmd.CombinedOutput()
	return sanitizeCmdOutput(string(out)), err
}

func sanitizeCmdOutput(raw string) string {
	var kept []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isLogfmtLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func isLogfmtLine(line string) bool {
	return strings.HasPrefix(line, "time=") && strings.Contains(line, " level=") && strings.Contains(line, " msg=")
}
