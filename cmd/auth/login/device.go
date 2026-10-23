package login

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

type startedLogin struct {
	LoginID                 string
	UserCode                string
	VerificationURIComplete string
	ExpiresIn               int64
	Interval                int64
}

type pollResult struct {
	Status       string
	Interval     int64
	Detail       string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	WorkspaceID  string
	SessionID    string
}

type issuedSession struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	WorkspaceID  string
	SessionID    string
}

type apiEnvelope struct {
	Data struct {
		LoginID                 string `json:"loginId"`
		UserCode                string `json:"userCode"`
		VerificationURIComplete string `json:"verificationUriComplete"`
		ExpiresIn               int64  `json:"expiresIn"`
		Interval                int64  `json:"interval"`
		Status                  string `json:"status"`
		Detail                  string `json:"detail"`
		AccessToken             string `json:"accessToken"`
		RefreshToken            string `json:"refreshToken"`
		WorkspaceID             string `json:"workspaceId"`
		SessionID               string `json:"sessionId"`
	} `json:"data"`
	Error struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
	} `json:"error"`
}

func startLogin(ctx context.Context, client *http.Client, apiURL string, deviceName string) (startedLogin, error) {
	body := map[string]string{}
	if deviceName != "" {
		body["deviceName"] = deviceName
	}
	envelope, err := postJSON(ctx, client, apiURL, "/v2/cli.startDeviceLogin", body)
	if err != nil {
		return startedLogin{}, err
	}
	return startedLogin{
		LoginID:                 envelope.Data.LoginID,
		UserCode:                envelope.Data.UserCode,
		VerificationURIComplete: envelope.Data.VerificationURIComplete,
		ExpiresIn:               envelope.Data.ExpiresIn,
		Interval:                envelope.Data.Interval,
	}, nil
}

func pollUntilKey(ctx context.Context, client *http.Client, apiURL string, loginID string, deadline time.Time, interval time.Duration, sleep func(context.Context, time.Duration) error) (issuedSession, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	waiting := false
	for {
		if err := ctx.Err(); err != nil {
			return issuedSession{}, err
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return issuedSession{}, fmt.Errorf("login timed out. Run unkey login again")
		}
		result, err := pollOnce(ctx, client, apiURL, loginID)
		if err != nil {
			if ctx.Err() != nil || !retryable(err) {
				return issuedSession{}, err
			}
			if err := sleep(ctx, interval); err != nil {
				return issuedSession{}, err
			}
			continue
		}
		switch result.Status {
		case "complete":
			if result.AccessToken == "" || result.RefreshToken == "" {
				return issuedSession{}, fmt.Errorf("login completed without a session")
			}
			return issuedSession{
				AccessToken:  result.AccessToken,
				RefreshToken: result.RefreshToken,
				ExpiresIn:    result.ExpiresIn,
				WorkspaceID:  result.WorkspaceID,
				SessionID:    result.SessionID,
			}, nil
		case "access_denied", "expired_token":
			if result.Detail != "" {
				return issuedSession{}, fmt.Errorf("%s", result.Detail)
			}
			return issuedSession{}, fmt.Errorf("login %s", strings.ReplaceAll(result.Status, "_", " "))
		case "permissions_required":
			if !waiting {
				fmt.Println("Waiting for you to allow this login in the browser...")
			}
			waiting = true
			if result.Interval > 0 {
				interval = time.Duration(result.Interval) * time.Second
			}
		default:
			return issuedSession{}, fmt.Errorf("unexpected login status %q", result.Status)
		}
		if err := sleep(ctx, interval); err != nil {
			return issuedSession{}, err
		}
	}
}

func pollOnce(ctx context.Context, client *http.Client, apiURL string, loginID string) (pollResult, error) {
	envelope, err := postJSON(ctx, client, apiURL, "/v2/cli.pollDeviceLogin", map[string]string{"loginId": loginID})
	if err != nil {
		return pollResult{}, err
	}
	return pollResult{
		Status:       envelope.Data.Status,
		Interval:     envelope.Data.Interval,
		Detail:       envelope.Data.Detail,
		AccessToken:  envelope.Data.AccessToken,
		RefreshToken: envelope.Data.RefreshToken,
		ExpiresIn:    envelope.Data.ExpiresIn,
		WorkspaceID:  envelope.Data.WorkspaceID,
		SessionID:    envelope.Data.SessionID,
	}, nil
}

func revokeIssuedSession(ctx context.Context, client *http.Client, apiURL string, issued issuedSession) error {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	raw, err := json.Marshal(map[string]string{"sessionId": issued.SessionID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiURL, "/")+"/v2/cli.revokeSession", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+issued.AccessToken)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 300 {
		return fmt.Errorf("revoke returned %d", res.StatusCode)
	}
	return nil
}

func postJSON(ctx context.Context, client *http.Client, apiURL string, path string, body any) (apiEnvelope, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return apiEnvelope{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiURL, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return apiEnvelope{}, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return apiEnvelope{}, transientError{err: err}
	}
	defer func() { _ = res.Body.Close() }()
	transient := res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return apiEnvelope{}, transientError{err: err}
	}
	var envelope apiEnvelope
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &envelope); err != nil {
			err := fmt.Errorf("unkey API returned %d", res.StatusCode)
			if transient {
				return apiEnvelope{}, transientError{err: err}
			}
			return apiEnvelope{}, err
		}
	}
	if res.StatusCode >= 300 {
		detail := envelope.Error.Detail
		if detail == "" {
			detail = envelope.Error.Title
		}
		if detail == "" {
			detail = fmt.Sprintf("unkey API returned %d", res.StatusCode)
		}
		err := fmt.Errorf("%s", detail)
		if transient {
			return apiEnvelope{}, transientError{err: err}
		}
		return apiEnvelope{}, err
	}
	return envelope, nil
}

type transientError struct{ err error }

func (e transientError) Error() string { return e.err.Error() }
func (e transientError) Unwrap() error { return e.err }

func retryable(err error) bool {
	var transient transientError
	return errors.As(err, &transient)
}

func formatUserCode(code string) string {
	fields := strings.Fields(strings.TrimSpace(code))
	compact := strings.ToUpper(strings.Join(fields, ""))
	var b strings.Builder
	for i, r := range compact {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
