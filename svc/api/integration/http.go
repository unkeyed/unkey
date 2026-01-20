package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    TBody
	RawBody string
}

type loadbalancer struct {
	mu      sync.RWMutex
	metrics map[string]int
	buffer  chan string
	h       *Harness
}

func NewLoadbalancer(h *Harness) *loadbalancer {
	lb := &loadbalancer{
		mu:      sync.RWMutex{},
		metrics: make(map[string]int),
		buffer:  make(chan string, 1_000_000),
		h:       h,
	}

	go func() {
		for host := range lb.buffer {
			lb.mu.Lock()
			lb.metrics[host]++
			lb.mu.Unlock()

		}
	}()

	return lb
}

func (lb *loadbalancer) GetMetrics() map[string]int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.metrics
}

func CallNode[Req any, Res any](t *testing.T, addr, method string, path string, headers http.Header, req Req) (TestResponse[Res], error) {
	t.Helper()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	if err != nil {
		return TestResponse[Res]{}, err
	}

	url := addr
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		url = fmt.Sprintf("http://%s", addr)
	}
	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", url, path), body)
	if err != nil {
		return TestResponse[Res]{}, err
	}

	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}

	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return TestResponse[Res]{}, err
	}
	defer func() { _ = httpRes.Body.Close() }()

	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return TestResponse[Res]{}, err
	}

	var res Res
	err = json.Unmarshal(resBody, &res)
	if err != nil {
		return TestResponse[Res]{}, err
	}

	return TestResponse[Res]{
		Status:  httpRes.StatusCode,
		Headers: httpRes.Header,
		Body:    res,
		RawBody: string(resBody),
	}, nil
}

func CallRandomNode[Req any, Res any](lb *loadbalancer, method string, path string, headers http.Header, req Req) (TestResponse[Res], error) {
	lb.h.t.Helper()
	// nolint:gosec
	addr := lb.h.instanceAddrs[rand.IntN(len(lb.h.instanceAddrs))]
	lb.buffer <- addr
	return CallNode[Req, Res](lb.h.t, addr, method, path, headers, req)

}
