// Package gitlabconnect implements the POC GitLab OAuth connect flow for
// ctrl-api, proving the one-click UX before a proper dashboard integration:
//
//	GET /poc/gitlab/connect?app_id=...  -> redirect to GitLab authorize
//	GET /poc/gitlab/callback            -> code exchange, project picker page
//	GET /poc/gitlab/select              -> mint clone token, register webhook,
//	                                       upsert connection row
//
// The OAuth token is used only during the flow. The stored clone credential is
// a project access token minted via the API (no refresh lifecycle); if minting
// is unavailable (group policy, tier) the flow falls back to storing the OAuth
// token itself, which doubles as the bearer-clone experiment.
//
// POC shortcuts, all deliberate: unauthenticated endpoints, in-memory session
// state (single replica only), plaintext token storage, HTML rendered inline.
package gitlabconnect

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Config carries the OAuth application credentials and the URLs the flow
// hands to GitLab.
type Config struct {
	ClientID     string
	ClientSecret string
	// RedirectBaseURL is where GitLab sends the OAuth callback (this ctrl-api).
	RedirectBaseURL string
	// PublicWebhookURL is where GitLab must deliver push webhooks (the tunnel).
	PublicWebhookURL string
	// WebhookSecret is registered on the created project hook; must match the
	// /webhooks/gitlab verifier secret.
	WebhookSecret string
}

// New builds the /poc/gitlab handler tree.
func New(database db.Database, cfg Config) http.Handler {
	h := &handler{
		db:       database,
		cfg:      cfg,
		gitlab:   newAPIClient(),
		mu:       sync.Mutex{},
		sessions: make(map[string]*session),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /poc/gitlab/connect", h.connect)
	mux.HandleFunc("GET /poc/gitlab/callback", h.callback)
	mux.HandleFunc("GET /poc/gitlab/select", h.selectProject)
	return mux
}

type handler struct {
	db     db.Database
	cfg    Config
	gitlab *apiClient

	mu       sync.Mutex
	sessions map[string]*session
}

// session is the in-flight connect state between the three requests. The
// OAuth token never leaves the server; the browser only carries the state
// nonce.
type session struct {
	AppID       string
	AccessToken string
	Projects    []project
	// PKCEVerifier is the code_verifier whose S256 challenge went into the
	// authorize request. PKCE is mandatory for public OAuth apps (Confidential
	// unchecked) and harmless for confidential ones.
	PKCEVerifier string
	ExpiresAt    time.Time
}

const sessionTTL = 15 * time.Minute

// connect validates the target app and redirects to GitLab's authorize page.
func (h *handler) connect(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		http.Error(w, "missing app_id query parameter", http.StatusBadRequest)
		return
	}

	if _, err := h.db.FindAppById(r.Context(), appID); err != nil {
		if db.IsNotFound(err) {
			http.Error(w, fmt.Sprintf("app %s not found", appID), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to look up app", http.StatusInternalServerError)
		return
	}

	state := newNonce()
	verifier := newNonce() + newNonce()
	h.putSession(state, &session{
		AppID:        appID,
		AccessToken:  "",
		Projects:     nil,
		PKCEVerifier: verifier,
		ExpiresAt:    time.Now().Add(sessionTTL),
	})

	challenge := sha256.Sum256([]byte(verifier))
	authorize := "https://gitlab.com/oauth/authorize?" + url.Values{
		"client_id":             {h.cfg.ClientID},
		"redirect_uri":          {h.redirectURI()},
		"response_type":         {"code"},
		"state":                 {state},
		"scope":                 {"api"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}.Encode()

	// Deliberately NOT a redirect: browsers prerender URL-bar suggestions and
	// will otherwise run the whole OAuth dance speculatively, consuming the
	// single-use authorization code before the user ever presses enter. A
	// human click is the prerender barrier.
	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Connect GitLab</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connect GitLab to app %s</h2>", html.EscapeString(appID))
	fmt.Fprintf(w, `<p><a href="%s" style="display:inline-block;padding:.6rem 1.2rem;background:#171321;color:#fff;border-radius:6px;text-decoration:none">Continue to GitLab</a></p>`, html.EscapeString(authorize))
}

// callback exchanges the code for a token and renders the project picker.
// Only projects with maintainer access are listed: webhook registration and
// access token minting both require it.
func (h *handler) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	sess := h.getSession(state)
	if sess == nil {
		http.Error(w, "unknown or expired state; restart at /poc/gitlab/connect", http.StatusBadRequest)
		return
	}

	// Browsers may prefetch the callback URL, consuming the single-use code
	// before the real navigation arrives. The first hit stores the token on
	// the session; later hits render the picker from it instead of failing
	// the exchange with invalid_grant.
	if sess.AccessToken == "" {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code query parameter", http.StatusBadRequest)
			return
		}

		token, err := h.gitlab.exchangeCode(r.Context(), h.cfg.ClientID, h.cfg.ClientSecret, code, h.redirectURI(), sess.PKCEVerifier)
		if err != nil {
			logger.Error("gitlab code exchange failed", "error", err)
			http.Error(w, "code exchange failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		projects, err := h.gitlab.listMaintainerProjects(r.Context(), token)
		if err != nil {
			logger.Error("gitlab project listing failed", "error", err)
			http.Error(w, "project listing failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		sess.AccessToken = token
		sess.Projects = projects
	}

	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Pick a project</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connect a GitLab project to app %s</h2>", html.EscapeString(sess.AppID))
	if len(sess.Projects) == 0 {
		fmt.Fprint(w, "<p>No projects with maintainer access found.</p>")
		return
	}
	fmt.Fprint(w, "<ul>")
	for _, p := range sess.Projects {
		link := "/poc/gitlab/select?" + url.Values{
			"state":      {state},
			"project_id": {strconv.FormatInt(p.ID, 10)},
		}.Encode()
		fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, link, html.EscapeString(p.PathWithNamespace))
	}
	fmt.Fprint(w, "</ul>")
}

// selectProject finishes the connect: clone credential, webhook, connection row.
func (h *handler) selectProject(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	sess := h.getSession(state)
	if sess == nil || sess.AccessToken == "" {
		http.Error(w, "unknown or expired state; restart at /poc/gitlab/connect", http.StatusBadRequest)
		return
	}

	projectID, err := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	var picked *project
	for i := range sess.Projects {
		if sess.Projects[i].ID == projectID {
			picked = &sess.Projects[i]
			break
		}
	}
	if picked == nil {
		http.Error(w, "project was not offered in this session", http.StatusBadRequest)
		return
	}

	// Prefer a minted project access token: PAT-semantics clone (proven path),
	// no OAuth refresh lifecycle. Fall back to the OAuth token where minting
	// is unavailable so the flow still completes.
	cloneToken, tokenKind, err := h.gitlab.createProjectAccessToken(r.Context(), sess.AccessToken, projectID)
	if err != nil {
		logger.Warn("project access token minting failed, storing OAuth token instead",
			"project_id", projectID, "error", err)
		cloneToken, tokenKind = sess.AccessToken, "oauth (fallback: "+err.Error()+")"
	}

	hookURL := h.cfg.PublicWebhookURL + "/webhooks/gitlab"
	if err := h.gitlab.ensurePushWebhook(r.Context(), sess.AccessToken, projectID, hookURL, h.cfg.WebhookSecret); err != nil {
		logger.Error("gitlab webhook registration failed", "project_id", projectID, "error", err)
		http.Error(w, "webhook registration failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	app, err := h.db.FindAppById(r.Context(), sess.AppID)
	if err != nil {
		http.Error(w, "failed to look up app", http.StatusInternalServerError)
		return
	}
	err = h.db.UpsertGitlabRepoConnection(r.Context(), db.UpsertGitlabRepoConnectionParams{
		WorkspaceID:        app.WorkspaceID,
		ProjectID:          app.ProjectID,
		AppID:              app.ID,
		InstallationID:     projectID,
		RepositoryID:       projectID,
		RepositoryFullName: picked.PathWithNamespace,
		AccessToken:        sql.NullString{String: cloneToken, Valid: cloneToken != ""},
		CreatedAt:          time.Now().UnixMilli(),
	})
	if err != nil {
		logger.Error("gitlab connection upsert failed", "app_id", sess.AppID, "error", err)
		http.Error(w, "failed to store connection", http.StatusInternalServerError)
		return
	}

	h.dropSession(state)

	logger.Info("gitlab connection established",
		"app_id", sess.AppID,
		"gitlab_project_id", projectID,
		"repository", picked.PathWithNamespace,
		"clone_token_kind", tokenKind,
	)

	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Connected</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connected</h2><p><b>%s</b> is now wired to app <b>%s</b>.</p>",
		html.EscapeString(picked.PathWithNamespace), html.EscapeString(sess.AppID))
	fmt.Fprintf(w, "<p>Clone credential: %s. Webhook registered at %s.</p><p>Push to deploy.</p>",
		html.EscapeString(tokenKind), html.EscapeString(hookURL))
}

func (h *handler) redirectURI() string {
	return h.cfg.RedirectBaseURL + "/poc/gitlab/callback"
}

func (h *handler) putSession(state string, s *session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Opportunistic sweep keeps the map from accumulating abandoned flows.
	for k, v := range h.sessions {
		if time.Now().After(v.ExpiresAt) {
			delete(h.sessions, k)
		}
	}
	h.sessions[state] = s
}

func (h *handler) getSession(state string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[state]
	if !ok || time.Now().After(s.ExpiresAt) {
		return nil
	}
	return s
}

func (h *handler) dropSession(state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, state)
}

func newNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	return hex.EncodeToString(b)
}
