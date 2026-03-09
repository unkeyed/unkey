import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

/**
 * Migrates region_config JSON from app_runtime_settings into
 * individual app_regional_settings rows.
 *
 * For each app_runtime_settings row with a non-empty region_config,
 * creates one app_regional_settings row per region entry.
 *
 * Safe to run multiple times — uses INSERT IGNORE to skip existing rows.
 */
async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const runtimeSettings = await db.query.appRuntimeSettings.findMany({
    columns: {
      workspaceId: true,
      appId: true,
      environmentId: true,
      regionConfig: true,
    },
  });

  console.info(`Found ${runtimeSettings.length} runtime settings rows`);

  let created = 0;
  let skipped = 0;

  for (const row of runtimeSettings) {
    const regionConfig = (row.regionConfig ?? {}) as Record<string, number>;
    const regions = Object.entries(regionConfig);

    if (regions.length === 0) {
      skipped++;
      continue;
    }

    for (const [regionId, replicas] of regions) {
      try {
        await db.insert(schema.appRegionalSettings).values({
          workspaceId: row.workspaceId,
          appId: row.appId,
          environmentId: row.environmentId,
          regionId,
          replicas,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        });
        created++;
        console.info(
          `Created: app=${row.appId} env=${row.environmentId} region=${regionId} replicas=${replicas}`,
        );
      } catch (err: unknown) {
        if (err instanceof Error && err.message.includes("Duplicate entry")) {
          console.info(
            `Skipped (already exists): app=${row.appId} env=${row.environmentId} region=${regionId}`,
          );
          skipped++;
        } else {
          throw err;
        }
      }
    }
  }

  console.info(`Migration complete. Created: ${created}, Skipped: ${skipped}`);
  await conn.end();
}

main();
