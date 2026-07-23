package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Response = openapi.V2WorkspacesInstallGithubResponseBody

// stateTTL bounds how long the returned install URL stays valid. It matches the
// dashboard's STATE_TTL_MS so a state minted here and verified there agree.
const stateTTL = 15 * time.Minute

type Handler struct {
	DB db.Database

	// GitHubAppName is the GitHub App slug used to build the install URL.
	// GitHubPrivateKeyPEM is the GitHub App private key used to derive the
	// install-state signing key. Either empty means GitHub connections are not
	// configured for this deployment.
	GitHubAppName       string
	GitHubPrivateKeyPEM string
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/workspaces.installGithub"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	if h.GitHubAppName == "" || h.GitHubPrivateKeyPEM == "" {
		return fault.New(
			"github not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("github app credentials are not configured for this deployment"),
			fault.Public("GitHub App installation is not enabled."),
		)
	}

	// Installing the GitHub App is a workspace-wide operation, so it is gated by
	// a workspace-level permission (workspace.*.install_github) rather than any
	// app or project scope.
	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Workspace,
		ResourceID:   "*",
		Action:       rbac.InstallGithub,
	}))
	if err != nil {
		return err
	}

	nonceBytes := make([]byte, 16)
	if _, err = rand.Read(nonceBytes); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to generate install nonce"),
			fault.Public("Failed to start the GitHub installation."),
		)
	}

	expiresAtMs := time.Now().Add(stateTTL).UnixMilli()

	state, err := newSigner(h.GitHubPrivateKeyPEM).sign(payload{
		WorkspaceID: principal.WorkspaceID,
		Nonce:       base64.RawURLEncoding.EncodeToString(nonceBytes),
		ExpMs:       expiresAtMs,
		// "api" flow: a workspace-wide install with no user binding, verified by
		// the callback's OAuth ownership proof.
		Flow: "api",
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to sign install state"),
			fault.Public("Failed to start the GitHub installation."),
		)
	}

	installURL := fmt.Sprintf(
		"https://github.com/apps/%s/installations/new?state=%s",
		h.GitHubAppName,
		url.QueryEscape(state),
	)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2WorkspacesInstallGithubResponseData{
			Url:       installURL,
			ExpiresAt: expiresAtMs,
		},
	})
}
