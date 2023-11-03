package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type Step[R any] struct {
	Debug  bool
	Name   string
	Body   map[string]any
	Url    string
	Method string
	Header map[string]string
}

type StepResponse[R any] struct {
	Status int
	Header http.Header
	Body   R
}

func (s Step[R]) Run(t *testing.T) StepResponse[R] {
	t.Helper()

	var res StepResponse[R]

	t.Run(s.Name, func(t *testing.T) {
		if s.Debug {
			t.Log("debugging enabled")
		}

		requestBody, err := json.Marshal(s.Body)
		require.NoError(t, err)

		req, err := http.NewRequest(s.Method, s.Url, bytes.NewBuffer(requestBody))
		require.NoError(t, err)

		for k, v := range s.Header {
			req.Header.Set(k, v)
		}
		if s.Debug {
			t.Logf("request: %+v", req)
		}
		httpResponse, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer httpResponse.Body.Close()
		res.Status = httpResponse.StatusCode
		res.Header = httpResponse.Header.Clone()

		body, err := io.ReadAll(httpResponse.Body)
		require.NoError(t, err)
		if s.Debug {
			t.Logf("response body: %s", string(body))
		}
		err = json.Unmarshal(body, &res.Body)
		require.NoError(t, err)

	})
	return res
}
