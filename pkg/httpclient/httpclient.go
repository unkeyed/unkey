package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Request builds and executes an HTTP request with JSON marshalling/unmarshalling.
// The body is optional — pass nil for requests without a body.
// Returns the decoded response or an error if the status code doesn't match expectedStatus.
func Request[T any](client *http.Client, method string, url string, headers map[string]string, body any, expectedStatus int) (T, error) {
	var zero T

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return zero, fault.Wrap(err, fault.Internal("failed to marshal request body"))
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return zero, fault.Wrap(err, fault.Internal("failed to create request"))
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return zero, fault.Wrap(err, fault.Internal(fmt.Sprintf("%s %s failed", method, url)))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != expectedStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return zero, fault.New(
			fmt.Sprintf("%s %s returned unexpected status", method, url),
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))),
		)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fault.Wrap(err, fault.Internal("failed to decode response"))
	}

	return result, nil
}

// Do executes an HTTP request without decoding the response body.
// Returns an error if the status code doesn't match expectedStatus.
func Do(client *http.Client, method string, url string, headers map[string]string, body any, expectedStatus int) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fault.Wrap(err, fault.Internal("failed to marshal request body"))
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to create request"))
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("%s %s failed", method, url)))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != expectedStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fault.New(
			fmt.Sprintf("%s %s returned unexpected status", method, url),
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))),
		)
	}

	return nil
}

// StatusCheck executes an HTTP request and returns whether the status matches expectedStatus.
// Unlike Do, this doesn't return an error for non-matching statuses — it returns false.
// Only returns an error for transport-level failures.
func StatusCheck(client *http.Client, method string, url string, headers map[string]string, expectedStatus int) (bool, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false, fault.Wrap(err, fault.Internal("failed to create request"))
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fault.Wrap(err, fault.Internal(fmt.Sprintf("%s %s failed", method, url)))
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == expectedStatus, nil
}

// GitHubHeaders returns common GitHub API headers with the given bearer token.
func GitHubHeaders(token string) map[string]string {
	h := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return h
}
