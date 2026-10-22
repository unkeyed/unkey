package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type verifierFunc func(*http.Request) (Event, error)

func (fn verifierFunc) Verify(r *http.Request) (Event, error) { return fn(r) }

func post(t *testing.T, rec *Receiver, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	rec.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhooks/test", strings.NewReader(body)))
	return w
}

func TestReceiver(t *testing.T) {
	eventID := uid.New(uid.TestPrefix)
	payload, err := json.Marshal(struct{}{})
	require.NoError(t, err)
	newReceiver := func(eventType string) *Receiver {
		return New("test", staticVerifier{event: Event{ID: eventID, Type: eventType, Payload: payload}})
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
			OnResponse([]string{"thing.created"}, http.StatusCreated, func(ctx context.Context, event Event, _ struct{}) (string, error) {
				return event.ID, nil
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
				require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
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
			OnResponse([]string{"thing.created"}, http.StatusCreated, func(ctx context.Context, event Event, _ struct{}) (string, error) {
				return "must not be sent", nil
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
				require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
				require.NotContains(t, response.Body.String(), "private detail")
			})
		}
	})

	t.Run("handled event returns 200", func(t *testing.T) {
		called := false
		rec := newReceiver("thing.created").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			called = true
			require.Equal(t, eventID, e.ID)
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
		require.True(t, called)
	})

	t.Run("ignored event returns 200", func(t *testing.T) {
		rec := newReceiver("thing.created").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: not ours", ErrIgnore)
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
	})

	t.Run("handler error returns 500 so the provider retries", func(t *testing.T) {
		rec := newReceiver("thing.created").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return errors.New("downstream broke")
		})
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "").Code)
	})

	t.Run("bad request returns 400 without retry", func(t *testing.T) {
		rec := newReceiver("thing.created").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			return fmt.Errorf("%w: malformed", ErrBadRequest)
		})
		require.Equal(t, http.StatusBadRequest, post(t, rec, "").Code)
	})

	t.Run("unregistered event type is acknowledged", func(t *testing.T) {
		rec := newReceiver("thing.deleted").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for other event types")
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
	})

	t.Run("verification failure returns 401", func(t *testing.T) {
		rec := newReceiver("thing.created").OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			t.Fatal("handler must not run for unverified requests")
			return nil
		})
		require.Equal(t, http.StatusUnauthorized, post(t, rec, "tampered").Code)
	})

	t.Run("one handler serves several event types", func(t *testing.T) {
		for _, eventType := range []string{"branch.created", "branch.deleted"} {
			called := false
			rec := newReceiver(eventType).OnRaw([]string{"branch.created", "branch.deleted"}, func(ctx context.Context, e Event) error {
				called = true
				return nil
			})
			require.Equal(t, http.StatusOK, post(t, rec, "").Code)
			require.True(t, called, eventType)
		}
	})

	t.Run("OnRaw without event types panics", func(t *testing.T) {
		require.Panics(t, func() {
			newReceiver("thing.created").OnRaw(nil, func(ctx context.Context, e Event) error { return nil })
		})
	})

	t.Run("default handler takes the fallback path", func(t *testing.T) {
		var got string
		rec := newReceiver("thing.deleted").
			OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
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
			OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
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
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	})

	t.Run("body over the size limit returns 413", func(t *testing.T) {
		rec := New("test",
			staticVerifier{event: Event{ID: eventID, Type: "thing.created"}},
			WithMaxBodySize(8),
		).OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
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
		).OnRaw([]string{"thing.created"}, func(ctx context.Context, e Event) error {
			called = true
			return nil
		})
		require.Equal(t, http.StatusOK, post(t, rec, strings.Repeat("a", 64)).Code)
		require.True(t, called)
	})
}

func TestReceiverTimeout(t *testing.T) {
	for _, test := range []struct {
		name          string
		timeout       time.Duration
		parentTimeout time.Duration
	}{
		{name: "no configured deadline"},
		{name: "nonpositive timeout", timeout: -time.Second},
		{name: "shared deadline", timeout: 20 * time.Millisecond},
		{name: "earlier parent deadline", timeout: time.Second, parentTimeout: 20 * time.Millisecond},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			if test.parentTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, test.parentTimeout)
				t.Cleanup(cancel)
			}
			parentDeadline, _ := ctx.Deadline()
			var verificationDeadline, verifiedAt time.Time
			verifier := verifierFunc(func(r *http.Request) (Event, error) {
				deadline, hasDeadline := r.Context().Deadline()
				require.Equal(t, test.timeout > 0 || test.parentTimeout > 0, hasDeadline)
				verificationDeadline = deadline
				verifiedAt = time.Now()
				return Event{Type: "thing.created"}, nil
			})
			rec := New("test", verifier, WithTimeout(test.timeout)).
				OnRaw([]string{"thing.created"}, func(ctx context.Context, event Event) error {
					deadline, hasDeadline := ctx.Deadline()
					require.Equal(t, verificationDeadline, deadline)
					if !hasDeadline {
						return nil
					}
					<-ctx.Done()
					return fmt.Errorf("%w: %w", ErrUnavailable, ctx.Err())
				})

			start := time.Now()
			response := httptest.NewRecorder()
			rec.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodPost, "/", nil))
			if test.timeout > 0 {
				require.Equal(t, http.StatusServiceUnavailable, response.Code)
				if test.parentTimeout > 0 {
					require.Equal(t, parentDeadline, verificationDeadline)
				} else {
					require.False(t, verificationDeadline.Before(start.Add(test.timeout)))
					require.False(t, verificationDeadline.After(verifiedAt.Add(test.timeout)))
				}
			} else {
				require.Equal(t, http.StatusOK, response.Code)
			}
		})
	}
}

func TestReceiverOn(t *testing.T) {
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
			func(ctx context.Context, e Event, payload thing) error {
				got = payload
				return nil
			},
		)
		require.Equal(t, http.StatusOK, post(t, rec, "").Code)
		require.Equal(t, "box", got.Name)
	})

	t.Run("malformed payload is a 400 bad request", func(t *testing.T) {
		rec := newReceiver([]byte(`not json`)).On(
			[]string{"thing.created"},
			func(ctx context.Context, e Event, payload thing) error {
				t.Fatal("handler must not run for unparseable payloads")
				return nil
			},
		)
		require.Equal(t, http.StatusBadRequest, post(t, rec, "").Code)
	})
}

func TestReceiverOnResponse(t *testing.T) {
	eventID := uid.New(uid.TestPrefix)
	newReceiver := func(payload []byte) *Receiver {
		return New("test", staticVerifier{
			event: Event{ID: eventID, Type: "thing.created", Payload: payload},
		})
	}
	type request struct {
		Name string `json:"name"`
	}
	type response struct {
		Message string `json:"message"`
	}
	payload, err := json.Marshal(request{Name: "box"})
	require.NoError(t, err)

	t.Run("typed request and JSON acknowledgment", func(t *testing.T) {
		rec := newReceiver(payload).OnResponse(
			[]string{"thing.created"},
			http.StatusCreated,
			func(ctx context.Context, event Event, input request) (response, error) {
				require.Equal(t, eventID, event.ID)
				return response{Message: "created " + input.Name}, nil
			},
		)
		got := post(t, rec, "")
		require.Equal(t, http.StatusCreated, got.Code)
		require.Equal(t, "application/json", got.Header().Get("Content-Type"))
		var body response
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &body))
		require.Equal(t, response{Message: "created box"}, body)
	})

	t.Run("malformed request returns 400", func(t *testing.T) {
		rec := newReceiver([]byte("not json")).OnResponse(
			[]string{"thing.created"},
			http.StatusCreated,
			func(ctx context.Context, event Event, input request) (response, error) {
				t.Fatal("handler must not run for unparseable payloads")
				return response{}, nil
			},
		)
		require.Equal(t, http.StatusBadRequest, post(t, rec, "").Code)
	})

	t.Run("handler error discards response", func(t *testing.T) {
		rec := newReceiver(payload).OnResponse(
			[]string{"thing.created"},
			http.StatusCreated,
			func(ctx context.Context, event Event, input request) (response, error) {
				return response{Message: "must not be sent"}, fmt.Errorf("private detail: %w", ErrUnavailable)
			},
		)
		got := post(t, rec, "")
		require.Equal(t, http.StatusServiceUnavailable, got.Code)
		require.NotContains(t, got.Body.String(), "must not be sent")
		require.NotContains(t, got.Body.String(), "private detail")
	})

	t.Run("encoding error returns 500 instead of success", func(t *testing.T) {
		rec := newReceiver(payload).OnResponse(
			[]string{"thing.created"},
			http.StatusCreated,
			func(ctx context.Context, event Event, input request) (float64, error) {
				return math.Inf(1), nil
			},
		)
		require.Equal(t, http.StatusInternalServerError, post(t, rec, "").Code)
	})
}
