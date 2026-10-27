package cli

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// OpenBrowser opens rawURL in the default system browser without waiting for
// it to exit. Only absolute http and https URLs are accepted, because the
// platform openers also launch local files and other registered handlers.
func OpenBrowser(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("refusing to open non-http(s) url %q", rawURL)
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start browser: %w", err)
	}
	go func() { _ = command.Wait() }()
	return nil
}
