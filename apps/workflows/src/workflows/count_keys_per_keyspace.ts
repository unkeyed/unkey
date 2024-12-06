import { WorkflowEntrypoint, type WorkflowEvent, type WorkflowStep } from "cloudflare:workers";

import { and, createConnection, eq, isNull, schema, sql } from "../lib/db";
import type { Env } from "../lib/env";

// User-defined params passed to your workflow
// biome-ignore lint/complexity/noBannedTypes: we just don't have any params here
type Params = {};

// <docs-tag name="workflow-entrypoint">
export class CountKeys extends WorkflowEntrypoint<Env, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    const now = event.timestamp.getUTCDate();

    const db = createConnection({
      host: this.env.DATABASE_HOST,
      username: this.env.DATABASE_USERNAME,
      password: this.env.DATABASE_PASSWORD,
    });

    let cursor = "";

    do {
      const keySpaces = await step.do(`fetch keyspaces - cursor:${cursor}`, async () =>
        db.query.keyAuth.findMany({
          where: (table, { gt, and, isNull, lt }) =>
            and(
              gt(table.id, cursor),
              isNull(table.deletedAt),
              lt(table.sizeLastUpdatedAt, now - 60_000),
            ), // if older than 60s
          limit: 100,
        }),
      );

      for (const keySpace of keySpaces) {
        const count = await db
          .select({ count: sql<string>`count(*)` })
          .from(schema.keys)
          .where(and(eq(schema.keys.keyAuthId, keySpace.id), isNull(schema.keys.deletedAt)));

        keySpace.sizeApprox = Number.parseInt(count?.at(0)?.count ?? "0");
        keySpace.sizeLastUpdatedAt = Date.now();

        await db
          .update(schema.keyAuth)
          .set({
            sizeApprox: keySpace.sizeApprox,
            sizeLastUpdatedAt: keySpace.sizeLastUpdatedAt,
          })
          .where(eq(schema.keyAuth.id, keySpace.id));
      }
      cursor = keySpaces.at(-1)?.id ?? "";
    } while (cursor);
    await step.do("heartbeat", async () => {
      await fetch(this.env.HEARTBEAT_URL_COUNT_KEYS);
    });
  }
}
