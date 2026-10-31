package workos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// get performs an authenticated GET and decodes the JSON response into out. A
// non-2xx status is an error carrying a bounded slice of the body, which is
// where WorkOS puts its error code and message.
func (c *client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return &statusError{method: http.MethodGet, path: path, code: res.StatusCode, body: body}
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// delete performs an authenticated DELETE and reports whether the resource
// existed. A 404 means it is already gone, which is the state the caller asked
// for, so it is not an error.
func (c *client) delete(ctx context.Context, path string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return false, fmt.Errorf("workos DELETE %s: status %d: %s", path, res.StatusCode, body)
	}
	return true, nil
}

// statusError is a non-2xx WorkOS response.
type statusError struct {
	method string
	path   string
	code   int
	body   []byte
}

func (e *statusError) Error() string {
	return fmt.Sprintf("workos %s %s: status %d: %s", e.method, e.path, e.code, e.body)
}
