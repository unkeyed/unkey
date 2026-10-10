package resolver

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func newTestForwardCache(t *testing.T, capacity int, clk clock.Clock) *forwardCache {
	t.Helper()
	c, err := newForwardCache(capacity, clk)
	require.NoError(t, err)
	t.Cleanup(c.close)
	return c
}

func TestForwardCacheTTLExpiryAndOwnership(t *testing.T) {
	now := time.Unix(100, 0)
	clk := clock.NewTestClock(now)
	cache := newTestForwardCache(t, MinForwardCacheEntries, clk)
	query := dnswire.NewMsg("first.example.", dnswire.TypeA)
	response := cachedAResponse("first.example.", 3)
	response.ID = 42
	require.NoError(t, cache.set(t.Context(), query, response))
	require.EqualValues(t, 42, response.ID)

	response.Answer[0].Header().TTL = 99
	clk.Set(now.Add(1500 * time.Millisecond))
	first := cache.get(t.Context(), query, false)
	require.Zero(t, first.ID)
	require.Equal(t, uint32(2), first.Answer[0].Header().TTL)
	first.Answer[0].Header().TTL = 77
	require.Equal(t, uint32(2), cache.get(t.Context(), query, false).Answer[0].Header().TTL)
	clk.Set(now.Add(3 * time.Second))
	require.Nil(t, cache.get(t.Context(), query, false))
	require.Equal(t, uint32(0), cache.get(t.Context(), query, true).Answer[0].Header().TTL)
	clk.Set(now.Add(32*time.Second + 999*time.Millisecond))
	require.Equal(t, uint32(0), cache.get(t.Context(), query, true).Answer[0].Header().TTL)
	clk.Set(now.Add(33 * time.Second))
	require.Nil(t, cache.get(t.Context(), query, true))
}

func TestForwardCacheKeepsTTLsBeyondOneHour(t *testing.T) {
	now := time.Unix(100, 0)
	clk := clock.NewTestClock(now)
	cache := newTestForwardCache(t, MinForwardCacheEntries, clk)
	query := dnswire.NewMsg("long.example.", dnswire.TypeA)
	require.NoError(t, cache.set(t.Context(), query, cachedAResponse("long.example.", 7200)))

	for _, age := range []time.Duration{time.Hour + time.Second, 7199 * time.Second} {
		clk.Set(now.Add(age))
		response := cache.get(t.Context(), query, false)
		require.NotNil(t, response, age)
		require.Equal(t, uint32(7200-age/time.Second), response.Answer[0].Header().TTL, age)
	}
	clk.Set(now.Add(7200 * time.Second))
	require.Nil(t, cache.get(t.Context(), query, false))
}

func TestNormalizedResponseCacheTTLWireBounds(t *testing.T) {
	additional := cachedAResponse("example.", 30)
	additional.Extra = cachedAResponse("extra.example.", 0x80000000).Answer
	negative := &dnswire.Msg{
		MsgHeader: dnswire.MsgHeader{Rcode: dnswire.RcodeNameError},
		Answer: []dnswire.RR{&dnswire.CNAME{
			Hdr:   dnswire.Header{Name: "example.", Class: dnswire.ClassINET, TTL: 0x80000000},
			CNAME: rdata.CNAME{Target: "missing.example."},
		}},
		Ns: []dnswire.RR{&dnswire.SOA{
			Hdr: dnswire.Header{Name: "example.", Class: dnswire.ClassINET, TTL: 20},
			SOA: rdata.SOA{Minttl: 7},
		}},
	}
	negativeSOA := &dnswire.Msg{
		MsgHeader: dnswire.MsgHeader{Rcode: dnswire.RcodeNameError},
		Ns: []dnswire.RR{&dnswire.SOA{
			Hdr: dnswire.Header{Name: "example.", Class: dnswire.ClassINET, TTL: 0x80000000},
			SOA: rdata.SOA{Minttl: 7},
		}},
	}
	withOPT := cachedAResponse("example.", 30)
	withOPT.Extra = []dnswire.RR{&dnswire.OPT{Hdr: dnswire.Header{Name: ".", TTL: 0x80000000}}}

	for _, tt := range []struct {
		name     string
		response *dnswire.Msg
		want     uint32
	}{
		{name: "largest valid TTL", response: cachedAResponse("example.", 0x7fffffff), want: 0x7fffffff},
		{name: "first invalid TTL", response: cachedAResponse("example.", 0x80000000)},
		{name: "all TTL bits set", response: cachedAResponse("example.", 0xffffffff)},
		{name: "invalid additional record", response: additional},
		{name: "invalid negative answer CNAME", response: negative},
		{name: "invalid negative answer SOA", response: negativeSOA},
		{name: "OPT carries flags, not a TTL", response: withOPT, want: 30},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zeroOutOfRangeTTLs(tt.response)
			require.Equal(t, tt.want, cacheTTL(tt.response))
			for _, section := range [][]dnswire.RR{tt.response.Answer, tt.response.Ns, tt.response.Extra} {
				for _, record := range section {
					if _, pseudo := record.(*dnswire.OPT); pseudo {
						require.Equal(t, uint32(0x80000000), record.Header().TTL)
						continue
					}
					require.LessOrEqual(t, record.Header().TTL, uint32(0x7fffffff))
				}
			}
		})
	}
}

func TestForwardCacheRejectsUnusableCapacity(t *testing.T) {
	_, err := newForwardCache(MinForwardCacheEntries-1, clock.New())
	require.ErrorContains(t, err, "below 10")
	_, err = New(nil, Config{QueriesInFlight: 1, ForwardsInFlight: 1, ForwardCacheEntries: MinForwardCacheEntries - 1, Clock: clock.New()})
	require.ErrorContains(t, err, "forward_cache_entries must be at least 10")
}

// TestForwardCacheIsBounded guarantees that the public response cache never
// holds more than forward_cache_entries responses, so callers can't grow
// resolver memory by querying distinct names.
func TestForwardCacheIsBounded(t *testing.T) {
	const capacity = 16
	cache := newTestForwardCache(t, capacity, clock.New())
	queries := make([]*dnswire.Msg, 0, 256)
	for i := range 256 {
		name := fmt.Sprintf("name-%d.example.", i)
		query := dnswire.NewMsg(name, dnswire.TypeA)
		require.NoError(t, cache.set(t.Context(), query, cachedAResponse(name, 30)))
		queries = append(queries, query)
	}
	require.Eventually(t, func() bool {
		cached := 0
		for _, query := range queries {
			if cache.get(t.Context(), query, false) != nil {
				cached++
			}
		}
		return cached > 0 && cached <= capacity
	}, 5*time.Second, 10*time.Millisecond)
}

func TestForwardCacheNegativeAndFailureBoundaries(t *testing.T) {
	negative := new(dnswire.Msg)
	negative.Rcode = dnswire.RcodeNameError
	negative.Ns = []dnswire.RR{&dnswire.SOA{
		Hdr: dnswire.Header{Name: "example.", Class: dnswire.ClassINET, TTL: 20},
		SOA: rdata.SOA{Minttl: 7},
	}}
	require.Equal(t, uint32(7), cacheTTL(negative))
	require.Equal(t, uint32(7), negative.Ns[0].Header().TTL)

	negative.Answer = []dnswire.RR{&dnswire.CNAME{
		Hdr:   dnswire.Header{Name: "alias.example.", Class: dnswire.ClassINET, TTL: 3},
		CNAME: rdata.CNAME{Target: "missing.example."},
	}}
	negative.Ns[0].Header().TTL = 20
	require.Equal(t, uint32(3), cacheTTL(negative))
	require.Equal(t, uint32(3), negative.Ns[0].Header().TTL)

	for _, rcode := range []uint16{dnswire.RcodeServerFailure, dnswire.RcodeRefused} {
		failure := cachedAResponse("example.", 30)
		failure.Rcode = rcode
		require.Zero(t, cacheTTL(failure))
	}
	truncated := cachedAResponse("example.", 30)
	truncated.Truncated = true
	require.Zero(t, cacheTTL(truncated))
	empty := new(dnswire.Msg)
	empty.Rcode = dnswire.RcodeSuccess
	require.Zero(t, cacheTTL(empty))
}

func TestForwardCacheSetReturnsPackError(t *testing.T) {
	response := cachedAResponse(strings.Repeat("a", 64)+".", 30)
	err := newTestForwardCache(t, MinForwardCacheEntries, clock.New()).set(t.Context(), dnswire.NewMsg("example.", dnswire.TypeA), response)
	require.ErrorContains(t, err, "pack forward cache response")
}

func TestForwardCacheNegativeAndZeroTTLInvalidatePositive(t *testing.T) {
	now := time.Unix(100, 0)
	clk := clock.NewTestClock(now)
	cache := newTestForwardCache(t, MinForwardCacheEntries, clk)
	query := dnswire.NewMsg("example.", dnswire.TypeA)
	require.NoError(t, cache.set(t.Context(), query, cachedAResponse("example.", 1)))

	clk.Set(now.Add(time.Second))
	negative := &dnswire.Msg{MsgHeader: dnswire.MsgHeader{Response: true, Rcode: dnswire.RcodeNameError}}
	require.NoError(t, cache.set(t.Context(), query, negative))
	require.Nil(t, cache.get(t.Context(), query, true))

	clk.Set(now)
	require.NoError(t, cache.set(t.Context(), query, cachedAResponse("example.", 1)))
	clk.Set(now.Add(time.Second))
	require.NoError(t, cache.set(t.Context(), query, cachedAResponse("example.", 0)))
	require.Nil(t, cache.get(t.Context(), query, true))
}

func TestForwardCacheRequestKeyBoundaries(t *testing.T) {
	cache := newTestForwardCache(t, MinForwardCacheEntries, clock.NewTestClock(time.Unix(100, 0)))
	query := dnswire.NewMsg("MiXeD.example.", dnswire.TypeA)
	require.NoError(t, cache.set(t.Context(), query, cachedAResponse("MiXeD.example.", 30)))
	require.NotNil(t, cache.get(t.Context(), dnswire.NewMsg("mixed.EXAMPLE.", dnswire.TypeA), false))
	require.Nil(t, cache.get(t.Context(), dnswire.NewMsg("other.example.", dnswire.TypeA), false))
	require.Nil(t, cache.get(t.Context(), dnswire.NewMsg("mixed.example.", dnswire.TypeAAAA), false))
	require.Nil(t, cache.get(t.Context(), dnswire.NewMsg("mixed.example.", dnswire.TypeA, dnswire.ClassCHAOS), false))
	query.RecursionDesired = !query.RecursionDesired
	require.Nil(t, cache.get(t.Context(), query, false))
}

func cachedAResponse(name string, ttl uint32) *dnswire.Msg {
	return &dnswire.Msg{
		MsgHeader: dnswire.MsgHeader{Response: true, Rcode: dnswire.RcodeSuccess},
		Question:  []dnswire.RR{dnswire.NewMsg(name, dnswire.TypeA).Question[0]},
		Answer: []dnswire.RR{&dnswire.A{
			Hdr: dnswire.Header{Name: name, Class: dnswire.ClassINET, TTL: ttl},
			A:   rdata.A{Addr: netip.MustParseAddr("192.0.2.1")},
		}},
	}
}
