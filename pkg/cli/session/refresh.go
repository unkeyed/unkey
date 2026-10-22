package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Refresh exchanges a refresh token for a new session pair.
func Refresh(ctx context.Context, apiURL string, refreshToken string) (Session, error) {
	body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiURL, "/")+"/v2/cli.refreshToken", bytes.NewReader(body))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("content-type", "application/json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = res.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Session{}, err
	}
	var envelope struct {
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresIn    int64  `json:"expiresIn"`
			WorkspaceID  string `json:"workspaceId"`
			SessionID    string `json:"sessionId"`
		} `json:"data"`
		Error struct {
			Detail string `json:"detail"`
		} `json:"error"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return Session{}, fmt.Errorf("refresh returned %d", res.StatusCode)
		}
	}
	if res.StatusCode >= 300 {
		if envelope.Error.Detail != "" {
			return Session{}, fmt.Errorf("%s", envelope.Error.Detail)
		}
		return Session{}, fmt.Errorf("refresh returned %d", res.StatusCode)
	}
	return Session{
		WorkspaceID:  envelope.Data.WorkspaceID,
		SessionID:    envelope.Data.SessionID,
		AccessToken:  envelope.Data.AccessToken,
		RefreshToken: envelope.Data.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(envelope.Data.ExpiresIn) * time.Second).Unix(),
		APIURL:       NormalizeAPIURL(apiURL),
	}, nil
}
