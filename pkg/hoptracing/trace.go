package hoptracing

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Trace struct {
	TraceID  string
	Route    []Hop
	Timing   []TimingMetric
	HopCount int
}

func ParseFromRequest(req *http.Request) Trace {
	t := Trace{
		TraceID:  req.Header.Get(HeaderTraceID),
		Route:    parseRoute(req.Header.Get(HeaderRoute)),
		Timing:   parseTiming(req.Header.Get(HeaderTiming)),
		HopCount: 0,
	}

	if hopStr := req.Header.Get(HeaderHopCount); hopStr != "" {
		if count, err := strconv.Atoi(hopStr); err == nil {
			t.HopCount = count
		}
	}

	return t
}

func (t *Trace) AppendHop(kind HopKind, region, id string) {
	t.Route = append(t.Route, Hop{Kind: kind, Region: region, ID: id})
}

func (t *Trace) AddTiming(kind HopKind, region, id string, durationMs int64) {
	t.Timing = append(t.Timing, TimingMetric{
		Kind:       kind,
		Region:     region,
		ID:         id,
		DurationMs: durationMs,
	})
}

func (t *Trace) IncrementHopCount() error {
	t.HopCount++
	if t.HopCount > MaxHops {
		return fmt.Errorf("exceeded maximum hop count: %d", MaxHops)
	}
	return nil
}

func (t Trace) RouteString() string {
	if len(t.Route) == 0 {
		return ""
	}
	parts := make([]string, len(t.Route))
	for i, h := range t.Route {
		parts[i] = h.String()
	}
	return strings.Join(parts, ", ")
}

func (t Trace) TimingString(totalMs int64) string {
	if len(t.Timing) == 0 && totalMs == 0 {
		return ""
	}
	parts := make([]string, 0, len(t.Timing)+1)
	for _, m := range t.Timing {
		parts = append(parts, m.String())
	}
	if totalMs > 0 {
		parts = append(parts, fmt.Sprintf("unkey_total;dur=%d", totalMs))
	}
	return strings.Join(parts, ", ")
}

func (t Trace) InjectDownstream(req *http.Request, deploymentID string) {
	if t.TraceID != "" {
		req.Header.Set(HeaderTraceID, t.TraceID)
	}
	if route := t.RouteString(); route != "" {
		req.Header.Set(HeaderRoute, route)
	}
	req.Header.Set(HeaderHopCount, strconv.Itoa(t.HopCount))

	if deploymentID != "" {
		req.Header.Set(HeaderDeploymentID, deploymentID)
	}
}

func (t Trace) InjectResponse(h http.Header, totalMs int64) {
	if t.TraceID != "" {
		h.Set(HeaderTraceID, t.TraceID)
	}
	if route := t.RouteString(); route != "" {
		h.Set(HeaderRoute, route)
	}
	if timing := t.TimingString(totalMs); timing != "" {
		h.Set(HeaderTiming, timing)
	}
}

func MergeDownstreamTiming(t *Trace, downstreamHeader string) {
	if downstreamHeader == "" {
		return
	}
	parts := strings.Split(downstreamHeader, ", ")
	for _, p := range parts {
		if strings.HasPrefix(p, "unkey_total") {
			continue
		}
		if m, ok := parseTimingMetric(p); ok {
			t.Timing = append(t.Timing, m)
		}
	}
}

func MergeDownstreamRoute(t *Trace, downstreamHeader string) {
	downstream := parseRoute(downstreamHeader)
	for _, h := range downstream {
		found := false
		for _, existing := range t.Route {
			if existing.Kind == h.Kind && existing.Region == h.Region && existing.ID == h.ID {
				found = true
				break
			}
		}
		if !found {
			t.Route = append(t.Route, h)
		}
	}
}
