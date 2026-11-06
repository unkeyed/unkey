package keys

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestDescribeKey(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  string
		want KeyShape
	}{
		{name: "prefixed base58", key: "prod_3ZbWnLyqhUSv6fsfa2AzNdh8M", want: KeyShape{Length: 30, Prefix: "prod", Charset: "base58"}},
		{name: "hex without prefix", key: "0123456789abcdef0123", want: KeyShape{Length: 20, Prefix: "", Charset: "hex"}},
		{name: "base58 excludes zero", key: "abc0defghijkmnopqrst", want: KeyShape{Length: 20, Prefix: "", Charset: "alnum"}},
		{name: "underscore in body", key: "sk_live_abcdefghijklmnop", want: KeyShape{Length: 24, Prefix: "sk", Charset: "base64url"}},
		{name: "short key keeps no prefix", key: "ab_cdefghijk", want: KeyShape{Length: 12, Prefix: "", Charset: "base64url"}},
		{name: "prefix longer than 8", key: "verylongpre_abcdefghijklmnop", want: KeyShape{Length: 28, Prefix: "", Charset: "base64url"}},
		{name: "prefix with symbols", key: "a-b_cdefghijkmnopqrs", want: KeyShape{Length: 20, Prefix: "", Charset: "base64url"}},
		{name: "leading underscore", key: "_abcdefghijkmnopqrst", want: KeyShape{Length: 20, Prefix: "", Charset: "base64url"}},
		{name: "prefix only", key: "prod_", want: KeyShape{Length: 5, Prefix: "", Charset: "base64url"}},
		{name: "empty body after prefix", key: "prod______________", want: KeyShape{Length: 18, Prefix: "prod", Charset: "base64url"}},
		{name: "other characters", key: "key with spaces and !", want: KeyShape{Length: 21, Prefix: "", Charset: "other"}},
		{name: "empty", key: "", want: KeyShape{Length: 0, Prefix: "", Charset: "empty"}},
		{name: "length capped", key: strings.Repeat("a", 70_000), want: KeyShape{Length: 65535, Prefix: "", Charset: "hex"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DescribeKey(tt.key))
		})
	}
}

type failureSink struct {
	mu   sync.Mutex
	rows []schema.KeyVerificationFailure
}

func newFailureRecorder(t *testing.T, clk clock.Clock, rowsPerSecond int) (*FailureRecorder, func() []schema.KeyVerificationFailure) {
	t.Helper()

	sink := &failureSink{}
	buffer := batch.New(batch.Config[schema.KeyVerificationFailure]{
		Name:          "key_verification_failures_test",
		Drop:          false,
		BatchSize:     1,
		BufferSize:    10_000,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, rows []schema.KeyVerificationFailure) {
			sink.mu.Lock()
			defer sink.mu.Unlock()
			sink.rows = append(sink.rows, rows...)
		},
	})

	r := NewFailureRecorder(FailureRecorderConfig{
		Buffer:        buffer,
		RowsPerSecond: rowsPerSecond,
		Clock:         clk,
	})

	return r, func() []schema.KeyVerificationFailure {
		buffer.Close()
		sink.mu.Lock()
		defer sink.mu.Unlock()
		return sink.rows
	}
}

func TestFailureRecorder_RecordsKeyShape(t *testing.T) {
	const rawKey = "prod_3ZbWnLyqhUSv6fsfa2AzNdh8M"

	r, rows := newFailureRecorder(t, clock.NewTestClock(), 100)
	r.Record(schema.KeyVerificationFailure{WorkspaceID: "ws_a"}, rawKey) //nolint:exhaustruct

	got := rows()
	require.Len(t, got, 1)
	require.Equal(t, uint16(len(rawKey)), got[0].KeyLength)
	require.Equal(t, "prod", got[0].KeyPrefix)
	require.Equal(t, "base58", got[0].KeyCharset)
	require.Equal(t, uint32(1), got[0].SampleWeight)
}

func TestFailureRecorder_SamplesAboveCap(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	r, rows := newFailureRecorder(t, clk, 10)

	// 10 full-weight rows, then 250 over the cap of which every 100th is kept.
	for range 260 {
		r.Record(schema.KeyVerificationFailure{WorkspaceID: "ws_a"}, "prod_abcdefghijkmnopqrs") //nolint:exhaustruct
	}
	// A new second resets the cap but not the overflow sampling.
	clk.Tick(time.Second)
	for range 5 {
		r.Record(schema.KeyVerificationFailure{WorkspaceID: "ws_a"}, "prod_abcdefghijkmnopqrs") //nolint:exhaustruct
	}

	weights := map[uint32]int{}
	var total uint64
	for _, row := range rows() {
		weights[row.SampleWeight]++
		total += uint64(row.SampleWeight)
	}
	require.Equal(t, map[uint32]int{1: 15, 100: 2}, weights)
	require.Equal(t, uint64(215), total)
}

func TestFailureRecorder_NeverStoresRawKeyOrHash(t *testing.T) {
	rawKey := "prod_3ZbWnLyqhUSv6fsfa2AzNdh8M"
	sum := sha256.Sum256([]byte(rawKey))
	secrets := []string{
		rawKey,
		rawKey[len("prod_"):],
		base64.StdEncoding.EncodeToString(sum[:]),
		hex.EncodeToString(sum[:]),
	}

	r, rows := newFailureRecorder(t, clock.NewTestClock(), 100)
	r.Record(schema.KeyVerificationFailure{WorkspaceID: "ws_a"}, rawKey) //nolint:exhaustruct

	got := rows()
	require.Len(t, got, 1)
	encoded, err := json.Marshal(got[0])
	require.NoError(t, err)
	for _, secret := range secrets {
		require.NotContains(t, string(encoded), secret)
	}
}
