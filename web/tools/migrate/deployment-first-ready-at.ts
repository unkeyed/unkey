import { pathToFileURL } from "node:url";
import { createCommentedPool, staticTagsFromEnv } from "@unkey/db";
import type { Pool, ResultSetHeader, RowDataPacket } from "mysql2/promise";

const UPDATE_BATCH_SIZE = 10_000;

export type BackfillResult = {
  batches: number[];
  updated: number;
  ambiguous: number;
};

export async function backfillDeploymentFirstReadyAt(
  pool: Pool,
  batchSize = UPDATE_BATCH_SIZE,
): Promise<BackfillResult> {
  const batches: number[] = [];
  let updated = 0;

  while (true) {
    const [result] = await pool.query<ResultSetHeader>(
      `UPDATE deployments
       SET first_ready_at = COALESCE(updated_at, created_at)
       WHERE first_ready_at IS NULL AND status = 'ready'
       ORDER BY pk
       LIMIT ?`,
      [batchSize],
    );

    batches.push(result.affectedRows);
    updated += result.affectedRows;
    console.info("progress", { updated });

    if (result.affectedRows < batchSize) {
      break;
    }
  }

  const [[{ ambiguous }]] = await pool.query<(RowDataPacket & { ambiguous: number })[]>(
    `SELECT COUNT(*) AS ambiguous
     FROM deployments AS deployment
     INNER JOIN deployment_steps AS finalizing
       ON finalizing.deployment_id = deployment.id
       AND finalizing.step = 'finalizing'
       AND finalizing.ended_at IS NOT NULL
       AND finalizing.error IS NULL
     WHERE deployment.first_ready_at IS NULL AND deployment.status <> 'ready'`,
  );
  return { batches, updated, ambiguous };
}

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
    console.info("Deployment first-ready-at backfill result", result);
    if (result.ambiguous > 0) {
      throw new Error(
        `${result.ambiguous} historical deployments need readiness review before enabling first_ready_at readers`,
      );
    }
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
