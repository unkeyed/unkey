# Why ratelimits leak in the middle of a window — ELI5

## The picture

Imagine a 60-second window with a limit of 162,000 tokens. The ClickHouse
chart shows two green vertical lines (window boundaries) every 60 seconds.
Orange bars = denied requests, blue bars = allowed.

```diagram
deny                       deny
 ┃                          ┃
 ┃  pass pass pass pass     ┃
 ┃    pass pass pass        ┃
 ▼          (leak!)         ▼
 │──────────────────────────│
 0s         30s            60s
 ┃                          ┃
 boundary                boundary
```

At the boundaries we block almost everything. In the **middle** of each
window a noticeable amount of traffic slips through, and customers end up
getting ~300K tokens through a 162K cap.

## Why it happens

Each API replica keeps two numbers in memory for the current window:

- `cur` — how many requests this replica thinks have been counted so far
- `prev` — the same number from the previous window (used by the sliding
  window math)

The sliding window formula is:

```
effective = cur + prev * (1 - elapsed_fraction)
```

- At `elapsed = 0` (start of window): `effective = cur + prev` — `prev` is
  fully weighted, so even a slightly under-counted `cur` gets covered by
  `prev`'s mass.
- At `elapsed = 0.5` (middle of window): `effective = cur + prev / 2` —
  `prev`'s contribution is halved, so any error in `cur` is no longer
  hidden.
- At `elapsed → 1` (end of window): `effective ≈ cur` — `prev` is gone.

So the math is only as good as `cur` is.

## The bug: `cur` goes stale

Each replica's `cur` is supposed to reflect the **shared** count across
the whole fleet. It's kept in sync through Redis: every accepted request
gets buffered and then async-pushed to Redis (`syncWithOrigin`), and the
post-push value is merged back into the local `cur`.

Two things break that sync mid-window:

1. **Async replay race.** The replay is asynchronous. When many replicas
   serve the same identifier at the same instant, each replica's local
   CAS loop admits requests against its own `cur` for tens of
   milliseconds before its own replays catch up. Every replica
   independently lets through a slice of the budget.

2. **No in-band refresh.** There's a periodic safety net,
   `EnsureFreshFromOrigin`, that re-reads Redis when a warm entry is
   stale. But it's gated by `originFetchAgeMax = 60s`, and the typical
   window duration is also 60s. So a `cur` hydrated at the start of the
   window is "fresh enough" for the entire 60s it exists — the safety net
   never fires while it could still help.

Result: `cur` on each replica drifts low. At `elapsed = 0.5` that drift
is no longer masked by `prev`, and the burst leaks through.

## Cross-region makes it worse

Inter-region sync uses MySQL, not Redis:

- Each region pushes its local count every 10s
  (`globalPushInterval = 10s`).
- Each region pulls others' counts every 10s
  (`globalPullInterval = 10s`).
- End-to-end propagation: 10–20s.
- A region whose own count is below 50% of the limit
  (`globalUtilizationFloor = 0.5`) doesn't push at all.

So in a multi-region deployment, several regions can each independently
sit at 40% of the limit, never publish their state, and collectively
exceed the cap.

## Why specifically the middle of the window

| Time in window | `prev` weight | Effect on stale `cur`      |
| -------------- | ------------- | -------------------------- |
| Boundary (0s)  | 100%          | Hides stale `cur` entirely |
| Middle (30s)   | 50%           | Half-hides it — leak shows |
| End (~60s)     | 0%            | Fully exposed (also leaks) |

The boundary is well-protected because `prev` (which has had a full
window to converge across replicas/regions) dominates. As `prev` decays
linearly, the under-counted `cur` becomes the deciding term, and
mid-window is exactly where customers see the burst-through.

## Mitigations

Each option below trades some combination of Redis/MySQL load, hot-path
latency, and code complexity for a tighter leak. They compose — the
recommended path is to stack 1+2+4 first and only adopt 5/6 if telemetry
shows we still leak.

### 1. Shrink `originFetchAgeMax` proportional to window duration

Today: hard-coded `60s`. For a 60s window this guarantees a stale `cur`
never refreshes mid-window.

Proposal: per-entry refresh budget = `min(originFetchAgeMaxCap, durationMs/N)`
with `N ≈ 60` and `originFetchAgeMaxCap ≈ 1s`.

| Window  | New refresh interval | Refreshes per window |
| ------- | -------------------- | -------------------- |
| 1s–60s  | 1s                   | 1–60                 |
| 5min    | 5s                   | 60                   |
| 1h      | 1min                 | 60                   |
| 24h     | 24min                | 60                   |

- **Pros:** Cheap. Bounds Redis GETs to ~60 per entry per window
  regardless of window length. Closes most of the 60s leak.
- **Cons:** Adds N×entries×replicas GETs to Redis. For 1000 active
  entries × 10 replicas = ~10k GET/s steady-state on a 60s window. Redis
  copes easily but the budget should be measured.
- **Status:** Lowest-risk first move.

### 2. Lower `globalUtilizationFloor`

Today: `0.5`. A region whose **own local** count is under 50% of the
limit never publishes — so K regions each sitting at 40% all silently
share the cap and collectively overshoot.

Proposal: drop to `0.1` (publish from 10% onwards). Optional refinement:
make it adaptive — `floor = max(0.1, 1/numRegions)` so the floor scales
inversely with how many regions can plausibly each hold a slice.

- **Pros:** Closes the "many quiet regions" failure mode without changing
  any in-region behavior.
- **Cons:** ~5× more rows pushed to `ratelimit_global_counters`. MySQL
  write rate increases; tune `globalPushChunkSize` if needed.
- **Status:** Cheap mitigation for cross-region case specifically.

### 3. Tighten `globalPushInterval` / `globalPullInterval`

Today: `10s` each → 10–20s end-to-end propagation.

Proposal: drop to `2s` each → 2–4s propagation.

- **Pros:** Cross-region convergence happens within a window.
- **Cons:** 5× more MySQL traffic in both directions. With the floor
  already lowered (option 2), aggregate write rate could grow ~25×.
  Probably acceptable but should be measured.
- **Status:** Pair with option 2 to bound the row count growth.

### 4. Proactive strict mode at high utilization

Today: strict mode is entered only **after a denial**. Until the first
deny, replicas serve from local state with no extra Redis pressure.

Proposal: enter strict mode when the local `effective` count crosses,
e.g., `0.8 * limit`. Strict mode forces a Redis GET on every subsequent
request, so the replica converges with the shared truth before the limit
is breached — not after.

- **Pros:** Targets the exact regime where it matters (near-limit) and
  costs nothing for low-utilization keys. Predictable upper bound on
  Redis load: each high-utilization entry pulls per request.
- **Cons:** Adds ~1 Redis round-trip of latency to hot-path requests on
  near-limit keys.
- **Status:** Recommended; complementary to option 1.

### 5. Synchronous Redis INCR on the hot path

Replace local CAS + async replay with `INCR` against Redis on every
request. This is what most production rate limiters do
(`redis-cell`, sliding-window Lua scripts, etc.).

- **Pros:** Eliminates the local-stale-view class of bug entirely. The
  shared count is always authoritative.
- **Cons:** Every authenticated request now pays a Redis round-trip
  (~1–3ms intra-AZ, more cross-AZ). Redis becomes a hard dependency on
  the request path; outage = no rate-limit decisions. Need careful
  circuit-breaker fallback design.
- **Status:** Architectural change; consider after measuring whether
  1+2+4 reduce the leak enough.

### 6. Coordinator / sharded ownership

Hash each (workspace, namespace, identifier, duration) tuple to one
"owner" replica per region. All other replicas forward decisions to the
owner. Owner holds the canonical local count.

- **Pros:** Eliminates within-region drift; sync becomes a single
  replica's responsibility per key.
- **Cons:** Adds a hop, requires consistent hashing and rebalancing
  logic, complicates failover. Significant engineering work.
- **Status:** Long-term option if 1–5 prove insufficient.

## Recommended rollout

1. Ship option 1 (`originFetchAgeMax` scaled to `duration/60`,
   capped at 1s) — small diff, big win.
2. Ship option 4 (proactive strict mode at 80% utilization) — also
   small, complementary.
3. Ship options 2+3 together (`globalUtilizationFloor = 0.1`,
   intervals down to 2s) once we've measured MySQL headroom.
4. Re-run `TestRatelimit_MultiReplicaMidWindowBurstExceedsLimit`.
   If still leaking by more than ~10%, revisit option 5.

## Test

`TestRatelimit_MultiReplicaMidWindowBurstExceedsLimit` in
`ratelimit_test.go` reproduces this with the production tuple
(limit=162000, cost=1500, 60s window) and currently shows
**~481,500 tokens admitted against a 162,000 cap** — matching the chart.
