package resilience

import (
	"net/http"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/hoptracing"
)

type InstrumentedTransport struct {
	base    http.RoundTripper
	tracker Tracker
	clock   clock.Clock
}

func NewInstrumentedTransport(base http.RoundTripper, tracker Tracker, clk clock.Clock) http.RoundTripper {
	if tracker == nil {
		return base
	}
	return &InstrumentedTransport{
		base:    base,
		tracker: tracker,
		clock:   clk,
	}
}

func (t *InstrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	region, ok := KeyFromContext(req.Context())
	if !ok {
		return t.base.RoundTrip(req)
	}

	resp, err := t.base.RoundTrip(req)
	now := t.clock.Now()

	if err != nil {
		t.tracker.Observe(region, now, Outcome{NetErr: err, StatusCode: 0, IsInfrastructure: true})
		return nil, err
	}

	isInfrastructure := resp.Header.Get(hoptracing.HeaderErrorSource) == "sentinel"

	t.tracker.Observe(region, now, Outcome{NetErr: nil, StatusCode: resp.StatusCode, IsInfrastructure: isInfrastructure})
	return resp, nil
}
