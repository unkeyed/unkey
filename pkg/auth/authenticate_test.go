package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
)

// fakeResolver returns canned values from Resolve. Tracks invocation count
// so tests can assert chain ordering and short-circuiting.
type fakeResolver struct {
	principal *Principal
	err       error
	calls     int
}

func (f *fakeResolver) Resolve(ctx context.Context, sess *zen.Session) (*Principal, Emit, error) {
	f.calls++
	if f.err != nil {
		return nil, EmptyEmit, f.err
	}
	if f.principal == nil {
		return nil, nil, nil
	}
	return f.principal, EmptyEmit, nil
}

// newSession builds a zen.Session ready for tests. Authenticate reads/writes
// WorkspaceID and reads RequestID; both work after Init.
func newSession(t *testing.T) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))
	return sess
}

func TestAuthenticate_NoResolvers_ReturnsMissingCredentials(t *testing.T) {
	p, emit, err := New().Authenticate(context.Background(), newSession(t))

	require.Nil(t, p)
	require.NotNil(t, emit) // EmptyEmit, never nil so handlers can defer it
	require.Error(t, err)
}

func TestAuthenticate_FirstResolverMatches_ReturnsItsPrincipal(t *testing.T) {
	want := &Principal{Scheme: SchemeRootKey, WorkspaceID: "ws_1", ID: "key_a"}
	first := &fakeResolver{principal: want}
	second := &fakeResolver{}
	sess := newSession(t)

	got, _, err := New(first, second).Authenticate(context.Background(), sess)

	require.NoError(t, err)
	require.Same(t, want, got)
	require.Equal(t, "ws_1", sess.WorkspaceID, "must propagate workspace to session")
	require.Equal(t, 1, first.calls)
	require.Equal(t, 0, second.calls, "chain must short-circuit on first match")
}

func TestAuthenticate_SkipsUnmatched_TriesNext(t *testing.T) {
	want := &Principal{Scheme: SchemeJWT, WorkspaceID: "ws_2"}
	first := &fakeResolver{} // returns (nil, nil, nil)
	second := &fakeResolver{principal: want}

	got, _, err := New(first, second).Authenticate(context.Background(), newSession(t))

	require.NoError(t, err)
	require.Same(t, want, got)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
}

func TestAuthenticate_AllUnmatched_ReturnsMissingCredentials(t *testing.T) {
	first := &fakeResolver{}
	second := &fakeResolver{}

	_, _, err := New(first, second).Authenticate(context.Background(), newSession(t))

	require.Error(t, err)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
}

// TestAuthenticate_MatchedWithError stops the chain immediately. Falling
// through would let a different scheme produce a misleading error for what
// was clearly an attempt at this scheme's credential format.
func TestAuthenticate_MatchedWithError_StopsChainAndReturnsError(t *testing.T) {
	boom := errors.New("token invalid")
	first := &fakeResolver{err: boom}
	second := &fakeResolver{principal: &Principal{WorkspaceID: "ws_x"}}

	_, _, err := New(first, second).Authenticate(context.Background(), newSession(t))

	require.ErrorIs(t, err, boom)
	require.Equal(t, 0, second.calls, "chain must not continue past a matched-but-failed resolver")
}

// nilEmitResolver returns a Principal with a nil emit, exercising the
// Authenticator contract that handlers can defer emit() unconditionally.
type nilEmitResolver struct{ p *Principal }

func (r *nilEmitResolver) Resolve(ctx context.Context, sess *zen.Session) (*Principal, Emit, error) {
	return r.p, nil, nil
}

func TestAuthenticate_NormalizesNilEmit(t *testing.T) {
	r := &nilEmitResolver{p: &Principal{WorkspaceID: "ws_5"}}

	_, emit, err := New(r).Authenticate(context.Background(), newSession(t))

	require.NoError(t, err)
	require.NotNil(t, emit, "Authenticate must never return a nil Emit so handlers can defer it unconditionally")
	emit() // calling it must not panic
}
