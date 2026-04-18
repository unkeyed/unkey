import { drizzle, eq, schema, sql } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

/**
 * One-shot rollout for the sentinel-tiers feature.
 *
 * Precondition: the schema migration has already run, so `sentinel_tiers`
 * and `sentinel_subscriptions` exist and `sentinels.subscription_id` has
 * been added as NULLABLE.
 *
 * This script does two things, both idempotent:
 *   1. Seeds the sentinel_tiers catalog (st-250, st-500, st-1000,
 *      st-2000 at version 2026-04). INSERT IGNORE so re-runs are safe.
 *   2. For every sentinel without a subscription_id, inserts a
 *      sentinel_subscriptions row at the free tier (st-250) with the
 *      sentinel's current replica count and denormalized workspace_id
 *      / region_id, then repoints the sentinel at the new subscription.
 *
 * Each sentinel is processed in its own transaction so a partial run is
 * resumable: re-running picks up only the sentinels still missing a
 * subscription_id.
 *
 * After this completes, tighten the schema with:
 *   ALTER TABLE `sentinels` MODIFY COLUMN `subscription_id` varchar(64) NOT NULL;
 */

const DEFAULT_TIER_ID = "st-250";
const DEFAULT_TIER_VERSION = "2026-04";

const TIER_CATALOG: Array<{
  id: string;
  tierId: string;
  version: string;
  cpuMillicores: number;
  memoryMib: number;
  pricePerSecond: string;
}> = [
  {
    id: "stier_st250",
    tierId: "st-250",
    version: "2026-04",
    cpuMillicores: 250,
    memoryMib: 256,
    pricePerSecond: "0",
  },
  {
    id: "stier_st500",
    tierId: "st-500",
    version: "2026-04",
    cpuMillicores: 500,
    memoryMib: 512,
    pricePerSecond: "0.00000694",
  },
  {
    id: "stier_st1000",
    tierId: "st-1000",
    version: "2026-04",
    cpuMillicores: 1000,
    memoryMib: 1024,
    pricePerSecond: "0.00001389",
  },
  {
    id: "stier_st2000",
    tierId: "st-2000",
    version: "2026-04",
    cpuMillicores: 2000,
    memoryMib: 2048,
    pricePerSecond: "0.00002778",
  },
];

async function main() {
  const url = process.env.DRIZZLE_DATABASE_URL;
  if (!url) {
    throw new Error("DRIZZLE_DATABASE_URL is required");
  }
  const conn = await mysql.createConnection(url);
  await conn.ping();
  const db = drizzle(conn, { schema, mode: "default" });

  // Seed the tier catalog first. INSERT IGNORE on the unique
  // (tier_id, version) key makes re-runs no-ops. Using a fixed
  // effective_from of "now" at first seed is fine; subsequent runs skip.
  const now = Date.now();
  await db
    .insert(schema.sentinelTiers)
    .values(
      TIER_CATALOG.map((t) => ({
        id: t.id,
        tierId: t.tierId,
        version: t.version,
        cpuMillicores: t.cpuMillicores,
        memoryMib: t.memoryMib,
        pricePerSecond: t.pricePerSecond,
        effectiveFrom: now,
      })),
    )
    // `onDuplicateKeyUpdate` is required by drizzle even when we mean
    // "ignore duplicates" — setting `id` to its own current value is a
    // safe no-op that keeps existing rows untouched.
    .onDuplicateKeyUpdate({ set: { id: sql`id` } });
  console.info(`seeded tier catalog (${TIER_CATALOG.length} entries, idempotent)`);

  // Resolve the default tier. All backfilled subscriptions point at this
  // row — prices are denormalized onto the subscription so future catalog
  // edits don't touch historical bills.
  const tier = await db.query.sentinelTiers.findFirst({
    where: (t, { and: a, eq: e }) =>
      a(e(t.tierId, DEFAULT_TIER_ID), e(t.version, DEFAULT_TIER_VERSION)),
  });
  if (!tier) {
    throw new Error(
      `default tier ${DEFAULT_TIER_ID}/${DEFAULT_TIER_VERSION} missing after seed — something's wrong`,
    );
  }

  const sentinels = await db.query.sentinels.findMany({
    where: (s, { or: o, eq: e, isNull: n }) => o(n(s.subscriptionId), e(s.subscriptionId, "")),
    columns: {
      id: true,
      workspaceId: true,
      regionId: true,
      desiredReplicas: true,
      createdAt: true,
    },
  });

  console.info(`found ${sentinels.length} sentinels needing backfill`);
  if (sentinels.length === 0) {
    await conn.end();
    return;
  }

  let done = 0;
  for (const s of sentinels) {
    const subscriptionId = newId("sentinelSubscription");
    await db.transaction(async (tx) => {
      await tx.insert(schema.sentinelSubscriptions).values({
        id: subscriptionId,
        sentinelId: s.id,
        workspaceId: s.workspaceId,
        regionId: s.regionId,
        tierId: tier.tierId,
        tierVersion: tier.version,
        cpuMillicores: tier.cpuMillicores,
        memoryMib: tier.memoryMib,
        replicas: s.desiredReplicas,
        pricePerSecond: tier.pricePerSecond,
        // Backdate to the sentinel's own creation so the age of the
        // subscription reflects how long this sentinel has existed
        // rather than starting the clock at migration time.
        createdAt: s.createdAt ?? Date.now(),
      });
      await tx
        .update(schema.sentinels)
        .set({ subscriptionId, updatedAt: Date.now() })
        .where(eq(schema.sentinels.id, s.id));
    });
    done++;
    if (done % 50 === 0) {
      console.info(`  ${done}/${sentinels.length}`);
    }
  }

  console.info(`done. backfilled ${done} sentinels`);
  await conn.end();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
