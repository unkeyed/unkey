package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/sdk/models/apierrors"
	"github.com/unkeyed/unkey/internal/sdk/models/operations"
)

// NewClient returns a typed client with bounded responses and no automatic retries.
// Non-2xx responses return ResponseError; failed transfers return TransportError.
// A nil HTTP client uses a 30-second timeout. Credentials are explicit per request.
func NewClient(server string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return New(WithServerURL(server), WithClient(httpClient{client: client}))
}

// Bearer authenticates one request without storing credentials on the client.
func Bearer(token string) operations.Option {
	return operations.WithSetHeaders(map[string]string{"Authorization": "Bearer " + token})
}

// ResponseError retains the HTTP status even when an error body is not JSON.
type ResponseError struct {
	StatusCode int
	Detail     string
}

func (e *ResponseError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("unkey API returned %d", e.StatusCode)
}

// TransportError identifies a failed request or incomplete response transfer.
type TransportError struct{ err error }

func (e *TransportError) Error() string { return e.err.Error() }
func (e *TransportError) Unwrap() error { return e.err }

type httpClient struct{ client *http.Client }

func (c httpClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, &TransportError{err: err}
	}
	const responseBytesMax = 1 << 20
	payload, readErr := io.ReadAll(io.LimitReader(res.Body, responseBytesMax+1))
	closeErr := res.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, &TransportError{err: err}
	}
	if len(payload) > responseBytesMax {
		return nil, &ResponseError{StatusCode: res.StatusCode, Detail: "unkey API response exceeds 1 MiB"}
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		responseErr := &ResponseError{StatusCode: res.StatusCode, Detail: ""}
		var envelope apierrors.UnauthorizedErrorResponse
		if json.Unmarshal(payload, &envelope) == nil {
			responseErr.Detail = envelope.Error_.Detail
			if responseErr.Detail == "" {
				responseErr.Detail = envelope.Error_.Title
			}
		}
		return nil, responseErr
	}
	res.Body = io.NopCloser(bytes.NewReader(payload))
	return res, nil
}
