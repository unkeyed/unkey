package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// The server lets a rotated refresh token be presented again for 10 seconds
// and answers with a fresh pair. Both attempts have to fit in that window, so
// a rotation whose response was lost is recovered by the retry instead of
// revoking the session on the next command.
var refreshAttemptTimeout = 4 * time.Second

type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }

// Refresh exchanges a refresh token for a new session pair. A transport
// failure, timeout, or 5xx response is retried once with the same token.
func Refresh(ctx context.Context, apiURL string, refreshToken string) (Session, error) {
	next, err := refreshOnce(ctx, apiURL, refreshToken)
	var retryable retryableError
	if err == nil || ctx.Err() != nil || !errors.As(err, &retryable) {
		return next, err
	}
	return refreshOnce(ctx, apiURL, refreshToken)
}

func refreshOnce(ctx context.Context, apiURL string, refreshToken string) (Session, error) {
	body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiURL, "/")+"/v2/cli.exchangeToken", bytes.NewReader(body))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("content-type", "application/json")
	res, err := (&http.Client{Timeout: refreshAttemptTimeout}).Do(req)
	if err != nil {
		return Session{}, retryableError{err: err}
	}
	defer func() { _ = res.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Session{}, retryableError{err: err}
	}
	if res.StatusCode >= 500 {
		return Session{}, retryableError{err: fmt.Errorf("refresh returned %d", res.StatusCode)}
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
