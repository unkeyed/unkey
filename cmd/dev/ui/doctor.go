package ui

import (
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type doctorReport struct {
	StripeKey bool
	Docker    bool
	Minikube  bool
	DepotEnv  bool
	Messages  []string
}

// doctorCmdTimeout bounds each external doctor probe. docker info can hang
// for a long time when the daemon is down; the TUI must never wait on it.
const doctorCmdTimeout = 3 * time.Second

func runDoctor() doctorReport {
	report := doctorReport{ //nolint:exhaustruct // filled below
		Messages: nil,
	}
	key := os.Getenv("STRIPE_SECRET_KEY")
	report.StripeKey = strings.HasPrefix(key, "sk_test_") || strings.HasPrefix(key, "rk_test_")
	if !report.StripeKey {
		report.Messages = append(report.Messages, "STRIPE_SECRET_KEY missing or not test mode")
	}
	if _, err := os.Stat("dev/.env.depot"); err == nil {
		report.DepotEnv = true
	} else {
		report.Messages = append(report.Messages, "dev/.env.depot missing (run mise run bootstrap)")
	}
	if probeCommand("docker", "info") {
		report.Docker = true
	} else {
		report.Messages = append(report.Messages, "docker not reachable")
	}
	report.Minikube = probeCommand("minikube", "status")
	return report
}

func probeCommand(name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), doctorCmdTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run() == nil
}

type doctorMsg struct {
	report doctorReport
}

func runDoctorCmd() app.Cmd {
	return func() app.Msg {
		return doctorMsg{report: runDoctor()}
	}
}

// refreshInterval paces the background doctor and port re-checks so the
// header recovers on its own after e.g. Docker is started mid-session.
const refreshInterval = 5 * time.Second

type refreshTickMsg struct{}

func refreshTick() app.Cmd {
	return app.Tick(refreshInterval, func(time.Time) app.Msg {
		return refreshTickMsg{}
	})
}

func doctorSummary(r doctorReport) string {
	var parts []string
	if r.StripeKey {
		parts = append(parts, okStyle.Render("stripe"))
	} else {
		parts = append(parts, warnStyle.Render("stripe"))
	}
	if r.Docker {
		parts = append(parts, okStyle.Render("docker"))
	} else {
		parts = append(parts, warnStyle.Render("docker"))
	}
	if r.Minikube {
		parts = append(parts, okStyle.Render("minikube"))
	} else {
		parts = append(parts, dimStyle.Render("minikube"))
	}
	if r.DepotEnv {
		parts = append(parts, okStyle.Render("depot"))
	} else {
		parts = append(parts, warnStyle.Render("depot"))
	}
	return strings.Join(parts, " · ")
}

func portOpen(port string) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

type portsMsg struct {
	open map[string]bool
}

func checkPortsCmd() app.Cmd {
	return func() app.Msg {
		open := make(map[string]bool, len(stackPorts()))
		for _, p := range stackPorts() {
			open[p.Port] = portOpen(p.Port)
		}
		return portsMsg{open: open}
	}
}

func stackPorts() []struct {
	Name string
	Port string
} {
	return []struct {
		Name string
		Port string
	}{
		{"Dashboard", "3000"},
		{"API", "7070"},
		{"Ctrl API", "7091"},
		{"Tilt", "10350"},
		{"MySQL", "3306"},
	}
}
