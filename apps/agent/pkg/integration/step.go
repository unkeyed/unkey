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

		requestBody, err := json.Marshal(s.Body)
		require.NoError(t, err)

		req, err := http.NewRequest(s.Method, s.Url, bytes.NewBuffer(requestBody))
		require.NoError(t, err)

		for k, v := range s.Header {
			req.Header.Set(k, v)
		}

		httpResponse, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer httpResponse.Body.Close()
		res.Status = httpResponse.StatusCode
		res.Header = httpResponse.Header.Clone()

		body, err := io.ReadAll(httpResponse.Body)
		require.NoError(t, err)

		err = json.Unmarshal(body, &res.Body)
		require.NoError(t, err)

	})
	return res
}
