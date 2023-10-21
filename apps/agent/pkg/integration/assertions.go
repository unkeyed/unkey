package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PaesslerAG/gval"
	"github.com/stretchr/testify/require"
)

type AssertRequest struct {
	Status int
	Header http.Header
	Body   string
}

type Assertion func(req AssertRequest)

func AssertHeaderExists(t *testing.T, key string) Assertion {
	return func(req AssertRequest) {
		t.Helper()
		require.NotEmpty(t, req.Header.Get(key), "header %s does not exist", key)

	}
}

func AssertStatus(t *testing.T, status int) Assertion {
	return func(req AssertRequest) {
		t.Helper()
		require.Equal(t, status, req.Status, "status %d does not match %d", req.Status, status)

	}
}

func AssertBody[T comparable](t *testing.T, expr string, value T) Assertion {
	return func(req AssertRequest) {
		t.Helper()
		var data interface{}
		err := json.Unmarshal([]byte(req.Body), &data)
		require.NoError(t, err)

		actual, err := gval.Evaluate(expr, data)
		if err != nil {
			t.Logf("expr: %s", expr)
			t.Logf("body: %s", req.Body)
		}
		require.NoError(t, err, "data: %s", req.Body)
		require.Equal(t, value, actual)

	}
}

func AssertBodyExists(t *testing.T, expr string) Assertion {
	return func(req AssertRequest) {
		t.Helper()
		var data interface{}
		err := json.Unmarshal([]byte(req.Body), &data)
		require.NoError(t, err)
		actual, err := gval.Evaluate(expr, data)
		require.NoError(t, err)

		require.NotNil(t, actual, "expr %s does not exist", expr)
	}
}
