package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/fault"
)

// request builds and executes an HTTP request with JSON marshalling/unmarshalling.
// The body is optional — pass nil for requests without a body.
// Returns the decoded response or an error if the status code doesn't match expectedStatus.
func request[T any](client *http.Client, method string, url string, headers map[string]string, body any, expectedStatus int) (T, error) {
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

// doRequest executes an HTTP request without decoding the response body.
// Returns an error if the status code doesn't match expectedStatus.
func doRequest(client *http.Client, method string, url string, headers map[string]string, body any, expectedStatus int) error {
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
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != expectedStatus {
		respBody, _ := io.ReadAll(resp.Body)
		return fault.New(
			fmt.Sprintf("%s %s returned unexpected status", method, url),
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))),
		)
	}

	return nil
}

// statusCheck executes an HTTP request and returns true if the response status matches expectedStatus.
// Returns (false, nil) for non-matching statuses, (false, error) for transport errors.
func statusCheck(client *http.Client, method string, url string, headers map[string]string, expectedStatus int) (bool, error) {
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
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	return resp.StatusCode == expectedStatus, nil
}

// githubHeaders returns common GitHub API headers with the given bearer token.
func githubHeaders(token string) map[string]string {
	h := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return h
}
