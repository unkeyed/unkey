package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// staticVerifier returns a fixed event, or an error when the body says so. It
// reads the body from the request like a real verifier does.
type staticVerifier struct {
	event Event
	err   error
}

func (v staticVerifier) Verify(r *http.Request) (Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return Event{}, err
	}
	if string(body) == "tampered" {
		return Event{}, errors.New("bad signature")
	}
	return v.event, v.err
}

func post(t *testing.T, rec *Receiver, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	rec.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhooks/test", strings.NewReader(body)))
	return w
}

func TestReceiver(t *testing.T) {
	eventID := uid.New(uid.TestPrefix)
	newReceiver := func(eventType string) *Receiver {
		return New("test", staticVerifier{event: Event{ID: eventID, Type: eventType}})
	}

	t.Run("custom acknowledgment survives middleware and concurrent requests", func(t *testing.T) {
		type requestIDKey struct{}
		rec := newReceiver("thing.created").
			Use(func(next HandlerFunc) HandlerFunc {
				return func(ctx context.Context, event Event) error {
					id, ok := ctx.Value(requestIDKey{}).(string)
					if !ok {
						return errors.New("missing request ID")
					}
					event.ID = id
					return next(ctx, event)
				}
			}).
			OnResponse([]string{"thing.created"}, func(ctx context.Context, event Event) (Response, error) {
				body, err := json.Marshal(event.ID)
				return Response{StatusCode: http.StatusCreated, ContentType: "application/json", Body: body}, err
			})
		for i := range 8 {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				want := uid.New(uid.TestPrefix)
				ctx := context.WithValue(t.Context(), requestIDKey{}, want)
				response := httptest.NewRecorder()
				rec.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodPost, "/", nil))
				require.Equal(t, http.StatusCreated, response.Code)
				require.Equal(t, "application/json", response.Header().Get("Content-Type"))
				var got string
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &got))
				require.Equal(t, want, got)
			})
		}
	})

	t.Run("middleware error discards acknowledgment", func(t *testing.T) {
		rec := newReceiver("thing.created").
			Use(func(next HandlerFunc) HandlerFunc {
				return func(ctx context.Context, event Event) error {
					if err := next(ctx, event); err != nil {
						return err
					}
					return fmt.Errorf("%w: queue unavailable", ErrUnavailable)
				}
			}).
			OnResponse([]string{"thing.created"}, func(ctx context.Context, event Event) (Response, error) {
				return Response{StatusCode: http.StatusCreated, Body: []byte("must not be sent")}, nil
			})
		response := post(t, rec, "")
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.NotContains(t, response.Body.String(), "must not be sent")
	})

	t.Run("verification errors distinguish retryable failures", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			err    error
			status int
		}{
			{name: "unavailable", err: ErrUnavailable, status: http.StatusServiceUnavailable},
			{name: "bad request", err: ErrBadRequest, status: http.StatusBadRequest},
			{name: "invalid signature", err: errors.New("signature mismatch"), status: http.StatusUnauthorized},
		} {
			t.Run(test.name, func(t *testing.T) {
				rec := New("test", staticVerifier{err: fmt.Errorf("private detail: %w", test.err)}).
					Default(func(ctx context.Context, event Event) error {
						t.Fatal("unverified requests must not reach the handler")
						return nil
					})
				response := post(t, rec, "")
				require.Equal(t, test.status, response.Code)
				require.NotContains(t, response.Body.String(), "private detail")
			})
		}
	})

	t.Run("handled event returns 200", func(t *testing.T) {
		called := false
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			called = true
			require.Equal(t, eventID, e.ID)
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
		require.True(t, called)
	})

	t.Run("ignored event returns 200", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
	})

	t.Run("handler error returns 500 so the provider retries", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		})
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "").Code)
	})

	t.Run("bad request returns 400 without retry", func(t *testing.T) {
		rec := newReceiver("thing.created").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: malformed", ErrBadRequest)
		})
		require.Equal(t, http.StatusBadRequest, post(t, rec, "").Code)
	})

	t.Run("unregistered event type is acknowledged", func(t *testing.T) {
		rec := newReceiver("thing.deleted").On([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for other event types")
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
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
			require.Equal(t, http.StatusOK, post(t, rec, "").Code)
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
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
		require.Equal(t, "thing.deleted", got)
	})

	t.Run("default handler error returns 500", func(t *testing.T) {
		rec := newReceiver("thing.deleted").Default(func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		})
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "").Code)
	})

	t.Run("default handler can ignore via ErrIgnore", func(t *testing.T) {
		rec := newReceiver("thing.deleted").Default(func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
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
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
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
			staticVerifier{event: Event{ID: eventID, Type: "thing.created"}},
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
			staticVerifier{event: Event{ID: eventID, Type: "thing.created"}},
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
	eventID := uid.New(uid.TestPrefix)
	newReceiver := func(payload []byte) *Receiver {
		return New("test", staticVerifier{
			event: Event{ID: eventID, Type: "thing.created", Payload: payload},
		})
	}

	type thing struct {
		Name string `json:"name"`
	}

	t.Run("handler receives the parsed payload", func(t *testing.T) {
		var got thing
		payload, err := json.Marshal(thing{Name: "box"})
		require.NoError(t, err)
		rec := newReceiver(payload).On(
			[]string{"thing.created"},
			Typed(func(ctx context.Context, e Event, payload thing) error {
				got = payload
				return nil
			}),
		)
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
		require.Equal(t, "box", got.Name)
	})

	t.Run("malformed payload is a 400 bad request", func(t *testing.T) {
		rec := newReceiver([]byte(`not json`)).On(
			[]string{"thing.created"},
			Typed(func(ctx context.Context, e Event, payload thing) error {
				t.Fatal("handler must not run for unparseable payloads")
				return nil
			}),
		)
		require.Equal(t, http.StatusBadRequest, post(t, rec, "").Code)
	})
}
