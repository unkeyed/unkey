package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/http2"
)

// NotDeliveredError means the request never reached a pod, so another
// instance may serve it.
type NotDeliveredError struct {
	Reason string
	Err    error
}

func (e *NotDeliveredError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("peer: not delivered (%s): %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("peer: not delivered (%s)", e.Reason)
}

func (e *NotDeliveredError) Unwrap() error { return e.Err }

// IsNotDelivered reports whether err means the request never reached a pod.
func IsNotDelivered(err error) bool {
	var nd *NotDeliveredError
	return errors.As(err, &nd)
}

// RoundTripper sends requests to Region over the pool. The request must carry
// signed metadata with a target in HeaderFrontlineMeta. An upgrade request
// returns a synthetic 101 whose body is the tunnel, which is what
// httputil.ReverseProxy needs to bridge the client connection.
type RoundTripper struct {
	Pool   *Pool
	Region string
}

func (t *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	kind := kindHTTP
	if isUpgrade(req.Header) {
		kind = kindUpgrade
	}
	resp, err := t.roundTrip(req, kind)
	switch {
	case err == nil:
		streamsTotal.WithLabelValues(t.Region, kind, outcomeOK).Inc()
	case IsNotDelivered(err):
		streamsTotal.WithLabelValues(t.Region, kind, outcomeNotDelivered).Inc()
	default:
		streamsTotal.WithLabelValues(t.Region, kind, outcomeError).Inc()
	}
	return resp, err
}

func (t *RoundTripper) roundTrip(req *http.Request, kind string) (*http.Response, error) {
	cc, err := t.Pool.Get(t.Region)
	if err != nil {
		return nil, &NotDeliveredError{Reason: "no_connection", Err: err}
	}

	ctx, cancel := detach(req.Context())
	ctx = httptrace.WithClientTrace(ctx, ackTrace(req))

	start := time.Now()
	if kind == kindUpgrade {
		resp, err := t.upgrade(ctx, cancel, cc, req)
		if err == nil {
			streamOpenSeconds.WithLabelValues(t.Region, kind).Observe(time.Since(start).Seconds())
		}
		return resp, err
	}

	out := req.Clone(ctx)
	out.URL.Scheme = "http"
	out.RequestURI = ""
	resp, err := cc.RoundTrip(out)
	if err != nil {
		cancel()
		var se http2.StreamError
		if errors.As(err, &se) && se.Code == http2.ErrCodeRefusedStream {
			return nil, &NotDeliveredError{Reason: "refused_stream", Err: err}
		}
		return nil, err
	}
	if reason := resp.Header.Get(headerReject); reason != "" {
		_ = resp.Body.Close()
		cancel()
		return nil, &NotDeliveredError{Reason: reason, Err: nil}
	}
	streamOpenSeconds.WithLabelValues(t.Region, kind).Observe(time.Since(start).Seconds())
	removePeerHeaders(resp.Header)
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func (t *RoundTripper) upgrade(ctx context.Context, cancel context.CancelFunc, cc *http2.ClientConn, req *http.Request) (*http.Response, error) {
	header := req.Header.Clone()
	token := header.Get("Upgrade")
	removeHopHeaders(header)
	removePeerHeaders(header)
	header.Set(headerRequestTarget, req.URL.RequestURI())
	header.Set(headerUpgrade, token)

	pr, pw := io.Pipe()
	//nolint:exhaustruct
	out := (&http.Request{
		Method:        http.MethodConnect,
		URL:           &url.URL{Host: req.Host},
		Host:          req.Host,
		Header:        header,
		Body:          pr,
		ContentLength: -1,
	}).WithContext(ctx)

	resp, err := cc.RoundTrip(out)
	if err != nil {
		_ = pw.Close()
		cancel()
		// An upgrade carries no body, so replaying it on another instance
		// is safe even if the relay saw it.
		return nil, &NotDeliveredError{Reason: "upgrade_failed", Err: err}
	}
	if reason := resp.Header.Get(headerReject); reason != "" {
		_ = resp.Body.Close()
		_ = pw.Close()
		cancel()
		return nil, &NotDeliveredError{Reason: reason, Err: nil}
	}
	status, err := strconv.Atoi(resp.Header.Get(headerUpstreamStatus))
	if err != nil {
		_ = resp.Body.Close()
		_ = pw.Close()
		cancel()
		return nil, fmt.Errorf("peer: relay response without upstream status")
	}
	upgradedTo := resp.Header.Get(headerUpgrade)
	removePeerHeaders(resp.Header)

	if status != http.StatusSwitchingProtocols {
		_ = pw.Close()
		resp.StatusCode = status
		resp.Status = fmt.Sprintf("%d %s", status, http.StatusText(status))
		resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
		return resp, nil
	}

	resp.Header.Set("Connection", "Upgrade")
	resp.Header.Set("Upgrade", upgradedTo)
	//nolint:exhaustruct
	return &http.Response{
		Status:        "101 Switching Protocols",
		StatusCode:    http.StatusSwitchingProtocols,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        resp.Header,
		Body:          &tunnel{r: resp.Body, w: pw, cancel: cancel},
		ContentLength: -1,
		Request:       req,
	}, nil
}

// detach returns a context that is cancelled with parent but does not carry
// its values. The parent's client trace would otherwise see, and forward to
// the client, the relay's private 1xx acknowledgement.
func detach(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(context.Background())
	stop := context.AfterFunc(parent, func() { cancel(context.Cause(parent)) })
	return ctx, func() {
		stop()
		cancel(context.Canceled)
	}
}

// ackTrace acknowledges the replayable body on the relay's dial ack and passes
// every other informational response to the caller's trace.
func ackTrace(req *http.Request) *httptrace.ClientTrace {
	parent := httptrace.ContextClientTrace(req.Context())
	attempt, _ := req.Body.(*replayAttempt)
	//nolint:exhaustruct
	return &httptrace.ClientTrace{
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			if code == statusDialed {
				if attempt != nil {
					attempt.Ack()
				}
				return nil
			}
			if parent != nil && parent.Got1xxResponse != nil {
				return parent.Got1xxResponse(code, header)
			}
			return nil
		},
	}
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// tunnel is the upgraded byte stream: reads come from the relay's response,
// writes go into the CONNECT request body.
type tunnel struct {
	r      io.ReadCloser
	w      *io.PipeWriter
	cancel context.CancelFunc
}

func (t *tunnel) Read(p []byte) (int, error)  { return t.r.Read(p) }
func (t *tunnel) Write(p []byte) (int, error) { return t.w.Write(p) }

func (t *tunnel) Close() error {
	_ = t.w.Close()
	err := t.r.Close()
	t.cancel()
	return err
}
