package engine

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
)

func TestMatchesRequest_EmptyList(t *testing.T) {
	t.Parallel()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/api"}, Header: http.Header{}}
	matched, err := matchesRequest(req, nil, newRegexCache())
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_PathExact(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/api/v1"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Exact{Exact: "/api/v1"}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_PathExactMismatch(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/api/v2"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Exact{Exact: "/api/v1"}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestMatchesRequest_PathPrefix(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/api/v1/users"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/api/v1"}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_PathRegex(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/api/v2/users/123"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Regex{Regex: `^/api/v\d+/users/\d+$`}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_PathCaseInsensitive(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/API/V1"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{
				IgnoreCase: true,
				Match:      &sentinelv1.StringMatch_Exact{Exact: "/api/v1"},
			},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_MethodMatch(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	tests := []struct {
		name     string
		method   string
		methods  []string
		expected bool
	}{
		{"exact match", "GET", []string{"GET"}, true},
		{"case insensitive", "get", []string{"GET"}, true},
		{"multiple methods", "POST", []string{"GET", "POST"}, true},
		{"no match", "DELETE", []string{"GET", "POST"}, false},
		{"empty methods matches all", "DELETE", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := &http.Request{Method: tt.method, URL: &url.URL{Path: "/"}, Header: http.Header{}}

			//nolint:exhaustruct
			exprs := []*sentinelv1.MatchExpr{
				{Expr: &sentinelv1.MatchExpr_Method{Method: &sentinelv1.MethodMatch{Methods: tt.methods}}},
			}

			matched, err := matchesRequest(req, exprs, rc)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestMatchesRequest_HeaderPresent(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/"}, Header: http.Header{}}
	req.Header.Set("Authorization", "Bearer token")

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Header{Header: &sentinelv1.HeaderMatch{
			Name:  "Authorization",
			Match: &sentinelv1.HeaderMatch_Present{Present: true},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_HeaderNotPresent(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Header{Header: &sentinelv1.HeaderMatch{
			Name:  "Authorization",
			Match: &sentinelv1.HeaderMatch_Present{Present: true},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestMatchesRequest_HeaderValue(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/"}, Header: http.Header{}}
	req.Header.Set("Content-Type", "application/json")

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Header{Header: &sentinelv1.HeaderMatch{
			Name: "Content-Type",
			Match: &sentinelv1.HeaderMatch_Value{Value: &sentinelv1.StringMatch{
				Match: &sentinelv1.StringMatch_Exact{Exact: "application/json"},
			}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_QueryParamPresent(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/", RawQuery: "debug=true"},
		Header: http.Header{},
	}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_QueryParam{QueryParam: &sentinelv1.QueryParamMatch{
			Name:  "debug",
			Match: &sentinelv1.QueryParamMatch_Present{Present: true},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_QueryParamValue(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/", RawQuery: "version=v2"},
		Header: http.Header{},
	}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_QueryParam{QueryParam: &sentinelv1.QueryParamMatch{
			Name: "version",
			Match: &sentinelv1.QueryParamMatch_Value{Value: &sentinelv1.StringMatch{
				Match: &sentinelv1.StringMatch_Prefix{Prefix: "v"},
			}},
		}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchesRequest_ANDSemantics(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	// Path matches but method doesn't
	req := &http.Request{Method: "DELETE", URL: &url.URL{Path: "/api/v1"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/api"}},
		}}},
		{Expr: &sentinelv1.MatchExpr_Method{Method: &sentinelv1.MethodMatch{Methods: []string{"GET", "POST"}}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestMatchesRequest_ANDSemanticsAllMatch(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	req := &http.Request{Method: "POST", URL: &url.URL{Path: "/api/v1"}, Header: http.Header{}}

	//nolint:exhaustruct
	exprs := []*sentinelv1.MatchExpr{
		{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
			Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/api"}},
		}}},
		{Expr: &sentinelv1.MatchExpr_Method{Method: &sentinelv1.MethodMatch{Methods: []string{"GET", "POST"}}}},
	}

	matched, err := matchesRequest(req, exprs, rc)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestRegexCache(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	re1, err := rc.get(`^/api/v\d+$`)
	require.NoError(t, err)
	assert.True(t, re1.MatchString("/api/v1"))

	// Second call should return cached regex
	re2, err := rc.get(`^/api/v\d+$`)
	require.NoError(t, err)
	assert.Equal(t, re1, re2)
}

func TestRegexCache_InvalidPattern(t *testing.T) {
	t.Parallel()
	rc := newRegexCache()

	_, err := rc.get(`[invalid`)
	assert.Error(t, err)
}
