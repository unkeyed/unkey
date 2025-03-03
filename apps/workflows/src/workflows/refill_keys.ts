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
    let now = new Date();
    try {
      // cf stopped sending valid `Date` objects for some reason, so we fall back to Date.now()
      now = event.timestamp;
    } catch {}

    // Set up last day of month so if refillDay is after last day of month, Key will be refilled today.
    const lastDayOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    const today = now.getUTCDate();
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
              isNull(table.deletedAtM),
              isNotNull(table.refillAmount),
              gt(table.refillAmount, table.remaining),
              or(isNull(table.refillDay), eq(table.refillDay, today)),
            );

            if (today === lastDayOfMonth) {
              return and(baseConditions, gt(table.refillDay, today));
            }

            return baseConditions;
          },
          with: {
            workspace: {
              with: {
                auditLogBuckets: {
                  where: (table, { eq }) => eq(table.name, BUCKET_NAME),
                },
              },
            },
          },
        }),
    );

    console.info(`found ${keys.length} keys with refill set for today`);

    for (const key of keys) {
      const bucket = key.workspace.auditLogBuckets.at(0);
      if (!bucket) {
        throw new Error(`workspace ${key.workspace.id} has no audit log bucket ${BUCKET_NAME}`);
      }

      await step.do(`refilling ${key.id}`, async () => {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.keys)
            .set({
              remaining: key.refillAmount,
              lastRefillAt: now,
            })
            .where(eq(schema.keys.id, key.id));

          const auditLogId = newId("auditLog");
          await tx.insert(schema.auditLog).values({
            id: auditLogId,
            workspaceId: key.workspaceId,
            bucketId: bucket.id,
            time: now.getTime(),
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
              bucketId: bucket.id,
              auditLogId,
              displayName: `workspace ${key.workspaceId}`,
            },
            {
              type: "key",
              id: key.id,
              workspaceId: key.workspaceId,
              bucketId: bucket.id,
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
