package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	sdkoauth "github.com/modelcontextprotocol/go-sdk/oauthex"
)

type whoamiArgs struct{}

// NewHandler serves both spike resources. Each path has its own MCP server,
// protected-resource metadata, and exact audience check.
func NewHandler(cfg Config, logger *slog.Logger) (http.Handler, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}

	verifier := newVerifier(cfg, logger)
	mux := http.NewServeMux()
	for _, name := range []string{ResourceAPI, ResourceCompute} {
		resourceURL := cfg.ResourceURL(name)
		metadataURL := cfg.MetadataURL(name)
		mux.Handle(
			"/.well-known/oauth-protected-resource/"+name,
			sdkauth.ProtectedResourceMetadataHandler(protectedResourceMetadata(cfg.Issuer, resourceURL)),
		)
		mcpServer := newMCPServer()
		stream := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
			return mcpServer
		}, nil)
		protected := sdkauth.RequireBearerToken(verifier.tokenVerifier(resourceURL), &sdkauth.RequireBearerTokenOptions{
			ResourceMetadataURL:    metadataURL,
			Scopes:                 nil,
			AllowMissingExpiration: false,
			ClockSkew:              0,
		})
		mux.Handle("/"+name, protected(stream))
	}
	return mux, nil
}

func protectedResourceMetadata(issuer string, resourceURL string) *sdkoauth.ProtectedResourceMetadata {
	return &sdkoauth.ProtectedResourceMetadata{
		Resource:                              resourceURL,
		AuthorizationServers:                  []string{issuer},
		JWKSURI:                               "",
		ScopesSupported:                       nil,
		BearerMethodsSupported:                []string{"header"},
		ResourceSigningAlgValuesSupported:     nil,
		ResourceName:                          "",
		ResourceDocumentation:                 "",
		ResourcePolicyURI:                     "",
		ResourceTOSURI:                        "",
		TLSClientCertificateBoundAccessTokens: false,
		AuthorizationDetailsTypesSupported:    nil,
		DPOPSigningAlgValuesSupported:         nil,
		DPOPBoundAccessTokensRequired:         false,
	}
}

func newMCPServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:        "unkey-mcp-spike",
		Title:       "",
		Description: "WorkOS AuthKit token spike",
		Version:     "0.0.0",
		WebsiteURL:  "",
		Icons:       nil,
	}, nil)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Meta:         sdkmcp.Meta{},
		Annotations:  nil,
		Description:  "Return the verified access-token claims for this resource, including exp minus iat. The bearer token is not included.",
		InputSchema:  nil,
		Name:         "whoami",
		OutputSchema: nil,
		Title:        "",
		Icons:        nil,
	}, whoami)
	return server
}

func whoami(ctx context.Context, _ *sdkmcp.CallToolRequest, _ whoamiArgs) (*sdkmcp.CallToolResult, Identity, error) {
	info := sdkauth.TokenInfoFromContext(ctx)
	if info == nil {
		return nil, zeroIdentity(), errors.New("verified token is not on the request")
	}
	identity, ok := info.Extra["identity"].(Identity)
	if !ok {
		return nil, zeroIdentity(), errors.New("verified token is not on the request")
	}
	return nil, identity, nil
}
