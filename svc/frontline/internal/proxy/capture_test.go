package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestForwardCapturePreservesStreamAndClosesUpstream(t *testing.T) {
	for _, capture := range []bool{false, true} {
		for _, readFailure := range []bool{false, true} {
			t.Run(fmt.Sprintf("capture=%t/readFailure=%t", capture, readFailure), func(t *testing.T) {
				payload := strings.Repeat("x", zen.MaxBodyCapture+19)
				var reader io.Reader = strings.NewReader(payload)
				if readFailure {
					reader = io.MultiReader(reader, failingBodyReader{})
				}
				body := &captureTestBody{Reader: reader}
				tracking := &RequestTracking{LogResponseBody: capture}
				ctx := WithRequestTracking(t.Context(), tracking)
				writer := httptest.NewRecorder()
				sess := &zen.Session{}
				require.NoError(t, sess.Init(writer, httptest.NewRequest(http.MethodGet, "http://example.test", nil), 0))
				clk := clock.NewTestClock()
				svc := &service{clock: clk}
				err := svc.forward(ctx, sess, forwardConfig{
					targetURL:    &url.URL{Scheme: "http", Host: "upstream.test"},
					startTime:    clk.Now(),
					directorFunc: func(_ *http.Request) {},
					destination:  "instance",
					transport:    captureTestTransport{body: body},
				})
				require.NoError(t, err)
				require.Equal(t, payload, writer.Body.String())
				require.Equal(t, 1, body.closes)
				if capture {
					require.Equal(t, payload[:zen.MaxBodyCapture], string(tracking.ResponseBody))
					require.LessOrEqual(t, cap(tracking.ResponseBody), zen.MaxBodyCapture)
				} else {
					require.Empty(t, tracking.ResponseBody)
				}
			})
		}
	}
}

type captureTestBody struct {
	io.Reader
	closes int
}

func (b *captureTestBody) Close() error {
	b.closes++
	return nil
}

type captureTestTransport struct{ body io.ReadCloser }

func (t captureTestTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: t.body, ContentLength: -1}, nil
}

type failingBodyReader struct{}

func (failingBodyReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
