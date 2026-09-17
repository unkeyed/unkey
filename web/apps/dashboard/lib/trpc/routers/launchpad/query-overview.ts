import { clickhouse } from "@/lib/clickhouse";
import { and, count, db, eq, inArray, isNull, schema, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const HOUR_MS = 60 * 60 * 1000;
const DEFAULT_PROJECT_SLUG = "default";

export const launchpadItem = z.object({
  id: z.string(),
  name: z.string(),
  kind: z.enum(["keyspace", "ratelimit"]),
  projectId: z.string().nullable(),
  /** Keys in the keyspace; always 0 for a ratelimit namespace. */
  keyCount: z.number(),
  total: z.number(),
  failed: z.number(),
  buckets: z.array(z.object({ ok: z.number(), bad: z.number() })),
});

export const launchpadOverview = z.object({
  windowHours: z.number(),
  items: z.array(launchpadItem),
  projects: z.array(
    z.object({ id: z.string(), name: z.string(), slug: z.string(), isDefault: z.boolean() }),
  ),
  identityCount: z.number(),
});

export type LaunchpadItem = z.infer<typeof launchpadItem>;
export type LaunchpadOverview = z.infer<typeof launchpadOverview>;

const bucketRow = z.object({
  resource_id: z.string(),
  bucket: z.number(),
  ok: z.number(),
  bad: z.number(),
});

/**
 * Both rollups are AggregatingMergeTree keyed per hour, so one grouped read per
 * product covers every resource in the workspace. Fanning out a per-resource
 * timeseries query instead would be one ClickHouse round-trip per keyspace,
 * and a migrated workspace can hold thousands.
 */
function verificationBuckets(workspaceId: string, startTime: number, endTime: number) {
  return clickhouse.querier.query({
    query: `
      SELECT
        key_space_id AS resource_id,
        toUnixTimestamp64Milli(CAST(toStartOfHour(time) AS DateTime64(3))) AS bucket,
        toInt64(sumIf(count, outcome = 'VALID')) AS ok,
        toInt64(sumIf(count, outcome != 'VALID')) AS bad
      FROM default.key_verifications_per_hour_v3
      WHERE workspace_id = {workspaceId: String}
        AND time >= toStartOfHour(fromUnixTimestamp64Milli({startTime: Int64}))
        AND time <= fromUnixTimestamp64Milli({endTime: Int64})
      GROUP BY resource_id, bucket`,
    params: z.object({
      workspaceId: z.string(),
      startTime: z.int(),
      endTime: z.int(),
    }),
    schema: bucketRow,
  })({ workspaceId, startTime, endTime });
}

function ratelimitBuckets(workspaceId: string, startTime: number, endTime: number) {
  return clickhouse.querier.query({
    query: `
      SELECT
        namespace_id AS resource_id,
        toUnixTimestamp64Milli(CAST(toStartOfHour(time) AS DateTime64(3))) AS bucket,
        toInt64(sum(passed)) AS ok,
        toInt64(sum(total) - sum(passed)) AS bad
      FROM default.ratelimits_per_hour_v2
      WHERE workspace_id = {workspaceId: String}
        AND time >= toStartOfHour(fromUnixTimestamp64Milli({startTime: Int64}))
        AND time <= fromUnixTimestamp64Milli({endTime: Int64})
      GROUP BY resource_id, bucket`,
    params: z.object({
      workspaceId: z.string(),
      startTime: z.int(),
      endTime: z.int(),
    }),
    schema: bucketRow,
  })({ workspaceId, startTime, endTime });
}

export const queryLaunchpadOverview = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ windowHours: z.int().min(1).max(168).default(24) }).optional())
  .output(launchpadOverview)
  .query(async ({ ctx, input }) => {
    const windowHours = input?.windowHours ?? 24;
    const endTime = Date.now();
    const startTime = endTime - windowHours * HOUR_MS;
    const workspaceId = ctx.workspace.id;

    const [apis, namespaces, projects, identities] = await Promise.all([
      db.query.apis.findMany({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)),
        columns: { id: true, name: true, keyAuthId: true, projectId: true },
      }),
      db.query.ratelimitNamespaces.findMany({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)),
        columns: { id: true, name: true, projectId: true },
      }),
      db.query.projects.findMany({
        where: (table, { eq }) => eq(table.workspaceId, workspaceId),
        columns: { id: true, name: true, slug: true },
      }),
      db
        .select({ count: sql<number>`count(*)` })
        .from(schema.identities)
        .where(
          and(eq(schema.identities.workspaceId, workspaceId), eq(schema.identities.deleted, false)),
        ),
    ]);

    const keyspaceIds = apis
      .map((api) => api.keyAuthId)
      .filter((id): id is string => Boolean(id));

    const keyCounts = new Map<string, number>();
    if (keyspaceIds.length > 0) {
      const rows = await db
        .select({ keyAuthId: schema.keys.keyAuthId, count: count(schema.keys.id) })
        .from(schema.keys)
        .where(
          and(
            eq(schema.keys.workspaceId, workspaceId),
            inArray(schema.keys.keyAuthId, keyspaceIds),
            isNull(schema.keys.deletedAtM),
          ),
        )
        .groupBy(schema.keys.keyAuthId);
      for (const row of rows) {
        keyCounts.set(row.keyAuthId, Number(row.count));
      }
    }

    const [verifications, ratelimits] = await Promise.all([
      verificationBuckets(workspaceId, startTime, endTime),
      ratelimitBuckets(workspaceId, startTime, endTime),
    ]);

    if (verifications.err || ratelimits.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to load activity for the workspace's resources.",
      });
    }

    const firstBucket = Math.floor(startTime / HOUR_MS) * HOUR_MS;
    const bucketCount = windowHours + 1;
    const series = new Map<string, { ok: number; bad: number }[]>();
    const add = (rows: z.infer<typeof bucketRow>[]) => {
      for (const row of rows) {
        const index = Math.round((row.bucket - firstBucket) / HOUR_MS);
        if (index < 0 || index >= bucketCount) {
          continue;
        }
        let buckets = series.get(row.resource_id);
        if (!buckets) {
          buckets = Array.from({ length: bucketCount }, () => ({ ok: 0, bad: 0 }));
          series.set(row.resource_id, buckets);
        }
        buckets[index].ok += row.ok;
        buckets[index].bad += row.bad;
      }
    };
    add(verifications.val);
    add(ratelimits.val);

    const empty = () => Array.from({ length: bucketCount }, () => ({ ok: 0, bad: 0 }));
    const toItem = (
      id: string,
      seriesId: string,
      name: string,
      kind: LaunchpadItem["kind"],
      projectId: string | null,
      keyCount: number,
    ): LaunchpadItem => {
      const buckets = series.get(seriesId) ?? empty();
      let total = 0;
      let failed = 0;
      for (const bucket of buckets) {
        total += bucket.ok + bucket.bad;
        failed += bucket.bad;
      }
      return { id, name, kind, projectId, keyCount, total, failed, buckets };
    };

    const items: LaunchpadItem[] = [
      ...apis.map((api) =>
        toItem(
          api.id,
          api.keyAuthId ?? api.id,
          api.name,
          "keyspace",
          api.projectId ?? null,
          api.keyAuthId ? (keyCounts.get(api.keyAuthId) ?? 0) : 0,
        ),
      ),
      ...namespaces.map((namespace) =>
        toItem(namespace.id, namespace.id, namespace.name, "ratelimit", namespace.projectId ?? null, 0),
      ),
    ];

    items.sort((a, b) => b.total - a.total || a.name.localeCompare(b.name));

    return {
      windowHours,
      items,
      projects: projects.map((project) => ({
        ...project,
        isDefault: project.slug === DEFAULT_PROJECT_SLUG,
      })),
      identityCount: Number(identities[0]?.count ?? 0),
    };
  });
