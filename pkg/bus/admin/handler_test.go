package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/bus"
)

// TestPauseFlipsBusState pins the contract the kill switch depends on:
// POST /pause must set the bus's paused flag, observable via IsPaused.
// We use the noop bus only as a sanity-check carrier; a richer
// integration test against the real bus lives in pkg/bus.
func TestPauseFlipsBusState(t *testing.T) {
	b := &spyBus{Bus: bus.NewNoop()}
	handler := NewHandler(b)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, pathPrefix+"/pause", nil)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, b.paused, "Pause should have been called")

	var body statusBody
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	require.True(t, body.Paused)
}

func TestResumeFlipsBusState(t *testing.T) {
	b := &spyBus{Bus: bus.NewNoop(), paused: true}
	handler := NewHandler(b)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, pathPrefix+"/resume", nil)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.False(t, b.paused, "Resume should have cleared the flag")
}

func TestStatusReadsBusState(t *testing.T) {
	b := &spyBus{Bus: bus.NewNoop(), paused: true}
	handler := NewHandler(b)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, pathPrefix+"/status", nil)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"paused":true`)
}

func TestRoutesAreLoopbackPrefixed(t *testing.T) {
	// The kill-switch only works if the routes mount under the documented
	// loopback prefix. Drift here would silently leak the endpoint to a
	// path operators don't expect.
	require.True(t, strings.HasPrefix(pathPrefix, "/_unkey/internal/"))
}

// spyBus wraps bus.NewNoop and records Pause/Resume/IsPaused so tests can
// assert without spinning up a real Serf instance.
type spyBus struct {
	bus.Bus
	paused bool
}

func (s *spyBus) Pause()          { s.paused = true }
func (s *spyBus) Resume()         { s.paused = false }
func (s *spyBus) IsPaused() bool  { return s.paused }
func (s *spyBus) Members() []bus.Member {
	return []bus.Member{{NodeID: "n1", Addr: "127.0.0.1:7946", Tags: nil}}
}
