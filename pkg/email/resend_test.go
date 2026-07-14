package email

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResendSend(t *testing.T) {
	t.Run("posts template id, variables, and auth", func(t *testing.T) {
		var gotAuth, gotContentType string
		var gotBody resendRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotContentType = r.Header.Get("Content-Type")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":"email_1"}`)
		}))
		defer srv.Close()

		err := newTestResend(srv.URL, "re_test_key", "Unkey <billing@unkey.com>").Send(
			context.Background(),
			Email{
				To:         []string{"a@example.com"},
				TemplateID: "tmpl_123",
				Variables:  map[string]string{"BUDGET": "$300"},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "Bearer re_test_key", gotAuth)
		require.Equal(t, "application/json", gotContentType)
		require.Equal(t, "Unkey <billing@unkey.com>", gotBody.From, "empty From falls back to the default")
		require.Equal(t, []string{"a@example.com"}, gotBody.To)
		require.Equal(t, "tmpl_123", gotBody.Template.ID)
		require.Equal(t, "$300", gotBody.Template.Variables["BUDGET"])
	})

	t.Run("explicit From overrides the default", func(t *testing.T) {
		var gotBody resendRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		err := newTestResend(srv.URL, "re_test_key", "default@unkey.com").Send(
			context.Background(),
			Email{From: "alerts@unkey.com", To: []string{"a@example.com"}, TemplateID: "tmpl_123"},
		)
		require.NoError(t, err)
		require.Equal(t, "alerts@unkey.com", gotBody.From)
	})

	t.Run("sets Idempotency-Key when provided", func(t *testing.T) {
		var gotKey string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotKey = r.Header.Get("Idempotency-Key")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		err := newTestResend(srv.URL, "re_test_key", "default@unkey.com").Send(
			context.Background(),
			Email{
				To:             []string{"a@example.com"},
				TemplateID:     "tmpl_123",
				IdempotencyKey: "budget-alert/ws_abc:2026-06",
			},
		)
		require.NoError(t, err)
		require.Equal(t, "budget-alert/ws_abc:2026-06", gotKey)
	})

	t.Run("non-2xx is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"message":"template not published"}`)
		}))
		defer srv.Close()

		err := newTestResend(srv.URL, "re_test_key", "default@unkey.com").Send(
			context.Background(),
			Email{To: []string{"a@example.com"}, TemplateID: "tmpl_123"},
		)
		require.Error(t, err)
	})

	t.Run("validates recipients and template before sending", func(t *testing.T) {
		s := NewResend("re_test_key", "default@unkey.com")
		require.Error(t, s.Send(context.Background(), Email{TemplateID: "tmpl_123"}))
		require.Error(t, s.Send(context.Background(), Email{To: []string{"a@example.com"}}))
	})

	t.Run("rejects idempotency keys outside 1-256 chars", func(t *testing.T) {
		s := NewResend("re_test_key", "default@unkey.com")
		require.Error(t, s.Send(context.Background(), Email{
			To:             []string{"a@example.com"},
			TemplateID:     "tmpl_123",
			IdempotencyKey: string(make([]byte, 257)),
		}))
	})
}

func TestNoopSend(t *testing.T) {
	require.NoError(t, NewNoop().Send(context.Background(), Email{
		To:         []string{"a@example.com"},
		TemplateID: "tmpl_123",
	}))
}
