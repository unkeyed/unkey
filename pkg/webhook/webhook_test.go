package webhook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// staticVerifier returns a fixed event, or an error when the body says so.
type staticVerifier struct {
	event Event
}

func (v staticVerifier) Verify(_ *http.Request, body []byte) (Event, error) {
	if string(body) == "tampered" {
		return Event{}, errors.New("bad signature")
	}
	return v.event, nil
}

func post(t *testing.T, rec *Receiver, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	rec.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhooks/test", strings.NewReader(body)))
	return w
}

func TestReceiver(t *testing.T) {
	newReceiver := func(eventType string) *Receiver {
		return New("test", staticVerifier{event: Event{ID: "evt_1", Type: eventType, Payload: []byte(`{}`)}})
	}

	t.Run("handled event returns 200", func(t *testing.T) {
		called := false
		rec := newReceiver("thing.created").On(func(ctx context.Context, e Event) error {
			called = true
			require.Equal(t, "evt_1", e.ID)
			return nil
		}, "thing.created")
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		require.True(t, called)
	})

	t.Run("ignored event returns 200", func(t *testing.T) {
		rec := newReceiver("thing.created").On(func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		}, "thing.created")
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
	})

	t.Run("handler error returns 500 so the provider retries", func(t *testing.T) {
		rec := newReceiver("thing.created").On(func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		}, "thing.created")
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "{}").Code)
	})

	t.Run("unregistered event type is acknowledged", func(t *testing.T) {
		rec := newReceiver("thing.deleted").On(func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for other event types")
			return nil
		}, "thing.created")
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
	})

	t.Run("verification failure returns 401", func(t *testing.T) {
		rec := newReceiver("thing.created").On(func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for unverified requests")
			return nil
		}, "thing.created")
		require.Equal(t, http.StatusUnauthorized, post(t, rec, "tampered").Code)
	})

	t.Run("one handler serves several event types", func(t *testing.T) {
		for _, eventType := range []string{"branch.created", "branch.deleted"} {
			called := false
			rec := newReceiver(eventType).On(func(ctx context.Context, e Event) error {
				called = true
				return nil
			}, "branch.created", "branch.deleted")
			require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
			require.True(t, called, eventType)
		}
	})

	t.Run("On without event types panics", func(t *testing.T) {
		require.Panics(t, func() {
			newReceiver("thing.created").On(func(ctx context.Context, e Event) error { return nil })
		})
	})

	t.Run("default handler takes the fallback path", func(t *testing.T) {
		var got string
		rec := newReceiver("thing.deleted").
			On(func(ctx context.Context, e Event) error {
				t.Fatal("registered handler must not run for other event types")
				return nil
			}, "thing.created").
			Default(func(ctx context.Context, e Event) error {
				got = e.Type
				return nil
			})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		require.Equal(t, "thing.deleted", got)
	})

	t.Run("default handler error returns 500", func(t *testing.T) {
		rec := newReceiver("thing.deleted").Default(func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		})
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "{}").Code)
	})

	t.Run("non-POST is rejected", func(t *testing.T) {
		rec := newReceiver("thing.created")
		w := httptest.NewRecorder()
		rec.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhooks/test", nil))
		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestTyped(t *testing.T) {
	newReceiver := func(payload string) *Receiver {
		return New("test", staticVerifier{
			event: Event{ID: "evt_1", Type: "thing.created", Payload: []byte(payload)},
		})
	}

	type thing struct {
		Name string `json:"name"`
	}

	t.Run("handler receives the parsed payload", func(t *testing.T) {
		var got thing
		rec := newReceiver(`{"name":"box"}`).On(
			Typed(func(ctx context.Context, e Event, payload thing) error {
				got = payload
				return nil
			}),
			"thing.created",
		)
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		require.Equal(t, "box", got.Name)
	})

	t.Run("malformed payload is a handler error", func(t *testing.T) {
		rec := newReceiver(`not json`).On(
			Typed(func(ctx context.Context, e Event, payload thing) error {
				t.Fatal("handler must not run for unparseable payloads")
				return nil
			}),
			"thing.created",
		)
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "{}").Code)
	})
}
