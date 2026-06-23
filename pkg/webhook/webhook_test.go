package webhook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// staticVerifier returns a fixed event, or an error when the body says so. It
// reads the body from the request like a real verifier does.
type staticVerifier struct {
	event Event
}

func (v staticVerifier) Verify(r *http.Request) (Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return Event{}, err
	}
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
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			called = true
			require.Equal(t, "evt_1", e.ID)
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		require.True(t, called)
	})

	t.Run("ignored event returns 200", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
	})

	t.Run("handler error returns 500 so the provider retries", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		})
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "{}").Code)
	})

	t.Run("bad request returns 400 without retry", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: malformed", ErrBadRequest)
		})
		require.Equal(t, http.StatusBadRequest, post(t, rec, "{}").Code)
	})

	t.Run("unregistered event type is acknowledged", func(t *testing.T) {
		rec := newReceiver("thing.deleted").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for other event types")
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
	})

	t.Run("verification failure returns 401", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for unverified requests")
			return nil
		})
		require.Equal(t, http.StatusUnauthorized, post(t, rec, "tampered").Code)
	})

	t.Run("one handler serves several event types", func(t *testing.T) {
		for _, eventType := range []string{"branch.created", "branch.deleted"} {
			called := false
			rec := newReceiver(eventType).On([]string{"branch.created", "branch.deleted"}, func(ctx context.Context, e Event) error {
				called = true
				return nil
			})
			require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
			require.True(t, called, eventType)
		}
	})

	t.Run("On without event types panics", func(t *testing.T) {
		require.Panics(t, func() {
			newReceiver("thing.created").On(nil, func(ctx context.Context, e Event) error { return nil })
		})
	})

	t.Run("default handler takes the fallback path", func(t *testing.T) {
		var got string
		rec := newReceiver("thing.deleted").
			On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
				t.Fatal("registered handler must not run for other event types")
				return nil
			}).
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

	t.Run("default handler can ignore via ErrIgnore", func(t *testing.T) {
		rec := newReceiver("thing.deleted").Default(func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
	})

	t.Run("middleware runs outermost-first in Use order", func(t *testing.T) {
		var order []string
		mw := func(name string) Middleware {
			return func(next HandlerFunc) HandlerFunc {
				return func(ctx context.Context, e Event) error {
					order = append(order, name)
					return next(ctx, e)
				}
			}
		}
		rec := newReceiver("thing.created").
			Use(mw("first"), mw("second")).
			On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
				order = append(order, "handler")
				return nil
			})
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		// The first Use wraps outermost, so it runs before the second.
		require.Equal(t, []string{"first", "second", "handler"}, order)
	})

	t.Run("non-POST is rejected", func(t *testing.T) {
		rec := newReceiver("thing.created")
		w := httptest.NewRecorder()
		rec.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhooks/test", nil))
		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("body over the size limit returns 413", func(t *testing.T) {
		rec := New("test",
			staticVerifier{event: Event{ID: "evt_1", Type: "thing.created", Payload: []byte(`{}`)}},
			WithMaxBodySize(8),
		).On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for oversized bodies")
			return nil
		})
		require.Equal(t, http.StatusRequestEntityTooLarge, post(t, rec, strings.Repeat("a", 64)).Code)
	})

	t.Run("WithMaxBodySize raises the limit", func(t *testing.T) {
		called := false
		rec := New("test",
			staticVerifier{event: Event{ID: "evt_1", Type: "thing.created", Payload: []byte(`{}`)}},
			WithMaxBodySize(128),
		).On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			called = true
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, strings.Repeat("a", 64)).Code)
		require.True(t, called)
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
			[]string{"thing.created"},
			Typed(func(ctx context.Context, e Event, payload thing) error {
				got = payload
				return nil
			}),
		)
		require.Equal(t, http.StatusOK, post(t, rec, "{}").Code)
		require.Equal(t, "box", got.Name)
	})

	t.Run("malformed payload is a 400 bad request", func(t *testing.T) {
		rec := newReceiver(`not json`).On(
			[]string{"thing.created"},
			Typed(func(ctx context.Context, e Event, payload thing) error {
				t.Fatal("handler must not run for unparseable payloads")
				return nil
			}),
		)
		require.Equal(t, http.StatusBadRequest, post(t, rec, "{}").Code)
	})
}
