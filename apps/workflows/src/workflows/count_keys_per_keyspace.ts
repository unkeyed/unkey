import { WorkflowEntrypoint, type WorkflowEvent, type WorkflowStep } from "cloudflare:workers";

import { and, count, createConnection, eq, isNull, schema } from "../lib/db";
import type { Env } from "../lib/env";

// User-defined params passed to your workflow
// biome-ignore lint/complexity/noBannedTypes: we just don't have any params here
type Params = {};

// <docs-tag name="workflow-entrypoint">
export class CountKeys extends WorkflowEntrypoint<Env, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    const now = event.timestamp.getTime();

    const db = createConnection({
      host: this.env.DATABASE_HOST,
      username: this.env.DATABASE_USERNAME,
      password: this.env.DATABASE_PASSWORD,
    });

    let done = false;

    while (!done) {
      /**
       * I know all of this is in a single step, which is stupid and does not use steps as intended.
       * But they have a 512 step limit and we need like 30k...
       */
      await step.do("fetch keyspaces", async () => {
        const keySpaces = await db.query.keyAuth.findMany({
          where: (table, { or, and, isNull, lt }) =>
            and(
              isNull(table.deletedAt),
              or(isNull(table.sizeLastUpdatedAt), lt(table.sizeLastUpdatedAt, now - 600_000)),
            ),
          orderBy: (table, { asc }) => asc(table.sizeLastUpdatedAt),
          limit: 490, // we can do 1000 subrequests and need 2 per keyspace + this requests
        });
        if (keySpaces.length === 0) {
          done = true;
        }
        console.info(`found ${keySpaces.length} key spaces`);

        for (const keySpace of keySpaces) {
          const rows = await db
            .select({ count: count() })
            .from(schema.keys)
            .where(and(eq(schema.keys.keyAuthId, keySpace.id), isNull(schema.keys.deletedAt)));

          await db
            .update(schema.keyAuth)
            .set({
              sizeApprox: rows.at(0)?.count ?? 0,
              sizeLastUpdatedAt: Date.now(),
            })
            .where(eq(schema.keyAuth.id, keySpace.id));
        }
        // this just prints on the cf dashboard, we don't use the return value
        return { keySpaces: keySpaces.length };
      });
    }
    await step.do("heartbeat", async () => {
      await fetch(this.env.HEARTBEAT_URL_COUNT_KEYS);
    });
  }
}
