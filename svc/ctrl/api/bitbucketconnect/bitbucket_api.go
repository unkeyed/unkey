package bitbucketconnect

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

const (
	// bitbucketBaseURL hosts the OAuth endpoints (/site/oauth2/*).
	bitbucketBaseURL = "https://bitbucket.org"
	// bitbucketAPIBaseURL hosts the REST API (separate domain, unlike GitLab).
	bitbucketAPIBaseURL = "https://api.bitbucket.org"
)

// apiClient is the minimal Bitbucket REST surface the connect flow needs.
type apiClient struct {
	http *http.Client
}

func newAPIClient() *apiClient {
	return &apiClient{http: &http.Client{Timeout: 15 * time.Second}}
}

type repository struct {
	UUID     string `json:"uuid"`
	FullName string `json:"full_name"`
}

// tokenPair is what the authorization-code exchange yields. AccessToken lives
// two hours; RefreshToken is the durable credential.
type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// exchangeCode trades the OAuth authorization code for tokens. Bitbucket
// authenticates the consumer via HTTP Basic auth on the token endpoint.
func (c *apiClient) exchangeCode(ctx context.Context, clientID, clientSecret, code string) (tokenPair, error) {
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bitbucketBaseURL+"/site/oauth2/access_token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return tokenPair{}, err
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var out tokenPair
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return tokenPair{}, err
	}
	if out.AccessToken == "" {
		return tokenPair{}, fmt.Errorf("token response carried no access_token")
	}
	return out, nil
}

// listWorkspaceSlugs attempts cross-workspace discovery. CHANGE-2770 removed
// every user-scoped listing endpoint (global /2.0/repositories,
// /2.0/user/permissions/*, role-filtered /2.0/workspaces), so this may return
// 410 depending on what Atlassian has left alive; callers must degrade to
// asking the user for a workspace slug.
func (c *apiClient) listWorkspaceSlugs(ctx context.Context, token string) ([]string, error) {
	u := bitbucketAPIBaseURL + "/2.0/workspaces?pagelen=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var out struct {
		Values []struct {
			Slug string `json:"slug"`
		} `json:"values"`
	}
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(out.Values))
	for _, ws := range out.Values {
		slugs = append(slugs, ws.Slug)
	}
	return slugs, nil
}

// listAdminRepositoriesInWorkspace returns the caller's admin-role
// repositories in one workspace, the minimum role for webhook registration.
// This workspace-scoped listing is the documented CHANGE-2770 workaround.
// First page only (100 repos); the POC picker does not paginate.
func (c *apiClient) listAdminRepositoriesInWorkspace(ctx context.Context, token, workspaceSlug string) ([]repository, error) {
	u := fmt.Sprintf("%s/2.0/repositories/%s?", bitbucketAPIBaseURL, url.PathEscape(workspaceSlug)) + url.Values{
		"role":    {"admin"},
		"pagelen": {"100"},
		"sort":    {"-updated_on"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var out struct {
		Values []repository `json:"values"`
	}
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, fmt.Errorf("list repositories in workspace %s: %w", workspaceSlug, err)
	}
	return out.Values, nil
}

// ensurePushWebhook registers the push webhook on the repository, skipping
// creation when a hook with the same URL already exists so reconnecting an
// app does not stack duplicate hooks. The secret makes Bitbucket sign
// deliveries with HMAC-SHA256 in X-Hub-Signature.
func (c *apiClient) ensurePushWebhook(ctx context.Context, token, repoFullName, hookURL, secret string) error {
	listURL := fmt.Sprintf("%s/2.0/repositories/%s/hooks?pagelen=100", bitbucketAPIBaseURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var hooks struct {
		Values []struct {
			URL string `json:"url"`
		} `json:"values"`
	}
	if err := c.do(req, http.StatusOK, &hooks); err != nil {
		return fmt.Errorf("list hooks: %w", err)
	}
	for _, hook := range hooks.Values {
		if hook.URL == hookURL {
			return nil
		}
	}

	body, err := json.Marshal(map[string]any{
		"description": "unkey-deploy",
		"url":         hookURL,
		"active":      true,
		"secret":      secret,
		"events":      []string{"repo:push"},
	})
	if err != nil {
		return err
	}
	createURL := fmt.Sprintf("%s/2.0/repositories/%s/hooks", bitbucketAPIBaseURL, repoFullName)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(body))
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
		return fmt.Errorf("bitbucket returned %s: %s", strconv.Itoa(resp.StatusCode), string(snippet))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
