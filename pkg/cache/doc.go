// Package cache is an in-memory cache with a fixed maximum size and
// stale-while-revalidate reads. It uses S3-FIFO to decide which entry to drop
// when it is full. It reads time only from an injected
// [github.com/unkeyed/unkey/pkg/clock.Clock], so a deterministic simulation
// can control it from a seed.
//
// If you have written a cache as a Go map behind a mutex, you already know
// most of this package. Three parts are probably new: how the cache decides
// what to drop when it is full (S3-FIFO), how it returns old data while it
// fetches new data (stale-while-revalidate), and how a test can control both.
// The sections below explain each part.
//
// # Fresh, stale, and expired
//
// When the cache writes an entry, it gives the entry two deadlines. They come
// from [Config.Fresh] and [Config.Stale]:
//
//	written        written+Fresh            written+Stale
//	   |---- fresh ----|-------- stale --------|---- expired ...
//
// The cache returns a fresh entry as it is. It also returns a stale entry, so
// the caller does not wait for the origin. In addition, [Cache.SWR] and its
// variants schedule a refresh of a stale entry in the background. The cache
// never returns an expired entry. Instead, the read deletes the entry and
// reports [Miss]. [Cache.SWR] then loads the key from the origin, and the
// caller waits for the result.
//
// The cache checks expiry when an entry is read, against [Config.Clock]. In
// addition, a sweep deletes every expired entry once a minute, so an entry
// that nobody reads leaves memory at most a minute after its stale time. The
// [Scheduler] decides when the sweep runs. No code in the package uses
// wall-clock time to decide whether an entry is alive.
//
// # Why a cache needs an eviction policy
//
// A plain map grows until the process runs out of memory. A cache has a fixed
// limit of [Config.MaxSize] entries. When a new key arrives and the cache is
// full, the cache must drop an old entry. The eviction policy decides which
// entry to drop. A good policy drops keys that will not be read again soon and
// keeps keys that will be.
//
// The most common policy is LRU (least recently used). An LRU cache keeps a
// linked list ordered by last access. Every read moves its entry to the front
// of the list, and eviction drops the entry at the back. This works, but every
// read changes the shared list, so every read needs the lock. When many
// goroutines read a busy cache, this lock slows them all down. LRU has a second
// problem. A burst of keys that are read only once can push out every useful
// key, because each new key looks recently used.
//
// # S3-FIFO
//
// S3-FIFO (https://s3fifo.com) solves both problems. It uses three
// first-in-first-out (FIFO) queues and a small counter on each entry. It is
// based on one observation: in real traffic, most keys are read once and never
// again. These keys are called one-hit wonders. A cache works best when it
// finds them quickly and drops them before they push out useful keys.
//
// Each entry has a read counter from 0 to 3. A read increments the counter and
// does nothing else. No entry moves, no queue changes, and no lock is taken.
// This is the main difference from LRU. The policy does all of its work when a
// write needs room, and a write already holds a lock.
//
// The three queues are:
//
//	small  about 10% of MaxSize. Every new key starts here.
//	main   the rest of MaxSize. Keys that were read again while in small.
//	ghost  keys only, no values. Keys that small recently dropped.
//
// When the cache is full, it drops one entry. While small holds at least its
// 10% share, or when main is empty, the cache takes the oldest entry of small:
//
//   - If the entry was read more than once while it was in small, it is not a
//     one-hit wonder. The cache moves it to main, resets its counter to 0, and
//     continues with the next entry.
//   - Otherwise, the cache drops the entry and remembers its key in ghost.
//
// Otherwise, the cache takes the oldest entry of main:
//
//   - If its counter is above 0, the entry was read since it was last checked.
//     The cache puts it back at the front of main, decrements its counter, and
//     continues with the next entry.
//   - Otherwise, the cache drops the entry.
//
// The ghost queue corrects mistakes. If a key that small dropped is written
// again soon, the key was not a one-hit wonder. The cache then puts the key
// directly into main and skips small. Ghost stores only keys. When it holds
// more keys than main has room for, it forgets its oldest key.
//
// Here is an example with a MaxSize of 10, so the share of small is 1 entry.
// Keys k0 to k9 fill the cache, and all of them are in small. k0 is read
// twice. Then the key "new" arrives:
//
//	small (oldest first): k0 k1 k2 ... k9      main: (empty)
//	k0 was read twice     -> move k0 to main
//	k1 was never read     -> drop k1, remember k1 in ghost
//	small: k2 ... k9 new                       main: k0
//
// A plain FIFO cache would have dropped k0, the only key in use. If k1 is
// written again while ghost still remembers it, k1 goes directly to main.
// After that, a burst of new keys only cycles through small and does not
// touch k0 or k1.
//
// Two limits keep each eviction cheap. First, the counter stops at 3, so a key
// that is no longer read survives at most three passes through main. Second,
// one eviction puts at most 20 entries of main back in a row. After that, it
// drops the next entry, even if that entry was read. So when every key in main
// is in use, one write never walks the whole queue.
//
// # Concurrency
//
// Entries are stored in a [sync.Map]. A read is therefore a map lookup that
// takes no lock. If the counter is below 3, the read also does one atomic
// compare-and-swap. A busy key whose counter is already 3 causes no shared
// writes at all. Writes, deletes, and evictions hold one mutex that protects
// the three queues. A cache writes only on misses and refreshes, not on hits,
// so readers rarely wait for this mutex.
//
// To update an existing key, the cache builds a new entry and puts it in the
// same queue position as the old one. A concurrent reader therefore sees
// either the complete old value or the complete new value.
//
// # Background revalidation
//
// A stale read gives a refresh to the [Scheduler] in [Config.Scheduler] and
// returns at once. The default scheduler runs 10 goroutines that take
// refreshes from a queue of up to 1000. [Cache.Close] stops these goroutines.
// When the queue is full, the cache drops the refresh. The entry is still
// stale, so the next read schedules the refresh again. The cache runs at most
// one refresh per key at a time. Concurrent [Cache.SWR] misses for the same
// key share one call to the origin.
//
// # Deterministic simulation
//
// Three things decide what a cache returns over time: the clock, the order of
// writes, and the time at which background refreshes run. This package gets
// all three from the caller:
//
//   - Expiry and every timing metric use [Config.Clock], so a
//     [github.com/unkeyed/unkey/pkg/clock.TestClock] controls time.
//   - Eviction runs inside the write that needs room. It uses no randomness
//     and no goroutines, so the same sequence of calls always drops the same
//     keys.
//   - A [Scheduler] that only stores refreshes and the sweep lets a
//     simulation run each of them at the time its seed chooses.
//
// simulation_test.go contains a seeded simulation built on pkg/sim. It checks
// every result against a model and runs each seed twice to confirm that both
// runs are the same.
//
// The package does not control one input. [Cache.SetMany] takes a map and
// writes its entries in Go's random map order. On a full cache, this order
// changes which keys are dropped. [Cache.SWRMany] does not have this problem,
// because it writes the keys it fetched in the order that the caller gave.
//
// # Metrics
//
// unkey_cache_deleted_total counts removed entries by reason:
//
//   - "Size": the entry was evicted.
//   - "Expired": a read or the sweep found the entry past its stale deadline.
//   - "Explicit": [Cache.Remove] removed the entry.
//   - "Replaced": a write replaced the value.
//
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
// hit is [Hit] when the cache returned a value. It is [Null] when the origin
// reported that the key does not exist and op told the cache to store that
// result. It is [Miss] when the cache could not return anything.
package cache
