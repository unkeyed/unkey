// Package bitbucketconnect implements the POC Bitbucket Cloud OAuth connect
// flow for ctrl-api, proving the one-click UX before a proper dashboard
// integration:
//
//	GET /poc/bitbucket/connect?app_id=...  -> button page to Bitbucket authorize
//	GET /poc/bitbucket/callback            -> code exchange, repo picker page
//	GET /poc/bitbucket/select              -> register webhook, upsert connection
//
// Unlike GitLab, Bitbucket OAuth access tokens expire after two hours, so the
// stored clone credential is the REFRESH token: the deploy worker exchanges it
// for a fresh access token at build time. Repository access tokens (the
// GitLab project-access-token analog) are Premium-only, so minting is not
// attempted at all.
//
// POC shortcuts, all deliberate: unauthenticated endpoints, in-memory session
// state (single replica only), plaintext token storage, HTML rendered inline.
package bitbucketconnect

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	bitbucketwebhook "github.com/unkeyed/unkey/svc/ctrl/api/webhooks/bitbucket"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Config carries the OAuth consumer credentials and the URLs the flow hands
// to Bitbucket.
type Config struct {
	ClientID     string
	ClientSecret string
	// RedirectBaseURL is where Bitbucket sends the OAuth callback (this ctrl-api).
	RedirectBaseURL string
	// PublicWebhookURL is where Bitbucket must deliver push webhooks (the tunnel).
	PublicWebhookURL string
	// WebhookSecret is registered on the created repository hook; must match
	// the /webhooks/bitbucket verifier secret.
	WebhookSecret string
}

// New builds the /poc/bitbucket handler tree.
func New(database db.Database, cfg Config) http.Handler {
	h := &handler{
		db:        database,
		cfg:       cfg,
		bitbucket: newAPIClient(),
		mu:        sync.Mutex{},
		sessions:  make(map[string]*session),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /poc/bitbucket/connect", h.connect)
	mux.HandleFunc("GET /poc/bitbucket/callback", h.callback)
	mux.HandleFunc("GET /poc/bitbucket/select", h.selectRepo)
	return mux
}

type handler struct {
	db        db.Database
	cfg       Config
	bitbucket *apiClient

	mu       sync.Mutex
	sessions map[string]*session
}

// session is the in-flight connect state between the three requests. Tokens
// never leave the server; the browser only carries the state nonce.
type session struct {
	AppID       string
	AccessToken string
	// RefreshToken is what gets persisted as the clone credential: access
	// tokens live two hours, refresh tokens are long-lived.
	RefreshToken string
	Repos        []repository
	ExpiresAt    time.Time
}

const sessionTTL = 15 * time.Minute

// connect validates the target app and renders a button to Bitbucket's
// authorize page. Bitbucket OAuth has no PKCE support; the consumer is
// confidential (private) so the code exchange is secret-bound.
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
	h.putSession(state, &session{
		AppID:        appID,
		AccessToken:  "",
		RefreshToken: "",
		Repos:        nil,
		ExpiresAt:    time.Now().Add(sessionTTL),
	})

	authorize := "https://bitbucket.org/site/oauth2/authorize?" + url.Values{
		"client_id":     {h.cfg.ClientID},
		"response_type": {"code"},
		"state":         {state},
	}.Encode()

	// Deliberately NOT a redirect: browsers prerender URL-bar suggestions and
	// will otherwise run the whole OAuth dance speculatively, consuming the
	// single-use authorization code before the user ever presses enter. A
	// human click is the prerender barrier. (Learned the hard way on GitLab.)
	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Connect Bitbucket</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connect Bitbucket to app %s</h2>", html.EscapeString(appID))
	fmt.Fprintf(w, `<p><a href="%s" style="display:inline-block;padding:.6rem 1.2rem;background:#0052cc;color:#fff;border-radius:6px;text-decoration:none">Continue to Bitbucket</a></p>`, html.EscapeString(authorize))
}

// callback exchanges the code for tokens and renders the repository picker.
// Only repositories with admin access are listed: webhook registration
// requires it.
func (h *handler) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	sess := h.getSession(state)
	if sess == nil {
		http.Error(w, "unknown or expired state; restart at /poc/bitbucket/connect", http.StatusBadRequest)
		return
	}

	// Browsers may prefetch the callback URL, consuming the single-use code
	// before the real navigation arrives. The first hit stores the tokens on
	// the session; later hits render from it instead of failing the exchange
	// with invalid_grant. The workspace form below also re-enters here with
	// the token already stored.
	if sess.AccessToken == "" {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code query parameter", http.StatusBadRequest)
			return
		}

		token, err := h.bitbucket.exchangeCode(r.Context(), h.cfg.ClientID, h.cfg.ClientSecret, code)
		if err != nil {
			logger.Error("bitbucket code exchange failed", "error", err)
			http.Error(w, "code exchange failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		sess.AccessToken = token.AccessToken
		sess.RefreshToken = token.RefreshToken
	}

	// CHANGE-2770 removed cross-workspace discovery, so repositories are
	// listed per workspace: either the one the user typed into the form, or
	// every workspace bare /2.0/workspaces still reveals. When discovery is
	// gone too, degrade to asking for the slug.
	if workspace := r.URL.Query().Get("workspace"); workspace != "" {
		repos, err := h.bitbucket.listAdminRepositoriesInWorkspace(r.Context(), sess.AccessToken, workspace)
		if err != nil {
			logger.Error("bitbucket repository listing failed", "workspace", workspace, "error", err)
			http.Error(w, "repository listing failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		sess.Repos = repos
	} else {
		slugs, err := h.bitbucket.listWorkspaceSlugs(r.Context(), sess.AccessToken)
		if err != nil {
			logger.Warn("bitbucket workspace discovery unavailable, asking for slug", "error", err)
			fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Pick a workspace</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
			fmt.Fprintf(w, "<h2>Connect a Bitbucket repository to app %s</h2>", html.EscapeString(sess.AppID))
			fmt.Fprint(w, "<p>Bitbucket removed workspace discovery (CHANGE-2770). Enter your workspace slug:</p>")
			fmt.Fprintf(w, `<form method="get" action="/poc/bitbucket/callback"><input type="hidden" name="state" value="%s"><input name="workspace" placeholder="workspace-slug" autofocus> <button>List repositories</button></form>`, html.EscapeString(state))
			return
		}
		var repos []repository
		for _, slug := range slugs {
			wsRepos, err := h.bitbucket.listAdminRepositoriesInWorkspace(r.Context(), sess.AccessToken, slug)
			if err != nil {
				logger.Error("bitbucket repository listing failed", "workspace", slug, "error", err)
				http.Error(w, "repository listing failed: "+err.Error(), http.StatusBadGateway)
				return
			}
			repos = append(repos, wsRepos...)
		}
		sess.Repos = repos
	}

	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Pick a repository</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connect a Bitbucket repository to app %s</h2>", html.EscapeString(sess.AppID))
	if len(sess.Repos) == 0 {
		fmt.Fprint(w, "<p>No repositories with admin access found.</p>")
		return
	}
	fmt.Fprint(w, "<ul>")
	for i, repo := range sess.Repos {
		link := "/poc/bitbucket/select?" + url.Values{
			"state": {state},
			"repo":  {strconv.Itoa(i)},
		}.Encode()
		fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, link, html.EscapeString(repo.FullName))
	}
	fmt.Fprint(w, "</ul>")
}

// selectRepo finishes the connect: webhook registration and connection row.
// The stored clone credential is the refresh token; see the package comment.
func (h *handler) selectRepo(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	sess := h.getSession(state)
	if sess == nil || sess.AccessToken == "" {
		http.Error(w, "unknown or expired state; restart at /poc/bitbucket/connect", http.StatusBadRequest)
		return
	}

	repoIndex, err := strconv.Atoi(r.URL.Query().Get("repo"))
	if err != nil || repoIndex < 0 || repoIndex >= len(sess.Repos) {
		http.Error(w, "repository was not offered in this session", http.StatusBadRequest)
		return
	}
	picked := sess.Repos[repoIndex]

	if sess.RefreshToken == "" {
		// Without a refresh token the connection dies with the 2h access token.
		// The consumer must have the "Refresh token" grant type enabled.
		http.Error(w, "bitbucket issued no refresh token; enable the refresh_token grant on the OAuth consumer", http.StatusBadGateway)
		return
	}

	hookURL := h.cfg.PublicWebhookURL + "/webhooks/bitbucket"
	if err := h.bitbucket.ensurePushWebhook(r.Context(), sess.AccessToken, picked.FullName, hookURL, h.cfg.WebhookSecret); err != nil {
		logger.Error("bitbucket webhook registration failed", "repository", picked.FullName, "error", err)
		http.Error(w, "webhook registration failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	app, err := h.db.FindAppById(r.Context(), sess.AppID)
	if err != nil {
		http.Error(w, "failed to look up app", http.StatusInternalServerError)
		return
	}
	repoID := bitbucketwebhook.RepoNumericID(picked.UUID)
	err = h.db.UpsertRepoConnection(r.Context(), db.UpsertRepoConnectionParams{
		WorkspaceID:        app.WorkspaceID,
		ProjectID:          app.ProjectID,
		AppID:              app.ID,
		InstallationID:     repoID,
		RepositoryID:       repoID,
		RepositoryFullName: picked.FullName,
		Provider:           "bitbucket",
		AccessToken:        sql.NullString{String: sess.RefreshToken, Valid: true},
		CreatedAt:          time.Now().UnixMilli(),
	})
	if err != nil {
		logger.Error("bitbucket connection upsert failed", "app_id", sess.AppID, "error", err)
		http.Error(w, "failed to store connection", http.StatusInternalServerError)
		return
	}

	h.dropSession(state)

	logger.Info("bitbucket connection established",
		"app_id", sess.AppID,
		"repository", picked.FullName,
		"repository_uuid", picked.UUID,
		"repository_numeric_id", repoID,
	)

	fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><title>Connected</title><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">`)
	fmt.Fprintf(w, "<h2>Connected</h2><p><b>%s</b> is now wired to app <b>%s</b>.</p>",
		html.EscapeString(picked.FullName), html.EscapeString(sess.AppID))
	fmt.Fprintf(w, "<p>Clone credential: refresh token (exchanged at build time). Webhook registered at %s.</p><p>Push to deploy.</p>",
		html.EscapeString(hookURL))
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
