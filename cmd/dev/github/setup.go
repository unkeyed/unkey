package github

// setup.go implements the `dev github setup` subcommand.
//
// Flow (GitHub App Manifest):
//  1. Builds a GitHub App manifest (permissions, events, webhook URL, redirect URL).
//  2. Spins up a local HTTP server on :9999.
//  3. Opens the browser to localhost:9999, which auto-submits the manifest to
//     https://github.com/settings/apps/new -- GitHub presents an app-creation UI.
//  4. After the user confirms, GitHub redirects to localhost:9999/callback with a
//     one-time `code`.
//  5. Exchanges the code at POST /app-manifests/{code}/conversions to get the app's
//     ID, PEM private key, and webhook secret.
//  6. Writes credentials to dev/.env.github, dev/.github-private-key.pem, and
//     web/apps/dashboard/.env / .github-private-key.pem.

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
)

type githubAppManifest struct {
	Name               string                 `json:"name"`
	URL                string                 `json:"url"`
	Public             bool                   `json:"public"`
	HookAttributes     manifestHookAttributes `json:"hook_attributes"`
	SetupURL           string                 `json:"setup_url"`
	SetupOnUpdate      bool                   `json:"setup_on_update"`
	RedirectURL        string                 `json:"redirect_url"`
	DefaultPermissions map[string]string      `json:"default_permissions"`
	DefaultEvents      []string               `json:"default_events"`
}

type manifestHookAttributes struct {
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

var setupCmd = &cli.Command{
	Name:  "setup",
	Usage: "Create a GitHub App via manifest flow and write local dev credentials",
	Flags: []cli.Flag{
		cli.String("app-name", "GitHub App name (must be globally unique)", cli.Default("unkey-dev")),
		cli.String("webhook-url", "Initial webhook URL; Tilt's github-tunnel resource overwrites this on boot", cli.Default("https://example.com/webhooks/github")),
		cli.String("port", "Local callback server port", cli.Default("9999")),
		cli.String("out-dir", "Directory to write .env.github and .github-private-key.pem", cli.Default("dev")),
	},
	Action: setupGitHubApp,
}

// manifestResponse is the shape of the GitHub App Manifest conversion response.
// https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest
type manifestResponse struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	PEM           string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
}

// htmlPage is the page served at GET / that auto-submits the manifest form to GitHub.
var htmlPage = template.Must(template.New("setup").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Creating GitHub App...</title></head>
<body>
<p>Redirecting to GitHub to create your app...</p>
<form id="f" method="post" action="https://github.com/settings/apps/new?state={{.State}}">
  <input type="hidden" name="manifest" value="{{.Manifest}}">
</form>
<script>document.getElementById("f").submit();</script>
</body>
</html>`))

func setupGitHubApp(_ context.Context, cmd *cli.Command) error {
	appName := cmd.String("app-name")
	webhookURL := cmd.String("webhook-url")
	port := cmd.String("port")
	outDir := cmd.String("out-dir")

	callbackURL := fmt.Sprintf("http://localhost:%s/callback", port)
	listenAddr := fmt.Sprintf(":%s", port)

	manifest := githubAppManifest{
		Name:   appName,
		URL:    "http://localhost:3000",
		Public: false,
		HookAttributes: manifestHookAttributes{
			URL:    webhookURL,
			Active: true,
		},
		SetupURL:      "http://localhost:3000/integrations/github/callback",
		SetupOnUpdate: true,
		RedirectURL:   callbackURL,
		DefaultPermissions: map[string]string{
			"deployments":   "write",
			"statuses":      "write",
			"contents":      "read",
			"pull_requests": "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{"push", "pull_request"},
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}

	// codeCh receives the GitHub code from the callback handler.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if err := htmlPage.Execute(w, map[string]string{
			"State":    "unkey-dev-setup",
			"Manifest": string(manifestJSON),
		}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("GitHub did not return a code")
			return
		}
		fmt.Fprintf(w, "<html><body><p>App created! You can close this tab.</p></body></html>")
		codeCh <- code
	})

	srv := &http.Server{
		Addr:        listenAddr,
		Handler:     mux,
		ReadTimeout: 5 * time.Minute,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("local server error: %w", err)
		}
	}()

	localURL := fmt.Sprintf("http://localhost:%s", port)
	fmt.Printf("Opening browser to create your GitHub App...\n")
	fmt.Printf("If the browser does not open, visit: %s\n\n", localURL)
	openBrowser(localURL)

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		return err
	}

	_ = srv.Close()

	fmt.Println("Exchanging code for app credentials...")
	app, err := exchangeManifestCode(code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	if err := writeCredentials(outDir, app); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	fmt.Printf("✔ GitHub App %q created (ID: %d)\n", app.Name, app.ID)
	fmt.Printf("✔ Written: %s/.env.github\n", outDir)
	fmt.Printf("✔ Written: %s/.github-private-key.pem\n", outDir)
	fmt.Println("✔ Written: web/apps/dashboard/.github-private-key.pem")
	fmt.Println("✔ Written: web/apps/dashboard/.env")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. make dev")
	fmt.Println("  2. go run . dev seed local")
	fmt.Println("  3. go run . dev github trigger-webhook --project <slug> --repository <owner/repo>")

	return nil
}

// exchangeManifestCode calls the GitHub API to convert a manifest code into app credentials.
func exchangeManifestCode(code string) (*manifestResponse, error) {
	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unkey-cli")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, string(body))
	}

	var result manifestResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// writeCredentials writes app credentials to:
//   - {outDir}/.env.github          (ctrl-api / ctrl-worker / Tiltfile)
//   - {outDir}/.github-private-key.pem
//   - web/apps/dashboard/.env       (dashboard needs GITHUB_APP_ID)
//   - web/apps/dashboard/.github-private-key.pem
func writeCredentials(outDir string, app *manifestResponse) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// dev/.env.github: consumed by Tiltfile and ctrl-* services
	envContent := fmt.Sprintf(
		"UNKEY_GITHUB_APP_ID=%d\nUNKEY_GITHUB_APP_WEBHOOK_SECRET=%s\nNEXT_PUBLIC_GITHUB_APP_NAME=%q\nUNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=false\n",
		app.ID,
		app.WebhookSecret,
		app.Slug,
	)
	if err := os.WriteFile(filepath.Join(outDir, ".env.github"), []byte(envContent), 0o600); err != nil {
		return fmt.Errorf("failed to write .env.github: %w", err)
	}

	// dev/.github-private-key.pem: consumed by Tiltfile
	if err := os.WriteFile(filepath.Join(outDir, ".github-private-key.pem"), []byte(app.PEM), 0o600); err != nil {
		return fmt.Errorf("failed to write .github-private-key.pem: %w", err)
	}

	// web/apps/dashboard/: write the PEM file and append GitHub vars to .env
	dashboardDir := filepath.Join("web", "apps", "dashboard")

	if err := os.WriteFile(filepath.Join(dashboardDir, ".github-private-key.pem"), []byte(app.PEM), 0o600); err != nil {
		return fmt.Errorf("failed to write dashboard .github-private-key.pem: %w", err)
	}

	dashboardVars := fmt.Sprintf(
		"\n# GitHub App (written by `go run . dev github setup`)\nGITHUB_APP_ID=%d\nNEXT_PUBLIC_GITHUB_APP_NAME=%q\n",
		app.ID,
		app.Slug,
	)
	f, err := os.OpenFile(filepath.Join(dashboardDir, ".env"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open dashboard .env: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(dashboardVars); err != nil {
		return fmt.Errorf("failed to write dashboard .env: %w", err)
	}

	return nil
}

// openBrowser opens the given URL in the default system browser. Non-fatal on failure.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
