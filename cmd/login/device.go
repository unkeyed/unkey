package login

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

type startedLogin struct {
	LoginID                 string
	UserCode                string
	VerificationURIComplete string
	ExpiresIn               int64
	Interval                int64
}

type pollResult struct {
	Status   string
	Interval int64
	Detail   string
	Key      string
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
		Key                     string `json:"key"`
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

func pollUntilKey(ctx context.Context, client *http.Client, apiURL string, loginID string, deadline time.Time, interval time.Duration, sleep func(context.Context, time.Duration) error) (string, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	last := ""
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return "", fmt.Errorf("login timed out. Run unkey login again")
		}
		result, err := pollOnce(ctx, client, apiURL, loginID)
		if err != nil {
			return "", err
		}
		switch result.Status {
		case "complete":
			if result.Key == "" {
				return "", fmt.Errorf("login completed without a root key")
			}
			return result.Key, nil
		case "access_denied", "expired_token":
			if result.Detail != "" {
				return "", fmt.Errorf("%s", result.Detail)
			}
			return "", fmt.Errorf("login %s", strings.ReplaceAll(result.Status, "_", " "))
		case "authorization_pending", "slow_down", "permissions_required":
			if result.Status != "slow_down" && result.Status != last {
				fmt.Println("Waiting for you to allow this login in the browser...")
			}
			last = result.Status
			if result.Interval > 0 {
				interval = time.Duration(result.Interval) * time.Second
			}
		default:
			return "", fmt.Errorf("unexpected login status %q", result.Status)
		}
		if err := sleep(ctx, interval); err != nil {
			return "", err
		}
	}
}

func pollOnce(ctx context.Context, client *http.Client, apiURL string, loginID string) (pollResult, error) {
	envelope, err := postJSON(ctx, client, apiURL, "/v2/cli.pollDeviceLogin", map[string]string{"loginId": loginID})
	if err != nil {
		return pollResult{}, err
	}
	return pollResult{
		Status:   envelope.Data.Status,
		Interval: envelope.Data.Interval,
		Detail:   envelope.Data.Detail,
		Key:      envelope.Data.Key,
	}, nil
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
		return apiEnvelope{}, err
	}
	defer func() { _ = res.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return apiEnvelope{}, err
	}
	var envelope apiEnvelope
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return apiEnvelope{}, fmt.Errorf("unkey API returned %d", res.StatusCode)
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
		return apiEnvelope{}, fmt.Errorf("%s", detail)
	}
	return envelope, nil
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
