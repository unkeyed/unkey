package gitlabconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const gitlabBaseURL = "https://gitlab.com"

// apiClient is the minimal GitLab REST surface the connect flow needs.
type apiClient struct {
	http *http.Client
}

func newAPIClient() *apiClient {
	return &apiClient{http: &http.Client{Timeout: 15 * time.Second}}
}

type project struct {
	ID                int64  `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}

// exchangeCode trades the OAuth authorization code for an access token.
func (c *apiClient) exchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI, pkceVerifier string) (string, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gitlabBaseURL+"/oauth/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("token response carried no access_token")
	}
	return out.AccessToken, nil
}

// listMaintainerProjects returns the caller's projects with maintainer access
// (level 40) — the minimum for webhook registration and token minting.
func (c *apiClient) listMaintainerProjects(ctx context.Context, token string) ([]project, error) {
	u := gitlabBaseURL + "/api/v4/projects?" + url.Values{
		"membership":       {"true"},
		"min_access_level": {"40"},
		"simple":           {"true"},
		"per_page":         {"100"},
		"order_by":         {"last_activity_at"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var out []project
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// createProjectAccessToken mints a read-only clone credential owned by the
// project, not the connecting user. Returns the token and a human-readable
// kind label for the success page.
func (c *apiClient) createProjectAccessToken(ctx context.Context, token string, projectID int64) (string, string, error) {
	// GitLab caps project access token lifetime at one year.
	expires := time.Now().AddDate(0, 11, 0).Format("2006-01-02")
	body, err := json.Marshal(map[string]any{
		"name":         "unkey-deploy",
		"scopes":       []string{"read_repository"},
		"access_level": 30,
		"expires_at":   expires,
	})
	if err != nil {
		return "", "", err
	}
	u := fmt.Sprintf("%s/api/v4/projects/%d/access_tokens", gitlabBaseURL, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	var out struct {
		Token string `json:"token"`
	}
	if err := c.do(req, http.StatusCreated, &out); err != nil {
		return "", "", err
	}
	if out.Token == "" {
		return "", "", fmt.Errorf("access token response carried no token")
	}
	return out.Token, "project access token (expires " + expires + ")", nil
}

// ensurePushWebhook registers the push webhook on the project, skipping
// creation when a hook with the same URL already exists so reconnecting an
// app does not stack duplicate hooks.
func (c *apiClient) ensurePushWebhook(ctx context.Context, token string, projectID int64, hookURL, secret string) error {
	listURL := fmt.Sprintf("%s/api/v4/projects/%d/hooks", gitlabBaseURL, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var hooks []struct {
		URL string `json:"url"`
	}
	if err := c.do(req, http.StatusOK, &hooks); err != nil {
		return fmt.Errorf("list hooks: %w", err)
	}
	for _, hook := range hooks {
		if hook.URL == hookURL {
			return nil
		}
	}

	body, err := json.Marshal(map[string]any{
		"url":         hookURL,
		"push_events": true,
		"token":       secret,
	})
	if err != nil {
		return err
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, listURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	if err := c.do(req, http.StatusCreated, nil); err != nil {
		return fmt.Errorf("create hook: %w", err)
	}
	return nil
}

// do executes the request, enforces the expected status, and decodes JSON
// into out when non-nil. Error bodies are truncated into the error message.
func (c *apiClient) do(req *http.Request, wantStatus int, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("gitlab returned %s: %s", strconv.Itoa(resp.StatusCode), string(snippet))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
