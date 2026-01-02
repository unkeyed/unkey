import { WorkflowEntrypoint, type WorkflowEvent, type WorkflowStep } from "cloudflare:workers";

import { and, count, createConnection, eq, isNull, schema } from "../lib/db";
import type { Env } from "../lib/env";

// User-defined params passed to your workflow
// biome-ignore lint/complexity/noBannedTypes: we just don't have any params here
type Params = {};

// <docs-tag name="workflow-entrypoint">
export class CountKeys extends WorkflowEntrypoint<Env, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    let now = Date.now();
    try {
      // cf stopped sending valid `Date` objects for some reason, so we fall back to Date.now()
      now = event.timestamp.getTime();
    } catch {}

    const db = createConnection({
      host: this.env.DATABASE_HOST,
      username: this.env.DATABASE_USERNAME,
      password: this.env.DATABASE_PASSWORD,
    });

    const keySpaces = await step.do("fetch outdated keyspaces", async () =>
      db.query.keyAuth.findMany({
        where: (table, { or, and, isNull, lt, not, like }) =>
          and(
            isNull(table.deletedAtM),
            or(isNull(table.sizeLastUpdatedAt), lt(table.sizeLastUpdatedAt, now - 600_000)),
            not(like(table.id, "test_%")),
          ),
        columns: {
          id: true,
        },
        orderBy: (table, { asc }) => asc(table.sizeLastUpdatedAt),
        limit: 200,
      }),
    );
    console.info(`found ${keySpaces.length} key spaces`);

    for (const keySpace of keySpaces) {
      const rows = await step.do(`count keys for ${keySpace.id} `, async () =>
        db
          .select({ count: count() })
          .from(schema.keys)
          .where(and(eq(schema.keys.keyAuthId, keySpace.id), isNull(schema.keys.deletedAtM))),
      );

      await step.do(`update ${keySpace.id}`, async () =>
        db
          .update(schema.keyAuth)
          .set({
            sizeApprox: rows.at(0)?.count ?? 0,
            sizeLastUpdatedAt: Date.now(),
          })
          .where(eq(schema.keyAuth.id, keySpace.id)),
      );
    }

    await step.do("heartbeat", async () => {
      await fetch(this.env.HEARTBEAT_URL_COUNT_KEYS);
    });
    // this just prints on the cf dashboard, we don't use the return value
    return { keySpaces: keySpaces.length };
  }
}
