package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsCreateGithubConnectionRequestBody
	Response = openapi.V2AppsCreateGithubConnectionResponseBody
)

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
	return "/v2/apps.createGithubConnection"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if h.GitHubAppName == "" || h.GitHubPrivateKeyPEM == "" {
		return fault.New(
			"github not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("github app credentials are not configured for this deployment"),
			fault.Public("GitHub connections are not enabled."),
		)
	}

	// Validate the optional repository up front: a non-empty value that can't be
	// reduced to a legal "owner/name" is rejected here so nothing unvalidated is
	// ever signed into the state (it later lands in a GitHub API path on the
	// dashboard side).
	repository, ok := normalizeRepository(req.Repository)
	if !ok {
		return fault.New(
			"invalid repository",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("repository could not be reduced to owner/name"),
			fault.Public(`The "repository" field must be "owner/name" or a GitHub repository URL.`),
		)
	}

	// Scoped to the caller's workspace so an app in another workspace reads as
	// not-found instead of leaking that it exists.
	app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, h.DB.RO(), db.FindAppByProjectAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"app not found",
				fault.Code(codes.Data.App.NotFound.URN()),
				fault.Internal("app not found"),
				fault.Public("The requested app does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve app."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.App,
			ResourceID:   "*",
			Action:       rbac.ConnectGithubApp,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.App,
			ResourceID:   app.ID,
			Action:       rbac.ConnectGithubApp,
		}),
	))
	if err != nil {
		return fault.New(
			"app not found",
			fault.Code(codes.Data.App.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking app existence"),
			fault.Public("The requested app does not exist."),
		)
	}

	nonceBytes := make([]byte, 16)
	if _, err = rand.Read(nonceBytes); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to generate install nonce"),
			fault.Public("Failed to start the GitHub connection."),
		)
	}

	expiresAtMs := time.Now().Add(stateTTL).UnixMilli()

	state, err := newSigner(h.GitHubPrivateKeyPEM).sign(payload{
		ProjectID:   app.ProjectID,
		AppID:       app.ID,
		WorkspaceID: principal.WorkspaceID,
		Nonce:       base64.RawURLEncoding.EncodeToString(nonceBytes),
		ExpMs:       expiresAtMs,
		// Without this callback from GH ends up in project creation wizard
		ReturnTo:   "settings",
		Repository: repository,
		// This helps us identify where the request originated from.
		Source: "api",
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to sign install state"),
			fault.Public("Failed to start the GitHub connection."),
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
		Data: openapi.V2AppsCreateGithubConnectionResponseData{
			Url:       installURL,
			ExpiresAt: expiresAtMs,
		},
	})
}

// repoFullNamePattern anchors the final "owner/name" so nothing that survives
// normalization can carry path traversal, whitespace, or query/fragment
// characters into the signed state.
var repoFullNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// normalizeRepository reduces any of these accepted forms to "owner/name":
//
//	owner/name
//	https://github.com/owner/name        (any scheme case, optional www., .git, trailing path)
//	git@github.com:owner/name.git
//
// It returns (repo, true) on success, ("", true) when nothing was supplied, and
// ("", false) when a non-empty value is not one of the forms above.
func normalizeRepository(repository *string) (string, bool) {
	if repository == nil {
		return "", true
	}
	raw := strings.TrimSpace(*repository)
	if raw == "" {
		return "", true
	}

	lower := strings.ToLower(raw)
	var path string
	switch {
	case strings.HasPrefix(lower, "git@github.com:"):
		path = raw[len("git@github.com:"):]
	case strings.Contains(lower, "github.com/"):
		candidate := raw
		if !strings.Contains(lower, "://") {
			candidate = "https://" + raw
		}
		u, err := url.Parse(candidate)
		if err != nil {
			return "", false
		}
		// url.Path already excludes the query and fragment.
		if strings.TrimPrefix(strings.ToLower(u.Host), "www.") != "github.com" {
			return "", false
		}
		path = u.Path
	case strings.Contains(lower, "://"):
		return "", false
	default:
		path = raw
	}

	segments := strings.SplitN(strings.Trim(path, "/"), "/", 3)
	if len(segments) < 2 {
		return "", false
	}
	repo := strings.TrimSuffix(segments[0]+"/"+segments[1], ".git")
	if !repoFullNamePattern.MatchString(repo) {
		return "", false
	}
	return repo, true
}
