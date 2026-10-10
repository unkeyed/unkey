package resolver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

const (
	forwardCacheStaleWindow = 30 * time.Second
	// maxDNSTTL is the largest TTL allowed by RFC 2181 section 8.
	maxDNSTTL          = math.MaxInt32 * time.Second
	forwardCacheMaxAge = maxDNSTTL + forwardCacheStaleWindow

	// MinForwardCacheEntries is the smallest usable capacity: otter rejects
	// every entry costing more than a tenth of the cache's capacity.
	MinForwardCacheEntries = 10
)

type forwardKey struct {
	name  string
	qtype uint16
	class uint16
	rd    bool
}

type forwardEntry struct {
	message   []byte
	storedAt  time.Time
	expiresAt time.Time
	staleAt   time.Time
	positive  bool
}

type forwardCache struct {
	entries cache.Cache[forwardKey, forwardEntry]
	clock   clock.Clock
}

func newForwardCache(capacity int, clk clock.Clock) (*forwardCache, error) {
	if capacity < MinForwardCacheEntries {
		return nil, fmt.Errorf("forward cache capacity %d is below %d", capacity, MinForwardCacheEntries)
	}
	entries, err := cache.New(cache.Config[forwardKey, forwardEntry]{
		Fresh:    forwardCacheMaxAge,
		Stale:    forwardCacheMaxAge,
		MaxSize:  capacity,
		Resource: "undns_forward_responses",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("create forward cache: %w", err)
	}
	return &forwardCache{entries: entries, clock: clk}, nil
}

func (c *forwardCache) get(ctx context.Context, request *dnswire.Msg, allowStale bool) *dnswire.Msg {
	key := requestKey(request)
	entry, hit := c.entries.Get(ctx, key)
	if hit != cache.Hit {
		return nil
	}
	now := c.clock.Now()
	expired := !now.Before(entry.expiresAt)
	if expired && (!entry.positive || !now.Before(entry.staleAt)) {
		c.entries.Remove(ctx, key)
		return nil
	}
	if expired && !allowStale {
		return nil
	}

	response := new(dnswire.Msg)
	response.Data = append(response.Data, entry.message...)
	if err := response.Unpack(); err != nil {
		c.entries.Remove(ctx, key)
		return nil
	}
	if expired {
		decayTTLs(response, ^uint32(0))
	} else {
		decayTTLs(response, uint32(now.Sub(entry.storedAt)/time.Second))
	}
	return response
}

func (c *forwardCache) set(ctx context.Context, request, response *dnswire.Msg) error {
	key := requestKey(request)
	ttl := cacheTTL(response)
	if ttl == 0 {
		if response.Rcode == dnswire.RcodeSuccess || response.Rcode == dnswire.RcodeNameError {
			c.entries.Remove(ctx, key)
		}
		return nil
	}

	positive := response.Rcode == dnswire.RcodeSuccess && len(response.Answer) > 0
	id := response.ID
	response.ID = 0
	defer func() { response.ID = id }()
	if err := response.Pack(); err != nil {
		return fmt.Errorf("pack forward cache response: %w", err)
	}

	now := c.clock.Now()
	expiresAt := now.Add(time.Duration(ttl) * time.Second)
	staleAt := expiresAt
	if positive {
		staleAt = staleAt.Add(forwardCacheStaleWindow)
	}
	c.entries.Set(ctx, key, forwardEntry{
		message:   append([]byte(nil), response.Data...),
		storedAt:  now,
		expiresAt: expiresAt,
		staleAt:   staleAt,
		positive:  positive,
	})
	return nil
}

func (c *forwardCache) close() {
	c.entries.Close()
}

func requestKey(request *dnswire.Msg) forwardKey {
	question := request.Question[0]
	return forwardKey{
		name:  strings.ToLower(question.Header().Name),
		qtype: dnswire.RRToType(question),
		class: question.Header().Class,
		rd:    request.RecursionDesired,
	}
}

func cacheTTL(response *dnswire.Msg) uint32 {
	if response.Truncated {
		return 0
	}
	if response.Rcode == dnswire.RcodeNameError || response.Rcode == dnswire.RcodeSuccess && len(response.Answer) == 0 {
		ttl := ^uint32(0)
		for _, record := range response.Answer {
			ttl = min(ttl, record.Header().TTL)
		}
		for _, record := range response.Ns {
			soa, ok := record.(*dnswire.SOA)
			if ok {
				ttl = min(ttl, soa.Hdr.TTL, soa.Minttl)
				soa.Hdr.TTL = ttl
				return ttl
			}
		}
		return 0
	}
	if response.Rcode != dnswire.RcodeSuccess || len(response.Answer) == 0 {
		return 0
	}

	ttl := ^uint32(0)
	for _, section := range [][]dnswire.RR{response.Answer, response.Ns, response.Extra} {
		for _, record := range section {
			if _, pseudo := record.(*dnswire.OPT); !pseudo {
				ttl = min(ttl, record.Header().TTL)
			}
		}
	}
	return ttl
}

func decayTTLs(response *dnswire.Msg, elapsed uint32) {
	for _, section := range [][]dnswire.RR{response.Answer, response.Ns, response.Extra} {
		for _, record := range section {
			if _, pseudo := record.(*dnswire.OPT); pseudo {
				continue
			}
			if record.Header().TTL > elapsed {
				record.Header().TTL -= elapsed
			} else {
				record.Header().TTL = 0
			}
		}
	}
}
