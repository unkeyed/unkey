import { pathToFileURL } from "node:url";
import { createCommentedPool, staticTagsFromEnv } from "@unkey/db";
import type { Pool, ResultSetHeader } from "mysql2/promise";

const UPDATE_BATCH_SIZE = 10_000;

export type BackfillResult = {
  batches: number[];
  updated: number;
};

export async function backfillDeploymentFirstReadyAt(
  pool: Pool,
  batchSize = UPDATE_BATCH_SIZE,
): Promise<BackfillResult> {
  const batches: number[] = [];
  let updated = 0;

  while (true) {
    const [result] = await pool.query<ResultSetHeader>(
      `UPDATE deployments AS deployment
       JOIN (
         SELECT candidate.pk, candidate.ready_at
         FROM (
           SELECT historical.pk,
             COALESCE(
               finalizing.ended_at,
               CASE
                 WHEN historical.status = 'ready'
                   THEN COALESCE(historical.updated_at, historical.created_at)
                 ELSE NULL
               END
             ) AS ready_at
           FROM deployments AS historical
           LEFT JOIN deployment_steps AS finalizing
             ON finalizing.deployment_id = historical.id
             AND finalizing.step = 'finalizing'
             AND finalizing.ended_at IS NOT NULL
             AND finalizing.error IS NULL
           WHERE historical.first_ready_at IS NULL
             AND (finalizing.pk IS NOT NULL OR historical.status = 'ready')
           ORDER BY historical.pk
           LIMIT ?
         ) AS candidate
       ) AS ready_deployments ON ready_deployments.pk = deployment.pk
       SET deployment.first_ready_at = ready_deployments.ready_at
       WHERE deployment.first_ready_at IS NULL`,
      [batchSize],
    );

    batches.push(result.affectedRows);
    updated += result.affectedRows;
    console.info("progress", { updated });

    if (result.affectedRows < batchSize) {
      break;
    }
  }

  return { batches, updated };
}

/**
 * Backfills deployments.first_ready_at after the nullable column is deployed.
 * A successful finalizing step proves that a historical deployment reached
 * ready because the worker sets ready before completing that step. Its ended_at
 * is the closest durable upper-bound approximation of the missing ready event.
 * A deployment that is still ready uses updated_at, or created_at when legacy
 * data has no updated_at. Existing markers and rows without either proof remain
 * unchanged.
 *
 * Run before enabling readers, then run again after writers that omit the marker
 * have stopped:
 * `mise exec -- pnpm --dir=web/tools/migrate deployment-first-ready-at`
 */
async function main(): Promise<void> {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool(
    { uri: databaseUrl },
    staticTagsFromEnv("deployment-first-ready-at-migration"),
  );

  try {
    const result = await backfillDeploymentFirstReadyAt(pool);
    console.info("Deployment first-ready-at migration finished", result);
  } finally {
    await pool.end();
  }
}

const scriptPath = process.argv[1];
if (scriptPath && import.meta.url === pathToFileURL(scriptPath).href) {
  main().catch((error: unknown) => {
    console.error("Deployment first-ready-at migration failed", error);
    process.exitCode = 1;
  });
}
