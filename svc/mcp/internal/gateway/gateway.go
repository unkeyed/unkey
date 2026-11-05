// Package gateway is svc/mcp's HTTP surface: an MCP endpoint per toolset and
// legacy alias, OAuth protected resource metadata, install pages, and the
// legacy install redirects. Every route is derived from [tools.Toolsets].
package gateway

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/mcp/internal/caller"
	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

const wellKnownResourcePath = "/.well-known/oauth-protected-resource"

// Config is everything [New] needs. All fields are required.
type Config struct {
	// PublicURL is the origin clients connect to, such as
	// https://mcp.unkey.com. Resources and metadata URLs are built from it,
	// never from the inbound Host header.
	PublicURL string
	// APIURL receives forwarded tool calls, such as https://api.unkey.com.
	APIURL string
	// Issuer is the OAuth authorization server advertised in protected
	// resource metadata. It must equal the API's OAuth issuer.
	Issuer string
	// Version is sent to the API as User-Agent unkey-mcp/<Version>.
	Version    string
	HTTPClient *http.Client
	Clock      clock.Clock
}

// mount is one URL path that serves a toolset. Canonical and legacy paths
// are separate mounts because each is its own OAuth protected resource.
type mount struct {
	toolset     tools.Toolset
	path        string
	resource    string
	metadataURL string
	installPath string
}

type gateway struct {
	cfg      Config
	origin   string
	registry *tools.Registry
}

// New builds the handler. It fails on configuration it can detect, never at
// request time. Routes, for each toolset path P and legacy path L:
//
//	P, L                                         MCP Streamable HTTP
//	/.well-known/oauth-protected-resource{P,L}   RFC 9728 metadata ("" for "/")
//	GET {P}/install                              install page
//	GET {L}/install                              308 to {P}/install
func New(cfg Config, registry *tools.Registry) (http.Handler, error) {
	public, err := url.Parse(cfg.PublicURL)
	if err != nil || caller.Origin(cfg.PublicURL) == "" || strings.Trim(public.Path, "/") != "" {
		return nil, fmt.Errorf("public URL %q must be an origin such as https://mcp.unkey.com", cfg.PublicURL)
	}
	if caller.Origin(cfg.APIURL) == "" {
		return nil, fmt.Errorf("API URL %q must be absolute", cfg.APIURL)
	}
	if cfg.Issuer == "" || cfg.HTTPClient == nil || cfg.Clock == nil {
		return nil, errors.New("issuer, HTTP client and clock are required")
	}

	g := &gateway{cfg: cfg, origin: caller.Origin(cfg.PublicURL), registry: registry}
	mux := http.NewServeMux()
	for _, toolset := range tools.Toolsets {
		installPath := path.Join(toolset.Path, "install")
		page, err := renderInstallPage(g.origin, toolset, registry.Tools(toolset.ID))
		if err != nil {
			return nil, err
		}
		mux.Handle("GET "+installPath, installHandler(page))
		mux.Handle("GET "+path.Join(toolset.LegacyPath, "install"), http.RedirectHandler(installPath, http.StatusPermanentRedirect))

		for _, mountPath := range []string{toolset.Path, toolset.LegacyPath} {
			m := g.mount(toolset, mountPath, installPath)
			mux.Handle(wellKnownResourcePath+strings.TrimSuffix(m.path, "/"), auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
				Resource:               m.resource,
				AuthorizationServers:   []string{cfg.Issuer},
				BearerMethodsSupported: []string{"header"},
				ResourceName:           toolset.Title,
				ResourceDocumentation:  g.origin + installPath,
			}))
			handler := g.mcpHandler(m)
			if m.path == "/" {
				mux.Handle("/{$}", handler)
				continue
			}
			mux.Handle(m.path, handler)
			mux.Handle(m.path+"/{$}", handler)
		}
	}
	return mux, nil
}

func (g *gateway) mount(toolset tools.Toolset, mountPath string, installPath string) mount {
	suffix := strings.TrimSuffix(mountPath, "/")
	return mount{
		toolset:     toolset,
		path:        mountPath,
		resource:    g.origin + suffix,
		metadataURL: g.origin + wellKnownResourcePath + suffix,
		installPath: installPath,
	}
}

// mcpHandler sends browsers to the install page and gates everything else
// before the MCP SDK sees it, including initialize, so an OAuth client
// discovers sign-in when it connects rather than at its first tool call.
func (g *gateway) mcpHandler(m mount) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: m.toolset.ServerName, Title: m.toolset.Title, Version: g.cfg.Version}, nil)
	for _, tool := range g.registry.Tools(m.toolset.ID) {
		server.AddTool(&mcp.Tool{
			Name:        tool.Name,
			Title:       tool.Title,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}, g.callTool(tool))
	}
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.Redirect(w, r, m.installPath, http.StatusFound)
			return
		}
		credential, err := caller.Parse(r.Header)
		if err == nil {
			err = credential.Triage(g.cfg.Clock.Now(), g.origin)
		}
		if err != nil {
			challenge(w, m, err)
			return
		}
		streamable.ServeHTTP(w, r)
	})
}

// challenge answers 401 with the RFC 9728 pointer to the mount's protected
// resource metadata, which is how an MCP client finds where to sign in.
func challenge(w http.ResponseWriter, m mount, err error) {
	value := fmt.Sprintf("Bearer resource_metadata=%q", m.metadataURL)
	if errors.Is(err, caller.ErrUnusable) {
		value += `, error="invalid_token"`
	}
	w.Header().Set("WWW-Authenticate", value)
	http.Error(w, "Sign in to Unkey or send a root key as a Bearer token.", http.StatusUnauthorized)
}
