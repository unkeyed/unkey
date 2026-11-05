package gateway

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

//go:embed install.html.tmpl
var installTemplateSource string

var installTemplate = template.Must(template.New("install").Parse(installTemplateSource))

type installPage struct {
	Title     string
	ServerURL string
	Clients   []installClient
	RootKey   []installClient
	Tools     []tools.Tool
}

// installClient is one MCP client's setup. Link is an optional one-click
// install link with a client-specific URL scheme.
type installClient struct {
	Name      string
	Steps     string
	Snippet   string
	Link      template.URL
	LinkLabel string
}

type urlServer struct {
	URL string `json:"url"`
}

type windsurfServer struct {
	ServerURL string `json:"serverUrl"`
}

type vscodeServer struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

type headerServer struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// renderInstallPage renders a toolset's install page once at startup. It is
// built from the registry, so it lists exactly what tools/list returns.
func renderInstallPage(origin string, toolset tools.Toolset, toolList []tools.Tool) ([]byte, error) {
	serverURL := origin + toolset.Path
	if toolset.Path == "/" {
		serverURL = origin
	}
	name := toolset.ServerName

	var encodeErrors []error
	encode := func(value any) string {
		encoded, err := json.Marshal(value)
		encodeErrors = append(encodeErrors, err)
		return string(encoded)
	}
	mcpServers := func(server any) string {
		encoded, err := json.MarshalIndent(map[string]map[string]any{"mcpServers": {name: server}}, "", "  ")
		encodeErrors = append(encodeErrors, err)
		return string(encoded)
	}
	cursorConfig := encode(urlServer{URL: serverURL})
	vscodeConfig := encode(vscodeServer{Name: name, Type: "http", URL: serverURL})
	cursorJSON := mcpServers(urlServer{URL: serverURL})
	windsurfJSON := mcpServers(windsurfServer{ServerURL: serverURL})
	rootKeyJSON := mcpServers(headerServer{URL: serverURL, Headers: map[string]string{"Authorization": "Bearer ${UNKEY_ROOT_KEY}"}})
	if err := errors.Join(encodeErrors...); err != nil {
		return nil, err
	}

	page := installPage{
		Title:     toolset.Title,
		ServerURL: serverURL,
		Tools:     toolList,
		Clients: []installClient{
			{
				Name:      "Claude Code",
				Steps:     "Add the server, then run /mcp in Claude Code and sign in to Unkey.",
				Snippet:   fmt.Sprintf("claude mcp add --transport http %s %s", name, serverURL),
				Link:      "",
				LinkLabel: "",
			},
			{
				Name:      "Cursor",
				Steps:     "Install with one click, or add this to ~/.cursor/mcp.json. Cursor opens the Unkey sign-in when it connects.",
				Snippet:   cursorJSON,
				Link:      template.URL("cursor://anysphere.cursor-deeplink/mcp/install?" + url.Values{"name": {name}, "config": {base64.StdEncoding.EncodeToString([]byte(cursorConfig))}}.Encode()),
				LinkLabel: "Add to Cursor",
			},
			{
				Name:      "VS Code",
				Steps:     "Install with one click, or run this command. VS Code asks you to sign in to Unkey when the server starts.",
				Snippet:   fmt.Sprintf("code --add-mcp '%s'", vscodeConfig),
				Link:      template.URL("vscode:mcp/install?" + url.QueryEscape(vscodeConfig)),
				LinkLabel: "Add to VS Code",
			},
			{
				Name:      "Claude Desktop",
				Steps:     "Open Settings, then Connectors, choose Add custom connector, and paste this URL. Claude opens the Unkey sign-in.",
				Snippet:   serverURL,
				Link:      "",
				LinkLabel: "",
			},
			{
				Name:      "Codex",
				Steps:     "Add the server, then sign in to Unkey.",
				Snippet:   fmt.Sprintf("codex mcp add %s --url %s\ncodex mcp login %s", name, serverURL, name),
				Link:      "",
				LinkLabel: "",
			},
			{
				Name:      "Gemini CLI",
				Steps:     "Add the server, then run /mcp auth " + name + " in Gemini CLI and sign in to Unkey.",
				Snippet:   fmt.Sprintf("gemini mcp add --transport http %s %s", name, serverURL),
				Link:      "",
				LinkLabel: "",
			},
			{
				Name:      "Windsurf",
				Steps:     "Add this to ~/.codeium/windsurf/mcp_config.json. Windsurf opens the Unkey sign-in when it connects.",
				Snippet:   windsurfJSON,
				Link:      "",
				LinkLabel: "",
			},
		},
		RootKey: []installClient{
			{
				Name:      "Claude Code",
				Steps:     "Pass the root key as a header instead of signing in.",
				Snippet:   fmt.Sprintf(`claude mcp add --transport http %s %s --header "Authorization: Bearer $UNKEY_ROOT_KEY"`, name, serverURL),
				Link:      "",
				LinkLabel: "",
			},
			{
				Name:      "Other clients",
				Steps:     "Most clients accept a headers map next to the server URL.",
				Snippet:   rootKeyJSON,
				Link:      "",
				LinkLabel: "",
			},
		},
	}

	var buf bytes.Buffer
	if err := installTemplate.Execute(&buf, page); err != nil {
		return nil, fmt.Errorf("render %s install page: %w", toolset.ID, err)
	}
	return buf.Bytes(), nil
}

func installHandler(page []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(page)
	})
}
