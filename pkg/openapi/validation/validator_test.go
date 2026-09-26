package validation

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const streamingSpec = `openapi: 3.0.3
info:
  title: Streaming validation
  version: 1.0.0
paths:
  /validated:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: OK
  /stream:
    post:
      responses:
        '200':
          description: OK
`

func TestSlowBodyDoesNotBlockOtherRequests(t *testing.T) {
	for _, blockClose := range []bool{false, true} {
		name := "read"
		if blockClose {
			name = "close"
		}
		t.Run(name, func(t *testing.T) {
			v, err := NewFromBytes([]byte(streamingSpec))
			require.NoError(t, err)

			started := make(chan struct{})
			release := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			body := &blockingBody{
				body:       io.NopCloser(strings.NewReader(`{}`)),
				blockClose: blockClose,
				block: sync.OnceFunc(func() {
					close(started)
					<-release
				}),
			}
			slow := httptest.NewRequest(http.MethodPost, "/validated", body)
			slow.Header.Set("Content-Type", "application/json")
			slowResult := make(chan *Result, 1)
			var workers sync.WaitGroup
			t.Cleanup(func() {
				unblock()
				workers.Wait()
			})
			workers.Go(func() { slowResult <- v.Validate(slow) })

			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("validation did not reach body I/O")
			}

			fast := httptest.NewRequest(http.MethodPost, "/validated", strings.NewReader(`{}`))
			fast.Header.Set("Content-Type", "application/json")
			fastResult := make(chan *Result, 1)
			workers.Go(func() { fastResult <- v.Validate(fast) })
			select {
			case result := <-fastResult:
				require.Nil(t, result)
			case <-time.After(5 * time.Second):
				t.Fatal("slow body blocked another request's validation")
			}

			unblock()
			require.Nil(t, <-slowResult)
			for _, request := range []*http.Request{slow, fast} {
				replayed, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				require.Equal(t, `{}`, string(replayed))
				require.NoError(t, request.Body.Close())
			}
		})
	}
}

func TestUnvalidatedBodyRemainsStreaming(t *testing.T) {
	v, err := NewFromBytes([]byte(streamingSpec))
	require.NoError(t, err)
	body := &blockingBody{
		body:       io.NopCloser(strings.NewReader("stream")),
		blockClose: false,
		block:      func() { t.Fatal("validator read an unvalidated body") },
	}
	r := httptest.NewRequest(http.MethodPost, "/stream", body)
	r.Header.Set("Content-Type", "application/json")
	require.Nil(t, v.Validate(r))
	require.Same(t, body, r.Body)
	require.NoError(t, r.Body.Close())
}

type blockingBody struct {
	body       io.ReadCloser
	blockClose bool
	block      func()
}

func (b *blockingBody) Read(p []byte) (int, error) {
	if !b.blockClose {
		b.block()
	}
	return b.body.Read(p)
}

func (b *blockingBody) Close() error {
	if b.blockClose {
		b.block()
	}
	return b.body.Close()
}
