package githubsecrets

import (
	"context"
	"crypto/ecdsa"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/singleflight"
)

type keyFetchResult struct {
	keys map[string]*ecdsa.PublicKey
	err  error
}

// New keeps signing keys fresh for one hour and usable for up to 24 hours.
// Unknown key IDs trigger a refresh, with all fetches limited to once per minute.
// The supplied clock must be non-nil. Call Close at shutdown.
func New(keysURL string, clk clock.Clock) (*Verifier, error) {
	keys, err := cache.New(cache.Config[string, map[string]*ecdsa.PublicKey]{
		Fresh:     time.Hour,
		Stale:     24 * time.Hour,
		MaxSize:   10,
		Resource:  "github_secret_scanning_keys",
		Clock:     clk,
		Scheduler: nil,
	})
	if err != nil {
		return nil, err
	}

	fetchResults, err := cache.New(cache.Config[string, keyFetchResult]{
		Fresh:     time.Minute,
		Stale:     time.Minute,
		MaxSize:   10,
		Resource:  "github_secret_scanning_key_fetches",
		Clock:     clk,
		Scheduler: nil,
	})
	if err != nil {
		keys.Close()
		return nil, err
	}

	return &Verifier{
		keysURL:      keysURL,
		client:       &http.Client{Timeout: 10 * time.Second},
		keys:         keys,
		fetchResults: fetchResults,
		fetchGroup:   singleflight.Group[keyFetchResult]{},
	}, nil
}

// Close stops background cache workers. It is idempotent.
func (v *Verifier) Close() {
	v.keys.Close()
	v.fetchResults.Close()
}

func (v *Verifier) publicKeys(ctx context.Context, keyID string) (map[string]*ecdsa.PublicKey, error) {
	keys, _, err := v.keys.SWR(ctx, v.keysURL, v.refreshKeys, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	if err != nil {
		return nil, err
	}
	if _, found := keys[keyID]; found {
		return keys, nil
	}

	keys, err = v.refreshKeys(ctx)
	if err != nil {
		return nil, err
	}
	v.keys.Set(ctx, v.keysURL, keys)
	return keys, nil
}

func (v *Verifier) refreshKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	result, err := v.fetchGroup.DoContext(ctx, v.keysURL, func() (keyFetchResult, error) {
		if previous, hit := v.fetchResults.Get(ctx, v.keysURL); hit == cache.Hit {
			return previous, nil
		}

		keys, err := v.fetchKeys(ctx)
		result := keyFetchResult{keys: keys, err: err}
		v.fetchResults.Set(ctx, v.keysURL, result)
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return result.keys, result.err
}
