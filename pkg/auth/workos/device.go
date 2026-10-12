package workos

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

const defaultDeviceAPIBaseURL = "https://api.workos.com"

type DeviceCode string

type DeviceAuthorization struct {
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int
	Interval                int
}

type DeviceError struct {
	Code        string
	Description string
	StatusCode  int
}

func (e *DeviceError) Error() string {
	if e.Description != "" {
		return e.Code + ": " + e.Description
	}
	return e.Code
}

type DeviceClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewDeviceClient returns a client for https://api.workos.com when baseURL is empty.
func NewDeviceClient(apiKey string, baseURL string) *DeviceClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultDeviceAPIBaseURL
	}
	return &DeviceClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (c *DeviceClient) CreateDevice(ctx context.Context, clientID string) (DeviceCode, DeviceAuthorization, error) {
	body, err := json.Marshal(map[string]string{"client_id": clientID})
	if err != nil {
		return "", DeviceAuthorization{}, err
	}
	var parsed deviceAuthorizationResponse
	if err := c.postJSON(ctx, "/user_management/authorize/device", body, &parsed); err != nil {
		return "", DeviceAuthorization{}, err
	}
	if parsed.DeviceCode == "" || parsed.UserCode == "" {
		return "", DeviceAuthorization{}, fmt.Errorf("workos device authorization response is missing codes")
	}
	if parsed.ExpiresIn <= 0 {
		parsed.ExpiresIn = 300
	}
	if parsed.Interval <= 0 {
		parsed.Interval = 5
	}
	return DeviceCode(parsed.DeviceCode), DeviceAuthorization{
		UserCode:                parsed.UserCode,
		VerificationURI:         parsed.VerificationURI,
		VerificationURIComplete: parsed.VerificationURIComplete,
		ExpiresIn:               parsed.ExpiresIn,
		Interval:                parsed.Interval,
	}, nil
}

func (c *DeviceClient) postJSON(ctx context.Context, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return c.do(req, out)
}

func (c *DeviceClient) do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var oauthErr oauthErrorBody
		_ = json.Unmarshal(payload, &oauthErr)
		if oauthErr.Error == "" {
			return fmt.Errorf("workos %s: status %d", req.URL.Path, res.StatusCode)
		}
		return &DeviceError{Code: oauthErr.Error, Description: oauthErr.ErrorDescription, StatusCode: res.StatusCode}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("workos %s: decode response: %w", req.URL.Path, err)
	}
	return nil
}
