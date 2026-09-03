package logdrain

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ssrf"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// probeBody is what the sink itself would send for a batch with no events:
// an empty JSON array, or nothing at all for NDJSON. Posting exactly that
// exercises the route without adding an event to the customer's log store,
// and an endpoint that rejects it would reject the sink's empty batches too.
func probeBody(format ctrlv1.LogdrainBodyFormat) []byte {
	if format == ctrlv1.LogdrainBodyFormat_LOGDRAIN_BODY_FORMAT_NDJSON {
		return nil
	}
	return []byte("[]")
}

// VerifyLogdrainEndpoint posts an empty batch and reports the answer. See the
// proto doc for what counts as a failure and what counts as an invalid endpoint.
func (s *Service) VerifyLogdrainEndpoint(
	ctx context.Context,
	req *connect.Request[ctrlv1.VerifyLogdrainEndpointRequest],
) (*connect.Response[ctrlv1.VerifyLogdrainEndpointResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	// An endpoint that fails shape validation is a bad request, not a failed
	// check. No change on the endpoint's side would make it deliverable.
	// Forbidden addresses are caught later, by the dialer.
	if err := ssrf.ValidateEndpoint(req.Msg.GetUrl(), s.ssrfOpts...); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	contentType := "application/json"
	if req.Msg.GetFormat() == ctrlv1.LogdrainBodyFormat_LOGDRAIN_BODY_FORMAT_NDJSON {
		contentType = "application/x-ndjson"
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, req.Msg.GetUrl(), bytes.NewReader(probeBody(req.Msg.GetFormat())),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	for name, value := range req.Msg.GetHeaders() {
		httpReq.Header.Set(name, value)
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("User-Agent", "unkey-logdrain/1")
	httpReq.Header.Set("X-Unkey-Drain-Verification", "1")

	start := time.Now()
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return connect.NewResponse(&ctrlv1.VerifyLogdrainEndpointResponse{
			Ok:         false,
			Error:      err.Error(),
			DurationMs: elapsedMs(start),
		}), nil
	}

	diagnostic, err := sink.ReadDiagnostic(resp.Body)
	if err != nil {
		return connect.NewResponse(&ctrlv1.VerifyLogdrainEndpointResponse{
			Ok:             false,
			ResponseStatus: int32(resp.StatusCode),
			Error:          err.Error(),
			DurationMs:     elapsedMs(start),
		}), nil
	}

	return connect.NewResponse(&ctrlv1.VerifyLogdrainEndpointResponse{
		Ok:             resp.StatusCode >= 200 && resp.StatusCode < 300,
		ResponseStatus: int32(resp.StatusCode),
		ResponseBody:   string(bytes.TrimSpace(diagnostic)),
		DurationMs:     elapsedMs(start),
	}), nil
}

func elapsedMs(start time.Time) int32 {
	return int32(time.Since(start).Milliseconds())
}
