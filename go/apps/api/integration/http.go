package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"

	"github.com/stretchr/testify/require"
)

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    TBody
	RawBody string
}

func CallNode[Req any, Res any](h *Harness, node ClusterNode, method string, path string, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()

	url := fmt.Sprintf("http://localhost:%d%s", node.HttpPort, path)

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(h.t, err)

	httpReq, err := http.NewRequest(method, url, body)
	require.NoError(h.t, err)

	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}

	httpRes, err := http.DefaultClient.Do(httpReq)
	require.NoError(h.t, err)
	defer httpRes.Body.Close()

	resBody, err := io.ReadAll(httpRes.Body)
	require.NoError(h.t, err)

	var res Res
	err = json.Unmarshal(resBody, &res)
	require.NoError(h.t, err, fmt.Sprintf("failed to decode response body: %s", string(resBody)))

	return TestResponse[Res]{
		Status:  httpRes.StatusCode,
		Headers: httpRes.Header,
		Body:    res,
		RawBody: string(resBody),
	}
}

func CallRandomNode[Req any, Res any](h *Harness, method string, path string, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()
	// nolint:gosec
	node := h.nodes[rand.IntN(len(h.nodes))]
	return CallNode[Req, Res](h, node, method, path, headers, req)

}
