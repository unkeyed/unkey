package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// refreshBitbucketToken exchanges a stored refresh token for a fresh two-hour
// access token via Bitbucket's OAuth token endpoint (POC: the connect flow
// persists refresh tokens as the clone credential). Bitbucket refresh tokens
// do not rotate, so the stored credential stays valid across exchanges.
func refreshBitbucketToken(ctx context.Context, cfg BitbucketConfig, refreshToken string) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://bitbucket.org/site/oauth2/access_token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("bitbucket token refresh returned %d: %s", resp.StatusCode, string(snippet))
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("bitbucket token refresh carried no access_token")
	}
	return out.AccessToken, nil
}
