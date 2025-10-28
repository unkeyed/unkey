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
    const creditsToRefill = await step.do("fetch credits", async () => {
      const { and, or, eq, isNull, isNotNull, gt } = await import("@unkey/db");

      // Build the refill conditions
      let refillConditions = and(
        isNotNull(schema.credits.refillAmount),
        gt(schema.credits.refillAmount, schema.credits.remaining),
        or(isNull(schema.credits.refillDay), eq(schema.credits.refillDay, today)),
      );

      if (today === lastDayOfMonth) {
        refillConditions = and(refillConditions, gt(schema.credits.refillDay, today));
      }

      // Query with joins to filter out deleted keys/identities at DB level
      const results = await db
        .select({
          id: schema.credits.id,
          workspaceId: schema.credits.workspaceId,
          keyId: schema.credits.keyId,
          identityId: schema.credits.identityId,
          refillAmount: schema.credits.refillAmount,
        })
        .from(schema.credits)
        .leftJoin(schema.keys, eq(schema.credits.keyId, schema.keys.id))
        .leftJoin(schema.identities, eq(schema.credits.identityId, schema.identities.id))
        .where(
          and(
            refillConditions,
            // Filter: if it's a key credit, key must not be deleted
            // OR if it's an identity credit, identity must not be deleted
            or(
              and(isNotNull(schema.credits.keyId), isNull(schema.keys.deletedAtM)),
              and(isNotNull(schema.credits.identityId), eq(schema.identities.deleted, false)),
            ),
          ),
        );

      return results.map((c) => ({
        id: c.id,
        workspaceId: c.workspaceId,
        keyId: c.keyId,
        identityId: c.identityId,
        refillAmount: c.refillAmount as number,
      }));
    });

    console.info(`found ${creditsToRefill.length} credits with refill set for today`);

    for (const credit of creditsToRefill) {
      const resourceType = credit.keyId ? "key" : "identity";
      const resourceId = credit.keyId || credit.identityId;
      await step.do(`refilling ${credit.id} (${resourceType})`, async () => {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.credits)
            .set({
              remaining: credit.refillAmount,
              refilledAt: now.getTime(),
              updatedAt: now.getTime(),
            })
            .where(eq(schema.credits.id, credit.id));

          const auditLogId = newId("auditLog");
          const auditLogTargets = [
            {
              type: "workspace",
              id: credit.workspaceId,
              workspaceId: credit.workspaceId,
              bucket: BUCKET_NAME,
              bucketId: "dummy",
              auditLogId,
              displayName: `workspace ${credit.workspaceId}`,
            },
          ];

          if (credit.keyId) {
            auditLogTargets.push({
              type: "key",
              id: credit.keyId,
              workspaceId: credit.workspaceId,
              bucket: BUCKET_NAME,
              bucketId: "dummy",
              auditLogId,
              displayName: `key ${credit.keyId}`,
            });
          }

          if (credit.identityId) {
            auditLogTargets.push({
              type: "identity",
              id: credit.identityId,
              workspaceId: credit.workspaceId,
              bucket: BUCKET_NAME,
              bucketId: "dummy",
              auditLogId,
              displayName: `identity ${credit.identityId}`,
            });
          }

          await tx.insert(schema.auditLog).values({
            id: auditLogId,
            workspaceId: credit.workspaceId,
            bucket: BUCKET_NAME,
            bucketId: "dummy",
            time: now.getTime(),
            event: resourceType === "key" ? "key.update" : "identity.update",
            actorId: "trigger",
            actorType: "system",
            display: `Refilled ${resourceType} ${resourceId} to ${credit.refillAmount} credits`,
          });
          await tx.insert(schema.auditLogTarget).values(auditLogTargets);
        });
        return { creditId: credit.id, resourceType, resourceId };
      });
    }

    await step.do("heartbeat", async () => {
      await fetch(this.env.HEARTBEAT_URL_REFILLS);
    });

    return {
      refillCreditIds: creditsToRefill.map((c) => c.id),
    };
  }
}
