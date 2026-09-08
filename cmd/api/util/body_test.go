package util

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/operations"
)

func TestSendBody(t *testing.T) {
	type request struct {
		Name  string `json:"name"`
		Limit int64  `json:"limit"`
	}
	type response struct {
		ID string
	}

	t.Run("typed valid body", func(t *testing.T) {
		got, err := SendBody(context.Background(), func(_ context.Context, req request, _ ...operations.Option) (*response, error) {
			require.Equal(t, request{Name: "production", Limit: 42}, req)
			return &response{ID: "res_123"}, nil
		}, `{"name":"production","limit":42}`)
		require.NoError(t, err)
		require.Equal(t, &response{ID: "res_123"}, got)
	})

	t.Run("malformed body is not sent", func(t *testing.T) {
		called := false
		got, err := SendBody(context.Background(), func(_ context.Context, _ request, _ ...operations.Option) (*response, error) {
			called = true
			return &response{}, nil
		}, `{"name":`)
		require.ErrorContains(t, err, "invalid JSON for --body")
		require.False(t, called)
		require.Nil(t, got)
	})

	t.Run("unknown field is not sent", func(t *testing.T) {
		called := false
		got, err := SendBody(context.Background(), func(_ context.Context, _ request, _ ...operations.Option) (*response, error) {
			called = true
			return &response{}, nil
		}, `{"name":"production","limti":42}`)
		require.ErrorContains(t, err, `invalid JSON for --body: json: unknown field "limti"`)
		require.False(t, called)
		require.Nil(t, got)
	})

	t.Run("multiple values are not sent", func(t *testing.T) {
		called := false
		got, err := SendBody(context.Background(), func(_ context.Context, _ request, _ ...operations.Option) (*response, error) {
			called = true
			return &response{}, nil
		}, `{} {}`)
		require.EqualError(t, err, "invalid JSON for --body: multiple JSON values")
		require.False(t, called)
		require.Nil(t, got)
	})

	t.Run("top-level null is not sent", func(t *testing.T) {
		called := false
		got, err := SendBody(context.Background(), func(_ context.Context, _ request, _ ...operations.Option) (*response, error) {
			called = true
			return &response{}, nil
		}, `null`)
		require.EqualError(t, err, "invalid JSON for --body: top-level null is not a request body")
		require.False(t, called)
		require.Nil(t, got)
	})

	t.Run("SDK error formatting", func(t *testing.T) {
		got, err := SendBody(context.Background(), func(_ context.Context, _ request, _ ...operations.Option) (*response, error) {
			return nil, errors.New("request failed")
		}, `{}`)
		require.EqualError(t, err, "request failed")
		require.Nil(t, got)
	})
}
