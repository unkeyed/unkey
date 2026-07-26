package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

// Managed processes run detached in their own session (setsid) with output to
// a log file and a JSON record on disk. They survive TUI exit; a later session
// reads the records back to list, tail, and stop them.

// procsDir is relative to the repo root, which is where unkey dev ui runs
// (env file loading already depends on that). .cache/ is gitignored.
const procsDir = ".cache/devui"

type procRecord struct {
	Name      string    `json:"name"`
	Pid       int       `json:"pid"`
	Args      []string  `json:"args"`
	StartedAt time.Time `json:"startedAt"`
	LogPath   string    `json:"logPath"`
}

type procStatus struct {
	procRecord
	Running bool
}

func procRecordPath(name string) string { return filepath.Join(procsDir, name+".json") }

func procLogPath(name string) string { return filepath.Join(procsDir, name+".log") }

func startManagedProc(name string, argv []string, stdin string) error {
	if st, ok := loadProcStatus(name); ok && st.Running {
		return fmt.Errorf("%s already running (pid %d)", name, st.Pid)
	}
	if err := os.MkdirAll(procsDir, 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(procLogPath(name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	// Setsid detaches the child into its own session and process group: it
	// survives TUI exit, and the negative-pid kill below reaches the whole
	// group (mise run spawns grandchildren).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} //nolint:exhaustruct
	if err := cmd.Start(); err != nil {
		return err
	}
	reapBackground(cmd)

	rec := procRecord{
		Name:      name,
		Pid:       cmd.Process.Pid,
		Args:      argv,
		StartedAt: time.Now(),
		LogPath:   procLogPath(name),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(procRecordPath(name), data, 0o644)
}

// reapBackground waits on a detached child so it does not linger as a zombie
// for the lifetime of the TUI once it exits.
func reapBackground(cmd *exec.Cmd) {
	go func() { _ = cmd.Wait() }()
}

func loadProcStatus(name string) (procStatus, bool) {
	data, err := os.ReadFile(procRecordPath(name))
	if err != nil {
		return procStatus{}, false //nolint:exhaustruct
	}
	var rec procRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return procStatus{}, false //nolint:exhaustruct
	}
	return procStatus{procRecord: rec, Running: pidAlive(rec.Pid)}, true
}

func listProcs() []procStatus {
	entries, err := os.ReadDir(procsDir)
	if err != nil {
		return nil
	}
	var out []procStatus
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		if st, ok := loadProcStatus(name); ok {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Running != out[j].Running {
			return out[i].Running
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	if err == nil {
		return true
	}
	// EPERM means the pid exists but belongs to another user.
	return errors.Is(err, syscall.EPERM)
}

func signalManagedProc(st procStatus, sig syscall.Signal) error {
	if !st.Running {
		return fmt.Errorf("%s is not running", st.Name)
	}
	// Negative pid signals the process group created by Setsid.
	if err := syscall.Kill(-st.Pid, sig); err != nil {
		return syscall.Kill(st.Pid, sig)
	}
	return nil
}

func clearExitedProcs() int {
	cleared := 0
	for _, st := range listProcs() {
		if st.Running {
			continue
		}
		if os.Remove(procRecordPath(st.Name)) == nil {
			cleared++
		}
	}
	return cleared
}

// tailLog reads up to maxLines from the end of a log file, capped to the last
// 64KiB so Tilt-sized logs never stall the UI loop.
func tailLog(path string, maxLines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	const tailBytes = 64 * 1024
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	offset := info.Size() - tailBytes
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return nil, err
	}
	buf := make([]byte, tailBytes)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(buf[:n]), "\n"), "\n")
	if offset > 0 && len(lines) > 0 {
		// The first line is almost certainly cut mid-way by the seek.
		lines = lines[1:]
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}

type procsMsg struct {
	rows []procStatus
}

func loadProcsCmd() app.Cmd {
	return func() app.Msg {
		return procsMsg{rows: listProcs()}
	}
}

type procLogMsg struct {
	name  string
	lines []string
}

func readProcLogCmd(name string) app.Cmd {
	return func() app.Msg {
		lines, err := tailLog(procLogPath(name), 400)
		if err != nil {
			lines = []string{"no log output yet"}
		}
		return procLogMsg{name: name, lines: dropToolNoise(lines)}
	}
}

// dropToolNoise strips mise's environment warnings (e.g. an unrelated ruby
// lockfile drift) so captured logs show the task's own output, not tooling
// chatter.
func dropToolNoise(lines []string) []string {
	kept := lines[:0]
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "mise WARN") || strings.HasPrefix(t, "hint: Run `mise install`") {
			continue
		}
		kept = append(kept, l)
	}
	return kept
}

func miseArgv(task string) []string {
	return []string{"mise", "run", task}
}

func unkeyArgv(subcommand ...string) []string {
	bin := unkeyBin()
	if bin == "go" {
		return append([]string{"go", "run", "."}, subcommand...)
	}
	return append([]string{bin}, subcommand...)
}
