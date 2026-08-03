import { and, createCommentedPool, drizzle, eq, schema, staticTagsFromEnv } from "@unkey/db";

/**
 * Backfills the explicit production flag for environments created before the
 * column existed. The update is batched for Vitess and is safe to rerun.
 *
 * Rollout sequence:
 * 1. Deploy the `is_production` column.
 * 2. Run this migration before deploying readers of the new flag.
 * 3. Run it again after all previous app writers have stopped.
 *
 * Run from the repository root with `DRIZZLE_DATABASE_URL` set:
 * `mise exec -- pnpm --dir=web/tools/migrate environment-production`
 */
const UPDATE_BATCH_SIZE = 10_000;

async function main() {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool(
    { uri: databaseUrl },
    staticTagsFromEnv("environment-production-migration"),
  );
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    let updated = 0;
    while (true) {
      const result = await db
        .update(schema.environments)
        .set({ isProduction: true })
        .where(
          and(
            eq(schema.environments.slug, "production"),
            eq(schema.environments.isProduction, false),
          ),
        )
        .limit(UPDATE_BATCH_SIZE);
      const affectedRows = result[0].affectedRows;
      updated += affectedRows;

      if (affectedRows < UPDATE_BATCH_SIZE) {
        break;
      }
    }

    console.info("Environment production migration finished", { updated });
  } finally {
    await pool.end();
  }
}

main().catch((error: unknown) => {
  console.error("Environment production migration failed", error);
  process.exitCode = 1;
});
