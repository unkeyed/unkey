package logfilter

import (
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

const (
	testPath   Column = "path"
	testHost   Column = "host"
	testStatus Column = "response_status"
)

func TestBuildCombinesOperators(t *testing.T) {
	b := New()
	b.Text(testPath, Text{
		StartsWith:  []string{"/v2/keys", "/v3/keys"},
		NotContains: []string{"verifyKey"},
	})
	b.Exact(testHost, Exact{IsNot: []string{"KEBAP"}})
	b.Int(testStatus, Int{Is: []int64{404}, IsNot: []int64{429}, Gte: new(int64(400)), Lt: new(int64(500))})

	where, params := b.Build()

	require.Equal(t,
		"(startsWith(path, {filter_0:String}) OR startsWith(path, {filter_1:String}))"+
			" AND position(path, {filter_2:String}) = 0"+
			" AND host != {filter_3:String}"+
			" AND (response_status = {filter_4:Int64})"+
			" AND response_status != {filter_5:Int64}"+
			" AND response_status >= {filter_6:Int64}"+
			" AND response_status < {filter_7:Int64}",
		where,
	)
	require.Equal(t, ch.Parameters{
		"filter_0": "/v2/keys",
		"filter_1": "/v3/keys",
		"filter_2": "verifyKey",
		"filter_3": "KEBAP",
		"filter_4": "404",
		"filter_5": "429",
		"filter_6": "400",
		"filter_7": "500",
	}, params)
}

func TestBuildWithoutFilters(t *testing.T) {
	b := New()
	b.Text(testPath, Text{})

	where, params := b.Build()

	require.Equal(t, "true", where)
	require.Empty(t, params)
}

// FuzzWhereIgnoresValues proves that request values never reach the query
// text: for a fixed filter shape, every value yields the same WHERE body.
func FuzzWhereIgnoresValues(f *testing.F) {
	fuzz.Seed(f)

	build := func(s string, n int64) string {
		b := New()
		b.Text(testPath, Text{
			Is:          []string{s},
			IsNot:       []string{s},
			Contains:    []string{s},
			NotContains: []string{s},
			StartsWith:  []string{s},
			EndsWith:    []string{s},
		})
		b.Exact(testHost, Exact{Is: []string{s}, IsNot: []string{s}})
		b.Int(testStatus, Int{Is: []int64{n}, IsNot: []int64{n}, Gt: &n, Gte: &n, Lt: &n, Lte: &n})
		where, _ := b.Build()

		return where
	}
	baseline := build("KEBAP", 200)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		require.Equal(t, baseline, build(c.String(), c.Int64()))
	})
}
