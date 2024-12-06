import { WorkflowEntrypoint, type WorkflowEvent, type WorkflowStep } from "cloudflare:workers";

import { newId } from "@unkey/id";
import { createConnection, eq, schema } from "../lib/db";
import type { Env } from "../lib/env";

// User-defined params passed to your workflow
// biome-ignore lint/complexity/noBannedTypes: we just don't have any params here
type Params = {};

// <docs-tag name="workflow-entrypoint">
export class RefillRemaining extends WorkflowEntrypoint<Env, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    // Set up last day of month so if refillDay is after last day of month, Key will be refilled today.
    const lastDayOfMonth = new Date(
      event.timestamp.getFullYear(),
      event.timestamp.getMonth() + 1,
      0,
    ).getDate();
    const today = event.timestamp.getUTCDate();
    const db = createConnection({
      host: this.env.DATABASE_HOST,
      username: this.env.DATABASE_USERNAME,
      password: this.env.DATABASE_PASSWORD,
    });
    const BUCKET_NAME = "unkey_mutations";

    // If refillDay is after last day of month, refillDay will be today.
    const keys = await step.do(
      "fetch keys",
      async () =>
        await db.query.keys.findMany({
          where: (table, { isNotNull, isNull, and, gt, or, eq }) => {
            const baseConditions = and(
              isNull(table.deletedAt),
              isNotNull(table.refillAmount),
              gt(table.refillAmount, table.remaining),
              or(isNull(table.refillDay), eq(table.refillDay, today)),
            );

            if (today === lastDayOfMonth) {
              return and(baseConditions, gt(table.refillDay, today));
            }

            return baseConditions;
          },
        }),
    );

    console.info(`found ${keys.length} keys with refill set for today`);

    for (const key of keys) {
      const bucketId = await step.do(
        `fetch bucketId for ${key.workspaceId}::${BUCKET_NAME}`,
        async () => {
          const found = await db.query.auditLogBucket.findFirst({
            where: (table, { eq, and }) =>
              and(eq(table.workspaceId, key.workspaceId), eq(table.name, BUCKET_NAME)),
            columns: {
              id: true,
            },
          });

          if (found) {
            return found.id;
          }
          const id = newId("auditLogBucket");
          await db.insert(schema.auditLogBucket).values({
            id,
            workspaceId: key.workspaceId,
            name: BUCKET_NAME,
          });
          return id;
        },
      );

      await step.do(`refilling ${key.id}`, async () => {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.keys)
            .set({
              remaining: key.refillAmount,
              lastRefillAt: event.timestamp,
            })
            .where(eq(schema.keys.id, key.id));

          const auditLogId = newId("auditLog");
          await tx.insert(schema.auditLog).values({
            id: auditLogId,
            workspaceId: key.workspaceId,
            bucketId: bucketId,
            time: event.timestamp.getTime(),
            event: "key.update",
            actorId: "trigger",
            actorType: "system",
            display: `Refilled ${key.id} to ${key.refillAmount}`,
          });
          await tx.insert(schema.auditLogTarget).values([
            {
              type: "workspace",
              id: key.workspaceId,
              workspaceId: key.workspaceId,
              bucketId: bucketId,
              auditLogId,
              displayName: `workspace ${key.workspaceId}`,
            },
            {
              type: "key",
              id: key.id,
              workspaceId: key.workspaceId,
              bucketId: bucketId,
              auditLogId,
              displayName: `key ${key.id}`,
            },
          ]);
        });
        return { keyId: key.id };
      });
    }

    await step.do("heartbeat", async () => {
      await fetch(this.env.HEARTBEAT_URL_REFILLS);
    });

    return {
      refillKeyIds: keys.map((k) => k.id),
    };
  }
}
