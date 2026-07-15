/**
 * Benchmark the audit_log query patterns against varying data volumes, on
 * BOTH the legacy MySQL schema (`audit_log` + `audit_log_target`) and the
 * new ClickHouse schema (`default.audit_logs_raw_v1` with Nested target
 * arrays).
 *
 * Workflow:
 *   1. `pnpm --filter @unkey/local exec tsx src/audit-log-benchmark.ts seed --count 1000000`
 *   2. `pnpm --filter @unkey/local exec tsx src/audit-log-benchmark.ts bench`
 *   3. `pnpm --filter @unkey/local exec tsx src/audit-log-benchmark.ts cleanup`
 *
 * Seeding writes the same logical events into both backends so per-pattern
 * latencies are comparable apples-to-apples.
 *
 * Bench patterns mirror the dashboard's filter surface (workspace + bucket
 * + time, optional event/actor/target). Each pattern is run against both
 * backends and reported side-by-side.
 *
 * Tuned for a local MySQL/ClickHouse — MySQL bulk INSERTs in parallel, CH
 * inserts as a single JSONEachRow batch per chunk.
 */
import { createClient } from "@clickhouse/client";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

type Argv = {
  command: "seed" | "bench" | "cleanup" | "help";
  workspaceId: string;
  count: number;
  bucket: string;
  lookbackHours: number;
  runs: number;
  noiseWorkspaces: number;
  noiseRatio: number;
  backend: "both" | "mysql" | "clickhouse";
};

const DATABASE_URL = process.env.DATABASE_URL ?? "mysql://unkey:password@localhost:3306/unkey";
const CLICKHOUSE_URL = process.env.CLICKHOUSE_URL ?? "http://default:password@localhost:8123";
const DEFAULT_WORKSPACE_ID = "ws_audit_bench";
const DEFAULT_COUNT = 1_000_000;
const DEFAULT_BUCKET = "unkey_mutations";
const DEFAULT_LOOKBACK_HOURS = 24;
const DEFAULT_RUNS = 5;
// Prod has many workspaces; without noise, the target workspace is 100% of
// the table and `time_idx` alone works well enough to hide the composite's
// value. Defaults spread ~30% of rows across 99 extra workspaces so the
// benchmark mirrors the "needle in a multi-tenant haystack" prod shape.
const DEFAULT_NOISE_WORKSPACES = 99;
const DEFAULT_NOISE_RATIO = 0.3;
const NOISE_WORKSPACE_PREFIX = "ws_audit_bench_noise_";

// Insert tuning — 10k events per batch keeps the audit_log_target INSERT
// (~14k rows after fan-out) under MySQL's max_allowed_packet on default
// configs. 16 workers is the sweet spot once the per-session
// `transaction_isolation = READ-COMMITTED` (set in the worker init) drops
// gap locks: contention disappears and we get linear speedup. Push higher
// only if your MySQL has the connection headroom and innodb_redo_log_capacity
// has been bumped.
const BATCH_SIZE = 10_000;
const CONCURRENCY = 16;
const POOL_SIZE = CONCURRENCY + 4;

// CH inserts are HTTP and benefit from larger batches; mirror MySQL's chunk
// so the same generated rows feed both backends.
const CH_TABLE = "default.audit_logs_raw_v1";

// Column list used by the raw bulk INSERT path. Order must match the 2D row
// arrays we pass to mysql2.
const AUDIT_LOG_COLUMNS = [
  "id",
  "workspace_id",
  "bucket",
  "bucket_id",
  "event",
  "time",
  "display",
  "remote_ip",
  "user_agent",
  "actor_type",
  "actor_id",
  "actor_name",
  "actor_meta",
  "created_at",
] as const;
const AUDIT_LOG_INSERT_SQL = `INSERT INTO audit_log (${AUDIT_LOG_COLUMNS.join(", ")}) VALUES ?`;

const AUDIT_LOG_TARGET_COLUMNS = [
  "audit_log_id",
  "workspace_id",
  "bucket",
  "bucket_id",
  "type",
  "id",
  "display_name",
  "name",
  "meta",
  "created_at",
] as const;
const AUDIT_LOG_TARGET_INSERT_SQL = `INSERT INTO audit_log_target (${AUDIT_LOG_TARGET_COLUMNS.join(
  ", ",
)}) VALUES ?`;

// Prod-like distribution: most rows land in unkey_mutations, smaller slices
// in auxiliary buckets. Keeps the query planner honest when choosing indexes.
const BUCKETS: Array<{ name: string; weight: number }> = [
  { name: "unkey_mutations", weight: 0.6 },
  { name: "api.verifications", weight: 0.2 },
  { name: "ratelimits", weight: 0.1 },
  { name: "keys", weight: 0.07 },
  { name: "permissions", weight: 0.03 },
];

const EVENTS = [
  "workspace.update",
  "api.create",
  "api.update",
  "api.delete",
  "key.create",
  "key.update",
  "key.delete",
  "key.verify",
  "ratelimit.create",
  "role.create",
  "permission.grant",
  "permission.revoke",
];

const ACTOR_TYPES = ["user", "key", "system"];
const TARGET_TYPES = ["api", "key", "ratelimit", "role", "permission", "workspace"] as const;
type TargetType = (typeof TARGET_TYPES)[number];

const NINETY_DAYS_MS = 90 * 24 * 60 * 60 * 1000;

function parseArgs(): Argv {
  const raw = process.argv.slice(2);
  const command = (raw[0] ?? "help") as Argv["command"];
  const argv: Argv = {
    command,
    workspaceId: DEFAULT_WORKSPACE_ID,
    count: DEFAULT_COUNT,
    bucket: DEFAULT_BUCKET,
    lookbackHours: DEFAULT_LOOKBACK_HOURS,
    runs: DEFAULT_RUNS,
    noiseWorkspaces: DEFAULT_NOISE_WORKSPACES,
    noiseRatio: DEFAULT_NOISE_RATIO,
    backend: "both",
  };

  for (let i = 1; i < raw.length; i++) {
    const key = raw[i];
    const value = raw[i + 1];
    switch (key) {
      case "--workspace":
        argv.workspaceId = value;
        i++;
        break;
      case "--count":
        argv.count = Number.parseInt(value, 10);
        i++;
        break;
      case "--bucket":
        argv.bucket = value;
        i++;
        break;
      case "--lookback-hours":
        argv.lookbackHours = Number.parseInt(value, 10);
        i++;
        break;
      case "--runs":
        argv.runs = Number.parseInt(value, 10);
        i++;
        break;
      case "--noise-workspaces":
        argv.noiseWorkspaces = Math.max(0, Number.parseInt(value, 10));
        i++;
        break;
      case "--noise-ratio":
        argv.noiseRatio = Math.min(1, Math.max(0, Number.parseFloat(value)));
        i++;
        break;
      case "--backend":
        argv.backend = value as Argv["backend"];
        i++;
        break;
    }
  }

  return argv;
}

function pickBucket(): string {
  const r = Math.random();
  let acc = 0;
  for (const b of BUCKETS) {
    acc += b.weight;
    if (r < acc) {
      return b.name;
    }
  }
  return BUCKETS[0].name;
}

function pickFrom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function buildNoiseWorkspaceIds(count: number): string[] {
  const out = new Array<string>(count);
  for (let i = 0; i < count; i++) {
    out[i] = `${NOISE_WORKSPACE_PREFIX}${i}`;
  }
  return out;
}

function pickWorkspace(target: string, noise: string[], noiseRatio: number): string {
  if (noise.length === 0 || Math.random() >= noiseRatio) {
    return target;
  }
  return noise[Math.floor(Math.random() * noise.length)];
}

// generated event shape, shared across both backends so seeded data agrees.
type GeneratedEvent = {
  id: string;
  workspaceId: string;
  bucket: string;
  event: string;
  time: number;
  actorType: string;
  actorId: string;
  display: string;
  targets: Array<{ type: string; id: string; name: string }>;
};

function generateEvent(workspaceId: string, now: number): GeneratedEvent {
  const bucket = pickBucket();
  const time = now - Math.floor(Math.random() * NINETY_DAYS_MS);
  const event = pickFrom(EVENTS);
  const actorType = pickFrom(ACTOR_TYPES);
  const actorId =
    actorType === "user"
      ? `user_${Math.floor(Math.random() * 1_000)}`
      : actorType === "key"
        ? `key_${Math.floor(Math.random() * 10_000)}`
        : "system";

  // Most events touch 1 target, a meaningful slice touches 2-3, almost
  // none are zero-target. Mirrors what the dashboard sees in practice.
  // newId() guarantees uniqueness within and across events, so the
  // audit_log_target UNIQUE(audit_log_id, id) constraint is satisfied for
  // free.
  const targetCount = Math.random() < 0.7 ? 1 : Math.random() < 0.9 ? 2 : 3;
  const targets = new Array<{ type: string; id: string; name: string }>(targetCount);
  for (let i = 0; i < targetCount; i++) {
    const type = pickFrom([...TARGET_TYPES]) as TargetType;
    targets[i] = { type, id: newId(type), name: `${type} #${i + 1}` };
  }

  return {
    id: newId("auditLog"),
    workspaceId,
    bucket,
    event,
    time,
    actorType,
    actorId,
    display: event,
    targets,
  };
}

// MySQL row layout follows AUDIT_LOG_COLUMNS exactly; bucket_id mirrors
// bucket since the column is deprecated but still NOT NULL.
function eventToMySQLLogRow(e: GeneratedEvent): unknown[] {
  return [
    e.id,
    e.workspaceId,
    e.bucket,
    e.bucket,
    e.event,
    e.time,
    e.display,
    null,
    null,
    e.actorType,
    e.actorId,
    null,
    null,
    e.time,
  ];
}

function eventToMySQLTargetRows(e: GeneratedEvent): unknown[][] {
  return e.targets.map((t) => [
    e.id,
    e.workspaceId,
    e.bucket,
    e.bucket,
    t.type,
    t.id,
    t.name,
    t.name,
    null,
    e.time,
  ]);
}

// CH row uses Nested subcolumn syntax for targets. Time fields are unix
// millis (Int64). Meta columns use the new JSON type — we send actual
// objects, not stringified JSON, and CH stores them natively.
function eventToCHRow(e: GeneratedEvent, insertedAtMs: number, expiresAtMs: number) {
  return {
    event_id: e.id,
    time: e.time,
    inserted_at: insertedAtMs,
    workspace_id: e.workspaceId,
    bucket: e.bucket,
    source: "platform",
    event: e.event,
    description: e.display,
    actor_type: e.actorType,
    actor_id: e.actorId,
    actor_name: "",
    actor_meta: {},
    remote_ip: "",
    user_agent: "",
    meta: {},
    "targets.type": e.targets.map((t) => t.type),
    "targets.id": e.targets.map((t) => t.id),
    "targets.name": e.targets.map((t) => t.name),
    "targets.meta": e.targets.map(() => ({})),
    expires_at: expiresAtMs,
  };
}

type Pool = mysql.Pool;

function createPool(connectionLimit = POOL_SIZE): Pool {
  return mysql.createPool({
    uri: DATABASE_URL,
    connectionLimit,
    namedPlaceholders: false,
  });
}

function createCH() {
  return createClient({
    url: CLICKHOUSE_URL,
    request_timeout: 60_000,
    // The Node client gets HTTP keep-alive + larger socket buffers vs the
    // web/fetch variant, plus optional gzip. For a tight bench loop with
    // many concurrent inserts this is meaningfully faster.
    keep_alive: { enabled: true },
    compression: { request: true, response: true },
    // Sync insert is faster than async for already-batched 25k row chunks:
    // async_insert is designed to consolidate many small inserts; we do
    // the consolidation client-side, so the async buffer wait (~200ms by
    // default) is pure overhead per call.
    clickhouse_settings: {
      output_format_json_quote_64bit_integers: 0,
      async_insert: 0,
    },
  });
}

async function seed(argv: Argv) {
  const seedMySQL = argv.backend !== "clickhouse";
  const seedCH = argv.backend !== "mysql";

  const pool = seedMySQL ? createPool() : null;
  const ch = seedCH ? createCH() : null;

  const noiseWorkspaceIds = buildNoiseWorkspaceIds(argv.noiseWorkspaces);
  const effectiveNoiseRatio = noiseWorkspaceIds.length === 0 ? 0 : argv.noiseRatio;
  const targetRowsEstimate = Math.round(argv.count * (1 - effectiveNoiseRatio));
  const noiseRowsEstimate = argv.count - targetRowsEstimate;

  console.info(
    `seeding ${argv.count.toLocaleString()} events — ~${targetRowsEstimate.toLocaleString()} into ${argv.workspaceId}, ~${noiseRowsEstimate.toLocaleString()} across ${noiseWorkspaceIds.length} noise workspaces — backends: ${argv.backend}`,
  );

  const startedAt = Date.now();
  const now = Date.now();
  const remaining = { value: argv.count };
  let inserted = 0;
  let lastLog = 0;
  // Per-batch timing accumulators, shared across workers. Updated inside
  // the parallel branches so we capture wall time per side, not the
  // serialized sum.
  const timing = { mysqlMs: 0, mysqlBatches: 0, chMs: 0, chBatches: 0 };

  const workers = Array.from({ length: CONCURRENCY }, async () => {
    const conn = pool ? await pool.getConnection() : null;
    try {
      if (conn) {
        // unique_checks=0 skips InnoDB's secondary-index dup probe at insert
        // time. foreign_key_checks=0 isn't strictly needed (audit_log has no
        // FKs) but is cheap insurance.
        await conn.query("SET SESSION unique_checks = 0");
        await conn.query("SET SESSION foreign_key_checks = 0");
        // Default REPEATABLE READ takes gap/next-key locks on UNIQUE index
        // ranges during insert, which serializes concurrent workers
        // touching adjacent ID ranges in audit_log_target. READ COMMITTED
        // drops gap locks entirely and keeps only record locks — for a
        // seed that generates collision-free IDs, this is safe and
        // typically 2-4x faster under concurrency.
        await conn.query("SET SESSION transaction_isolation = 'READ-COMMITTED'");
      }

      while (remaining.value > 0) {
        const thisBatch = Math.min(BATCH_SIZE, remaining.value);
        remaining.value -= thisBatch;

        const events: GeneratedEvent[] = new Array(thisBatch);
        for (let i = 0; i < thisBatch; i++) {
          const workspaceId = pickWorkspace(
            argv.workspaceId,
            noiseWorkspaceIds,
            effectiveNoiseRatio,
          );
          events[i] = generateEvent(workspaceId, now);
        }

        // MySQL INSERTs and the CH INSERT are independent — fire them in
        // parallel so each batch's wall time is max(mysql, ch) instead of
        // mysql + ch. The two MySQL inserts share `conn`, so they have to
        // stay sequential within their branch. Time each side separately
        // to attribute slowness.
        const insertedAtMs = Date.now();
        await Promise.all([
          conn
            ? (async () => {
                const t0 = Date.now();
                const logTuples = events.map(eventToMySQLLogRow);
                const targetTuples = events.flatMap(eventToMySQLTargetRows);
                await conn.query(AUDIT_LOG_INSERT_SQL, [logTuples]);
                if (targetTuples.length > 0) {
                  await conn.query(AUDIT_LOG_TARGET_INSERT_SQL, [targetTuples]);
                }
                timing.mysqlMs += Date.now() - t0;
                timing.mysqlBatches++;
              })()
            : Promise.resolve(),
          ch
            ? (async () => {
                const t0 = Date.now();
                await ch.insert({
                  table: CH_TABLE,
                  format: "JSONEachRow",
                  values: events.map((e) => eventToCHRow(e, insertedAtMs, e.time + NINETY_DAYS_MS)),
                });
                timing.chMs += Date.now() - t0;
                timing.chBatches++;
              })()
            : Promise.resolve(),
        ]);

        inserted += thisBatch;
        const nowMs = Date.now();
        if (nowMs - lastLog > 2_000) {
          lastLog = nowMs;
          const elapsed = (nowMs - startedAt) / 1000;
          const rps = Math.round(inserted / Math.max(elapsed, 0.001));
          const etaSec = Math.round((argv.count - inserted) / Math.max(rps, 1));
          const mysqlAvg =
            timing.mysqlBatches > 0 ? Math.round(timing.mysqlMs / timing.mysqlBatches) : 0;
          const chAvg = timing.chBatches > 0 ? Math.round(timing.chMs / timing.chBatches) : 0;
          process.stdout.write(
            `\r${inserted.toLocaleString()}/${argv.count.toLocaleString()} — ${rps.toLocaleString()}/s — ETA ${etaSec}s — mysql avg=${mysqlAvg}ms ch avg=${chAvg}ms   `,
          );
        }
      }
    } finally {
      conn?.release();
    }
  });

  await Promise.all(workers);
  process.stdout.write("\n");

  const totalSec = (Date.now() - startedAt) / 1000;
  const mysqlAvg = timing.mysqlBatches > 0 ? Math.round(timing.mysqlMs / timing.mysqlBatches) : 0;
  const chAvg = timing.chBatches > 0 ? Math.round(timing.chMs / timing.chBatches) : 0;
  console.info(
    `seeded ${argv.count.toLocaleString()} events in ${totalSec.toFixed(1)}s (${Math.round(
      argv.count / totalSec,
    ).toLocaleString()}/s)`,
  );
  console.info(
    `per-batch wall time: mysql avg=${mysqlAvg}ms (${timing.mysqlBatches} batches) | ch avg=${chAvg}ms (${timing.chBatches} batches)`,
  );

  await pool?.end();
  await ch?.close();
}

type Stats = { min: number; p50: number; p99: number; max: number; avg: number };

function stats(samples: number[]): Stats {
  const sorted = [...samples].sort((a, b) => a - b);
  const at = (p: number) => sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * p))];
  const avg = sorted.reduce((a, b) => a + b, 0) / sorted.length;
  return {
    min: sorted[0],
    p50: at(0.5),
    p99: at(0.99),
    max: sorted[sorted.length - 1],
    avg: Math.round(avg),
  };
}

// Sample real values from the seeded data so filter benches actually match
// rows. Falls back to placeholder values when the table is empty so the
// driver still runs the query (returns zero rows, latency is still useful).
async function sampleFilterValues(
  pool: Pool,
  workspaceId: string,
  bucket: string,
  lookbackMs: number,
): Promise<{ events: string[]; actorIds: string[]; targetIds: string[] }> {
  const since = Date.now() - lookbackMs;

  const [eventRows] = await pool.query(
    `SELECT DISTINCT event FROM audit_log
     WHERE workspace_id = ? AND bucket = ? AND time >= ?
     LIMIT 5`,
    [workspaceId, bucket, since],
  );
  const events = (eventRows as Array<{ event: string }>).map((r) => r.event);

  const [actorRows] = await pool.query(
    `SELECT actor_id FROM audit_log
     WHERE workspace_id = ? AND bucket = ? AND time >= ? AND actor_type = 'user'
     ORDER BY time DESC LIMIT 5`,
    [workspaceId, bucket, since],
  );
  const actorIds = [...new Set((actorRows as Array<{ actor_id: string }>).map((r) => r.actor_id))];

  const [targetRows] = await pool.query(
    `SELECT t.id FROM audit_log_target t
     JOIN audit_log l ON l.id = t.audit_log_id
     WHERE l.workspace_id = ? AND l.bucket = ? AND l.time >= ?
     LIMIT 5`,
    [workspaceId, bucket, since],
  );
  const targetIds = [...new Set((targetRows as Array<{ id: string }>).map((r) => r.id))];

  return {
    events: events.length ? events : ["key.create"],
    actorIds: actorIds.length ? actorIds : ["user_0"],
    targetIds: targetIds.length ? targetIds : ["key_0"],
  };
}

type Pattern = {
  name: string;
  buildMySQLCount: () => { sql: string; params: unknown[] };
  buildMySQLList: () => { sql: string; params: unknown[] };
  buildCHCount: () => { sql: string; query_params: Record<string, unknown> };
  buildCHList: () => { sql: string; query_params: Record<string, unknown> };
};

function buildPatterns(
  workspaceId: string,
  bucket: string,
  lookbackMs: number,
  filters: { events: string[]; actorIds: string[]; targetIds: string[] },
): Pattern[] {
  const since = Date.now() - lookbackMs;
  const baseMySQLWhere = "workspace_id = ? AND bucket = ? AND time >= ?";
  const baseMySQLParams = [workspaceId, bucket, since];
  const baseCHWhere = `
    workspace_id = {workspaceId: String}
    AND bucket = {bucket: String}
    AND time >= {since: Int64}
  `;
  const baseCHParams = { workspaceId, bucket, since };

  return [
    {
      name: "list (workspace + bucket + time)",
      buildMySQLCount: () => ({
        sql: `SELECT count(*) AS c FROM audit_log WHERE ${baseMySQLWhere}`,
        params: baseMySQLParams,
      }),
      buildMySQLList: () => ({
        sql: `SELECT pk, id, workspace_id, bucket, event, time
              FROM audit_log
              WHERE ${baseMySQLWhere}
              ORDER BY time DESC, pk DESC
              LIMIT 50`,
        params: baseMySQLParams,
      }),
      buildCHCount: () => ({
        sql: `SELECT count() AS c FROM ${CH_TABLE} WHERE ${baseCHWhere}`,
        query_params: baseCHParams,
      }),
      buildCHList: () => ({
        sql: `SELECT event_id, time, event, actor_id, actor_type, \`targets.type\`, \`targets.id\`
              FROM ${CH_TABLE}
              WHERE ${baseCHWhere}
              ORDER BY time DESC, event_id DESC
              LIMIT 50`,
        query_params: baseCHParams,
      }),
    },
    {
      name: "filter by actor (1 user)",
      buildMySQLCount: () => ({
        sql: `SELECT count(*) AS c FROM audit_log
              WHERE ${baseMySQLWhere} AND actor_id = ?`,
        params: [...baseMySQLParams, filters.actorIds[0]],
      }),
      buildMySQLList: () => ({
        sql: `SELECT pk, id, workspace_id, bucket, event, time
              FROM audit_log
              WHERE ${baseMySQLWhere} AND actor_id = ?
              ORDER BY time DESC, pk DESC
              LIMIT 50`,
        params: [...baseMySQLParams, filters.actorIds[0]],
      }),
      buildCHCount: () => ({
        sql: `SELECT count() AS c FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND actor_id = {actorId: String}`,
        query_params: { ...baseCHParams, actorId: filters.actorIds[0] },
      }),
      buildCHList: () => ({
        sql: `SELECT event_id, time, event, actor_id, actor_type, \`targets.type\`, \`targets.id\`
              FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND actor_id = {actorId: String}
              ORDER BY time DESC, event_id DESC
              LIMIT 50`,
        query_params: { ...baseCHParams, actorId: filters.actorIds[0] },
      }),
    },
    {
      name: "filter by event IN (...)",
      buildMySQLCount: () => ({
        sql: `SELECT count(*) AS c FROM audit_log
              WHERE ${baseMySQLWhere} AND event IN (?)`,
        params: [...baseMySQLParams, filters.events],
      }),
      buildMySQLList: () => ({
        sql: `SELECT pk, id, workspace_id, bucket, event, time
              FROM audit_log
              WHERE ${baseMySQLWhere} AND event IN (?)
              ORDER BY time DESC, pk DESC
              LIMIT 50`,
        params: [...baseMySQLParams, filters.events],
      }),
      buildCHCount: () => ({
        sql: `SELECT count() AS c FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND event IN {events: Array(String)}`,
        query_params: { ...baseCHParams, events: filters.events },
      }),
      buildCHList: () => ({
        sql: `SELECT event_id, time, event, actor_id, actor_type, \`targets.type\`, \`targets.id\`
              FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND event IN {events: Array(String)}
              ORDER BY time DESC, event_id DESC
              LIMIT 50`,
        query_params: { ...baseCHParams, events: filters.events },
      }),
    },
    {
      name: "filter by target id",
      // MySQL needs a JOIN through audit_log_target — this is the slow path
      // the redesign was meant to eliminate. CH does it via has(targets.id).
      buildMySQLCount: () => ({
        sql: `SELECT count(*) AS c
              FROM audit_log l
              JOIN audit_log_target t ON l.id = t.audit_log_id
              WHERE l.workspace_id = ? AND l.bucket = ? AND l.time >= ?
                AND t.id = ?`,
        params: [workspaceId, bucket, since, filters.targetIds[0]],
      }),
      buildMySQLList: () => ({
        sql: `SELECT l.pk, l.id, l.workspace_id, l.bucket, l.event, l.time
              FROM audit_log l
              JOIN audit_log_target t ON l.id = t.audit_log_id
              WHERE l.workspace_id = ? AND l.bucket = ? AND l.time >= ?
                AND t.id = ?
              ORDER BY l.time DESC, l.pk DESC
              LIMIT 50`,
        params: [workspaceId, bucket, since, filters.targetIds[0]],
      }),
      buildCHCount: () => ({
        sql: `SELECT count() AS c FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND has(\`targets.id\`, {targetId: String})`,
        query_params: { ...baseCHParams, targetId: filters.targetIds[0] },
      }),
      buildCHList: () => ({
        sql: `SELECT event_id, time, event, actor_id, actor_type, \`targets.type\`, \`targets.id\`
              FROM ${CH_TABLE}
              WHERE ${baseCHWhere} AND has(\`targets.id\`, {targetId: String})
              ORDER BY time DESC, event_id DESC
              LIMIT 50`,
        query_params: { ...baseCHParams, targetId: filters.targetIds[0] },
      }),
    },
    {
      name: "combined: actor + event",
      buildMySQLCount: () => ({
        sql: `SELECT count(*) AS c FROM audit_log
              WHERE ${baseMySQLWhere} AND actor_id = ? AND event IN (?)`,
        params: [...baseMySQLParams, filters.actorIds[0], filters.events],
      }),
      buildMySQLList: () => ({
        sql: `SELECT pk, id, workspace_id, bucket, event, time
              FROM audit_log
              WHERE ${baseMySQLWhere} AND actor_id = ? AND event IN (?)
              ORDER BY time DESC, pk DESC
              LIMIT 50`,
        params: [...baseMySQLParams, filters.actorIds[0], filters.events],
      }),
      buildCHCount: () => ({
        sql: `SELECT count() AS c FROM ${CH_TABLE}
              WHERE ${baseCHWhere}
                AND actor_id = {actorId: String}
                AND event IN {events: Array(String)}`,
        query_params: {
          ...baseCHParams,
          actorId: filters.actorIds[0],
          events: filters.events,
        },
      }),
      buildCHList: () => ({
        sql: `SELECT event_id, time, event, actor_id, actor_type, \`targets.type\`, \`targets.id\`
              FROM ${CH_TABLE}
              WHERE ${baseCHWhere}
                AND actor_id = {actorId: String}
                AND event IN {events: Array(String)}
              ORDER BY time DESC, event_id DESC
              LIMIT 50`,
        query_params: {
          ...baseCHParams,
          actorId: filters.actorIds[0],
          events: filters.events,
        },
      }),
    },
  ];
}

async function bench(argv: Argv) {
  const benchMySQL = argv.backend !== "clickhouse";
  const benchCH = argv.backend !== "mysql";

  const pool = benchMySQL ? createPool(2) : null;
  const ch = benchCH ? createCH() : null;
  const lookbackMs = argv.lookbackHours * 60 * 60 * 1000;

  // Sample real filter values from MySQL so each pattern actually selects
  // rows. If MySQL isn't seeded, fall back to defaults.
  const filters = pool
    ? await sampleFilterValues(pool, argv.workspaceId, argv.bucket, lookbackMs)
    : { events: ["key.create"], actorIds: ["user_0"], targetIds: ["key_0"] };

  if (pool) {
    const [totalRow] = await pool.query(
      "SELECT count(*) AS c FROM audit_log WHERE workspace_id = ?",
      [argv.workspaceId],
    );
    const total = Number((totalRow as Array<{ c: number | string }>)[0]?.c ?? 0);
    console.info(
      `MySQL: workspace_id=${argv.workspaceId} bucket=${argv.bucket} lookback=${argv.lookbackHours}h total_rows=${total.toLocaleString()}`,
    );
  }
  if (ch) {
    const totalCH = await ch.query({
      query: `SELECT count() AS c FROM ${CH_TABLE} WHERE workspace_id = {workspaceId: String}`,
      query_params: { workspaceId: argv.workspaceId },
      format: "JSONEachRow",
    });
    const totalCHRow = (await totalCH.json()) as Array<{ c: number | string }>;
    const total = Number(totalCHRow[0]?.c ?? 0);
    console.info(
      `CH: workspace_id=${argv.workspaceId} bucket=${argv.bucket} lookback=${argv.lookbackHours}h total_rows=${total.toLocaleString()}`,
    );
  }

  console.info(
    `\nfilters used: events=[${filters.events.join(",")}] actor=${filters.actorIds[0]} target=${filters.targetIds[0]}\n`,
  );

  const patterns = buildPatterns(argv.workspaceId, argv.bucket, lookbackMs, filters);

  for (const pattern of patterns) {
    console.info(`\n# pattern: ${pattern.name}`);

    if (pool) {
      const countLatencies: number[] = [];
      const listLatencies: number[] = [];
      const c = pattern.buildMySQLCount();
      const l = pattern.buildMySQLList();
      for (let i = 0; i < argv.runs; i++) {
        let t0 = Date.now();
        await pool.query(c.sql, c.params);
        countLatencies.push(Date.now() - t0);
        t0 = Date.now();
        await pool.query(l.sql, l.params);
        listLatencies.push(Date.now() - t0);
      }
      console.info(`  MySQL count   ${fmtStats(stats(countLatencies))}`);
      console.info(`  MySQL list    ${fmtStats(stats(listLatencies))}`);
    }

    if (ch) {
      const countLatencies: number[] = [];
      const listLatencies: number[] = [];
      const c = pattern.buildCHCount();
      const l = pattern.buildCHList();
      for (let i = 0; i < argv.runs; i++) {
        let t0 = Date.now();
        const res1 = await ch.query({
          query: c.sql,
          query_params: c.query_params,
          format: "JSONEachRow",
        });
        await res1.json();
        countLatencies.push(Date.now() - t0);

        t0 = Date.now();
        const res2 = await ch.query({
          query: l.sql,
          query_params: l.query_params,
          format: "JSONEachRow",
        });
        await res2.json();
        listLatencies.push(Date.now() - t0);
      }
      console.info(`  CH    count   ${fmtStats(stats(countLatencies))}`);
      console.info(`  CH    list    ${fmtStats(stats(listLatencies))}`);
    }
  }

  await pool?.end();
  await ch?.close();
}

function fmtStats(s: Stats): string {
  return `min=${s.min}ms p50=${s.p50}ms p99=${s.p99}ms max=${s.max}ms avg=${s.avg}ms`;
}

async function cleanup(_argv: Argv) {
  console.info("truncating audit_log + audit_log_target + audit_logs_raw_v1...");
  const startedAt = Date.now();

  const pool = createPool(1);
  // TRUNCATE is DDL on InnoDB: constant-time drop+recreate. Fine for a local
  // bench DB; do not run this against a shared or prod database.
  await pool.query("TRUNCATE TABLE audit_log");
  await pool.query("TRUNCATE TABLE audit_log_target");
  await pool.end();

  const ch = createCH();
  await ch.command({ query: `TRUNCATE TABLE IF EXISTS ${CH_TABLE}` });
  await ch.close();

  console.info(`truncated in ${Date.now() - startedAt}ms`);
}

function help() {
  console.info(
    `audit-log-benchmark

usage:
  tsx src/audit-log-benchmark.ts seed    [--count N] [--workspace WS] [--noise-workspaces N] [--noise-ratio R] [--backend both|mysql|clickhouse]
  tsx src/audit-log-benchmark.ts bench   [--workspace WS] [--bucket B] [--lookback-hours H] [--runs R] [--backend both|mysql|clickhouse]
  tsx src/audit-log-benchmark.ts cleanup

defaults:
  --workspace        ${DEFAULT_WORKSPACE_ID}
  --count            ${DEFAULT_COUNT.toLocaleString()}
  --bucket           ${DEFAULT_BUCKET}
  --lookback-hours   ${DEFAULT_LOOKBACK_HOURS}
  --runs             ${DEFAULT_RUNS}
  --noise-workspaces ${DEFAULT_NOISE_WORKSPACES}    (extra workspaces to distribute rows across — 0 disables)
  --noise-ratio      ${DEFAULT_NOISE_RATIO}  (fraction of rows that go to noise workspaces)
  --backend          both     (which backend(s) to seed/bench: both|mysql|clickhouse)

env:
  DATABASE_URL       ${DATABASE_URL}
  CLICKHOUSE_URL     ${CLICKHOUSE_URL}

bench patterns (each runs against MySQL and CH side-by-side):
  - list (workspace + bucket + time)
  - filter by actor (1 user)
  - filter by event IN (...)
  - filter by target id   (MySQL needs JOIN, CH uses has(targets.id))
  - combined: actor + event

notes:
  cleanup wipes audit_log + audit_log_target in MySQL and audit_logs_raw_v1 in
  ClickHouse. local-only.
`,
  );
}

async function main() {
  const argv = parseArgs();
  switch (argv.command) {
    case "seed":
      await seed(argv);
      return;
    case "bench":
      await bench(argv);
      return;
    case "cleanup":
      await cleanup(argv);
      return;
    default:
      help();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
