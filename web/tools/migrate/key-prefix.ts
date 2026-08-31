import { pathToFileURL } from "node:url";
import { and, createCommentedPool, drizzle, eq, gt, schema, staticTagsFromEnv } from "@unkey/db";

const READ_BATCH_SIZE = 1_000;

/**
 * Splits the combined `prefix_abcd` value stored by the legacy key generator.
 */
export function splitLegacyKeyStart(start: string): { prefix: string; start: string } | null {
  const separatorIndex = start.length - 5;
  if (separatorIndex < 1 || start[separatorIndex] !== "_") {
    return null;
  }

  return {
    prefix: start.slice(0, separatorIndex),
    start: start.slice(-4),
  };
}

/**
 * Backfills keys.prefix and removes the prefix from keys.start. The migration
 * skips rows that already have a prefix and rows without a legacy prefix.
 *
 * The migration is safe to run more than once. Run it again after all key
 * writers store the prefix directly.
 *
 * Run from the repository root with DRIZZLE_DATABASE_URL set:
 * `mise exec -- pnpm --dir=web/tools/migrate key-prefix`
 */
async function main(): Promise<void> {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool({ uri: databaseUrl }, staticTagsFromEnv("key-prefix-migration"));
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    let cursor = 0;
    let scanned = 0;
    let updated = 0;

    while (true) {
      const rows = await db
        .select({ pk: schema.keys.pk, start: schema.keys.start })
        .from(schema.keys)
        .where(and(gt(schema.keys.pk, cursor), eq(schema.keys.prefix, "")))
        .orderBy(schema.keys.pk)
        .limit(READ_BATCH_SIZE);

      const lastRow = rows.at(-1);
      if (!lastRow) {
        break;
      }

      cursor = lastRow.pk;
      scanned += rows.length;

      for (const row of rows) {
        const splitStart = splitLegacyKeyStart(row.start);
        if (splitStart === null) {
          continue;
        }

        const result = await db
          .update(schema.keys)
          .set(splitStart)
          .where(and(eq(schema.keys.pk, row.pk), eq(schema.keys.prefix, "")));
        updated += result[0].affectedRows;
      }

      console.info("progress", { scanned, updated });
    }

    console.info("Key prefix migration finished", { scanned, updated });
  } finally {
    await pool.end();
  }
}

const scriptPath = process.argv[1];
if (scriptPath && import.meta.url === pathToFileURL(scriptPath).href) {
  main().catch((error: unknown) => {
    console.error("Key prefix migration failed", error);
    process.exitCode = 1;
  });
}
