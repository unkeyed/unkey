package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDo_JSONDecode(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(payload{Name: "test", Count: 42}))
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := Do[payload](context.Background(), client.Get("/data"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "test", resp.Body.Name)
	require.Equal(t, 42, resp.Body.Count)
	require.NotEmpty(t, resp.RawBody)
}

func TestSend_NoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Delete("/resource").Send(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestPostWithBody(t *testing.T) {
	type req struct {
		Value string `json:"value"`
	}
	type res struct {
		ID int `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var got req
		require.NoError(t, json.Unmarshal(body, &got))
		require.Equal(t, "hello", got.Value)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		require.NoError(t, json.NewEncoder(w).Encode(res{ID: 1}))
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := Do[res](context.Background(),
		client.Post("/items", req{Value: "hello"}).
			AcceptStatus(http.StatusCreated),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, 1, resp.Body.ID)
}

func TestStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/missing").Send(context.Background())
	require.Error(t, err)

	var httpErr *Error
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusNotFound, httpErr.StatusCode)
	require.Contains(t, string(httpErr.Body), "not found")

	// Response is still populated on error
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.NotEmpty(t, resp.RawBody)
}

func TestAcceptStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))

	// Without AcceptStatus, 202 is accepted (2xx range)
	resp, err := client.Get("/").Send(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	// With AcceptStatus(200), 202 is an error
	_, err = client.Get("/").AcceptStatus(http.StatusOK).Send(context.Background())
	require.Error(t, err)

	var httpErr *Error
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusAccepted, httpErr.StatusCode)
}

func TestAcceptMultipleStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Patch("/cancel", nil).
		AcceptStatus(http.StatusAccepted, http.StatusNotFound).
		Send(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBaseURLConcat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/users", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL + "/api/v1"))
	_, err := client.Get("/users").Send(context.Background())
	require.NoError(t, err)
}

func TestHeaderMerging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Client-level header
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		// Per-request header overrides client-level
		require.Equal(t, "custom-token", r.Header.Get("Authorization"))
		// Per-request only header
		require.Equal(t, "req-id-123", r.Header.Get("X-Request-Id"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithHeader("Accept", "application/json"),
		WithBearerToken("client-token"),
	)
	_, err := client.Get("/").
		Header("Authorization", "custom-token").
		Header("X-Request-Id", "req-id-123").
		Send(context.Background())
	require.NoError(t, err)
}

func TestQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "bar", r.URL.Query().Get("foo"))
		require.Equal(t, "10", r.URL.Query().Get("limit"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	_, err := client.Get("/search").
		Query("foo", "bar").
		Query("limit", "10").
		Send(context.Background())
	require.NoError(t, err)
}

func TestEmptyResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	type result struct {
		Value string `json:"value"`
	}

	client := New(WithBaseURL(srv.URL))
	resp, err := Do[result](context.Background(), client.Get("/empty"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, result{}, resp.Body) // zero value
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := New(WithBaseURL(srv.URL))
	_, err := client.Get("/slow").Send(ctx)
	require.Error(t, err)
}

func TestWithUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "unkey/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL), WithUserAgent("unkey/1.0"))
	_, err := client.Get("/").Send(context.Background())
	require.NoError(t, err)
}

func TestPutMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	_, err := client.Put("/resource", map[string]string{"key": "val"}).Send(context.Background())
	require.NoError(t, err)
}

func TestPatchMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	_, err := client.Patch("/resource", map[string]string{"key": "val"}).Send(context.Background())
	require.NoError(t, err)
}

func TestErrorMessage(t *testing.T) {
	e := &Error{StatusCode: 404, Body: []byte(`{"error":"not found"}`)}
	require.Equal(t, `http 404: {"error":"not found"}`, e.Error())
}

func TestContentTypeOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	_, err := client.Post("/data", "plain text").
		Header("Content-Type", "text/plain").
		Send(context.Background())
	require.NoError(t, err)
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	client := New(WithHTTPClient(custom))
	require.Equal(t, custom, client.httpClient)
}

func TestResponseHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "hello")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/").Send(context.Background())
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Headers.Get("X-Custom"))
}
