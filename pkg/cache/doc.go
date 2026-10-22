// Package cache is an in-memory, size-bounded cache with stale-while-revalidate
// reads. It evicts with S3-FIFO and runs entirely on an injected
// [github.com/unkeyed/unkey/pkg/clock.Clock], so a deterministic simulation
// can drive it from a seed.
//
// If you have written a cache as a Go map behind a mutex, you already know
// most of this package. The parts that are new are how it decides what to
// forget when it is full (S3-FIFO), how it serves old data while it fetches
// new data (stale-while-revalidate), and how it keeps both under the control
// of a test. This comment explains each in turn.
//
// # Fresh, stale, and expired
//
// Every entry carries two deadlines, set from [Config.Fresh] and
// [Config.Stale] when the entry is written:
//
//	written        written+Fresh            written+Stale
//	   |---- fresh ----|-------- stale --------|---- expired ...
//
// A fresh entry is returned as is. A stale entry is still returned, so the
// caller does not wait for the origin, but [Cache.SWR] and its variants also
// schedule a background refresh. An expired entry is never returned: the read
// deletes it and reports [Miss]. [Cache.SWR] then loads the key from the
// origin while the caller waits.
//
// Expiry is checked on read against [Config.Clock], not by a background
// timer. Nothing in the package looks at wall-clock time to decide whether an
// entry is alive. An expired entry that nobody reads stays in memory until
// eviction removes it, which S3-FIFO does early because nobody reads it.
//
// # Why a cache needs an eviction policy
//
// A map grows until the process runs out of memory. A cache has a fixed
// budget, [Config.MaxSize] entries, and when a new key arrives at a full cache
// it must forget an old one. The eviction policy picks which. A good policy
// forgets keys that will not be read again soon and keeps keys that will.
//
// The usual choice is LRU: keep a linked list ordered by last access, move an
// entry to the front on every read, and evict from the back. LRU works, but
// every read writes to the shared list, so every read needs the lock. On a hot
// cache read by many goroutines, that lock is the bottleneck. LRU also lets a
// single scan of keys that are read once push out the entire working set,
// because each new key looks "recently used".
//
// # S3-FIFO
//
// S3-FIFO (Yang et al., "FIFO queues are all you need for cache eviction",
// SOSP 2023) fixes both problems with three first-in-first-out queues and a
// small counter per entry. It rests on one observation: in real workloads
// most keys are read once and never again. These are "one-hit wonders". A
// cache does best when it notices them quickly and drops them before they
// displace anything useful.
//
// Each entry has a frequency counter from 0 to 3. A read increments it and
// does nothing else: no node moves, no list changes, and no lock is taken.
// This is the main difference from LRU. All of the policy's work happens when
// a write needs room, and a write already takes a lock.
//
// The queues are:
//
//	small  about 10% of MaxSize. Every new key starts here.
//	main   the rest of MaxSize. Keys that proved they are read again.
//	ghost  keys only, no values. Keys recently evicted from small.
//
// When the cache is full and must evict one entry, it looks at the oldest
// entry of small while small holds at least its 10% share:
//
//   - If the entry was read more than once while it was in small, it is not a
//     one-hit wonder. It moves to main with its counter reset to 0, and the
//     search continues.
//   - Otherwise the entry is evicted, and its key is remembered in ghost.
//
// When small holds less than its share, the cache looks at the oldest entry of
// main instead:
//
//   - If its counter is above 0, it was read since it last came around. It goes
//     back to the front of main with its counter decremented, and the search
//     continues. This is the "second chance" of the CLOCK algorithm.
//   - Otherwise it is evicted.
//
// The ghost queue catches a mistake. If a key that small evicted is written
// again soon, it was not a one-hit wonder after all, so it skips small and
// goes straight into main. Ghost entries hold only the key, and ghost forgets
// its oldest key once it holds more keys than main has room for.
//
// A short example with MaxSize 10, so small's share is 1 entry. Keys k0 to k9
// fill the cache, all in small. k0 is read twice. Then key "new" arrives:
//
//	small (oldest first): k0 k1 k2 ... k9      main: (empty)
//	k0 was read twice     -> move k0 to main
//	k1 was never read     -> evict k1, remember k1 in ghost
//	small: k2 ... k9 new                       main: k0
//
// A plain FIFO would have evicted k0, the one key that was in use. If k1 is
// written again while ghost still remembers it, k1 goes straight to main, and a
// later scan of new keys only churns small without touching k0 or k1.
//
// Two bounds keep the cost of one eviction small. The counter stops at 3, so
// a hot key survives at most three trips through main without new reads. And
// main gives at most 20 entries in a row a second chance before it evicts the
// next one regardless, so a cache full of hot keys never walks the whole
// queue to make room for one write.
//
// # Concurrency
//
// Entries live in a [sync.Map], so a read is a lock-free map lookup plus, if
// the counter is below 3, one atomic compare-and-swap. A hot key at counter 3
// causes no shared writes at all. Writes, deletes, and evictions take one
// mutex that guards the three queues. The mutex is held only on writes, which
// in a cache happen on misses and refreshes, not on hits.
//
// An update to an existing key builds a new entry and swaps it into the same
// queue position, so a concurrent reader sees either the old value or the new
// one, never a torn write.
//
// # Background revalidation
//
// A stale read hands a refresh to the [Scheduler] in [Config.Scheduler] and
// returns at once. The default scheduler is a pool of 10 goroutines with a
// queue of 1000 refreshes, which [Cache.Close] stops. When the queue is full the
// refresh is dropped; the entry is still stale, so the next read schedules it
// again. Refreshes for a key are deduplicated while one is queued or running,
// and concurrent [Cache.SWR] misses for the same key share one origin call.
//
// # Deterministic simulation
//
// Three things decide what a cache returns over time: the clock, the order of
// writes, and when background refreshes run. This package takes all three from
// the caller. Expiry and every timing metric use [Config.Clock], so a
// [github.com/unkeyed/unkey/pkg/clock.TestClock] controls time. Eviction runs
// synchronously inside the write that needs room and uses no randomness and
// no goroutines, so the same sequence of calls always evicts the same keys.
// And a [Scheduler] that only queues refreshes lets a simulation run each
// refresh when its seed says so. See
// simulation_test.go for a seeded simulation built on pkg/sim that checks
// every result against a model and replays each seed twice.
//
// One input stays outside the package's control: [Cache.SetMany] takes a map
// and writes it in Go's randomized map order. On a full cache that order
// changes which keys are evicted. [Cache.SWRMany] avoids this by writing the
// keys it fetched in the order the caller passed them.
//
// # Metrics
//
// unkey_cache_deleted_total counts removed entries by reason: "Size" for
// evictions, "Expired" for entries a read found past their stale time,
// "Explicit" for [Cache.Remove], and "Replaced" for overwritten values.
// [Cache.Clear] is not counted. The revalidation metrics count how refreshes
// were scheduled, how long they waited, and how long they ran.
//
// # Usage
//
//	c, err := cache.New(cache.Config[string, User]{
//		Fresh:    time.Minute,
//		Stale:    time.Hour,
//		MaxSize:  10_000,
//		Resource: "users",
//		Clock:    clock.New(),
//	})
//	if err != nil {
//		return err
//	}
//	defer c.Close()
//
//	user, hit, err := c.SWR(ctx, userID,
//		func(ctx context.Context) (User, error) {
//			return loadUser(ctx, userID)
//		},
//		func(err error) cache.Op {
//			switch {
//			case err == nil:
//				return cache.WriteValue
//			case db.IsNotFound(err):
//				return cache.WriteNull
//			default:
//				return cache.Noop
//			}
//		},
//	)
//
// hit is [Hit] for a value, [Null] when the origin reported that the key does
// not exist and op cached that fact, and [Miss] when nothing could be served.
package cache
