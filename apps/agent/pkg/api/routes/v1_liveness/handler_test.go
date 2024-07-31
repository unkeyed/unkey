package handler_test

import (
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	v1Liveness "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_liveness"
)

func TestLiveness(t *testing.T) {
	_, api := humatest.New(t)

	v1Liveness.Register(api, routes.Services{})

	resp := api.Get("/v1/liveness")

	require.Equal(t, 200, resp.Code)
	require.JSONEq(t, `{"message":"OK"}`, resp.Body.String())
}
