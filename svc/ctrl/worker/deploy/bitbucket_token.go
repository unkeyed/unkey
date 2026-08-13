package deploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// bitbucketTokens is what a refresh-grant exchange yields. RefreshToken is
// the ROTATED credential: Atlassian OAuth clients invalidate the used refresh
// token on every exchange, and reusing a stale one trips reuse detection,
// which revokes the entire token family. Callers must persist it immediately.
type bitbucketTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// bitbucketCloneAuthHeader builds the Authorization header value BuildKit
// sends verbatim on bitbucket.org fetches. Bitbucket's git endpoint rejects
// Bearer for OAuth access tokens (git then falls back to the disabled
// credential prompt); the documented form is basic auth with the literal
// x-token-auth username. BuildKit's GIT_AUTH_TOKEN path can't produce this:
// it hardcodes the x-access-token username GitHub expects.
func bitbucketCloneAuthHeader(accessToken string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("x-token-auth:"+accessToken))
}

// refreshBitbucketToken exchanges a stored refresh token for a fresh two-hour
// access token via Bitbucket's OAuth token endpoint (POC: the connect flow
// persists refresh tokens as the clone credential).
func refreshBitbucketToken(ctx context.Context, cfg BitbucketConfig, refreshToken string) (bitbucketTokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://bitbucket.org/site/oauth2/access_token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return bitbucketTokens{}, err
	}
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return bitbucketTokens{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return bitbucketTokens{}, fmt.Errorf("bitbucket token refresh returned %d: %s", resp.StatusCode, string(snippet))
	}

	var out bitbucketTokens
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return bitbucketTokens{}, err
	}
	if out.AccessToken == "" {
		return bitbucketTokens{}, fmt.Errorf("bitbucket token refresh carried no access_token")
	}
	return out, nil
}
