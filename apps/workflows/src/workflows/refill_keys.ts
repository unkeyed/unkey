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
    const creditsToRefill = await step.do("fetch credits from credits table", async () => {
      const { and, or, eq, isNull, isNotNull, gt, sql } = await import("@unkey/db");

      // Build the refill conditions
      const baseConditions = [
        isNotNull(schema.credits.refillAmount),
        isNotNull(schema.credits.remaining),
        gt(schema.credits.refillAmount, schema.credits.remaining),
        or(isNull(schema.credits.refillDay), eq(schema.credits.refillDay, today)),
      ];

      if (today === lastDayOfMonth) {
        baseConditions.push(gt(schema.credits.refillDay, today));
      }

      const refillConditions = and(...baseConditions);

      // Query with joins to filter out deleted keys/identities at DB level
      const query = db
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

      const results = await query;

      return results.map((c) => ({
        id: c.id,
        workspaceId: c.workspaceId,
        keyId: c.keyId,
        identityId: c.identityId,
        refillAmount: c.refillAmount as number,
        isLegacy: false,
      }));
    });

    // Also fetch legacy keys that still use the old remaining fields
    const legacyKeysToRefill = await step.do("fetch legacy keys with refill", async () => {
      const { and, or, eq, isNull, isNotNull, gt } = await import("@unkey/db");

      // Build the refill conditions for legacy keys
      const baseConditions = [
        isNotNull(schema.keys.refillAmount),
        isNotNull(schema.keys.remaining),
        gt(schema.keys.refillAmount, schema.keys.remaining),
        or(isNull(schema.keys.refillDay), eq(schema.keys.refillDay, today)),
      ];

      if (today === lastDayOfMonth) {
        baseConditions.push(gt(schema.keys.refillDay, today));
      }

      const refillConditions = and(...baseConditions);

      // Only get keys that DON'T have a credits entry (to avoid double refill)
      const results = await db
        .select({
          id: schema.keys.id,
          workspaceId: schema.keys.workspaceId,
          refillAmount: schema.keys.refillAmount,
        })
        .from(schema.keys)
        .leftJoin(schema.credits, eq(schema.keys.id, schema.credits.keyId))
        .where(
          and(
            refillConditions,
            isNull(schema.keys.deletedAtM),
            isNull(schema.credits.id), // Key doesn't have a credits entry
          ),
        );

      return results.map((k) => ({
        id: k.id,
        workspaceId: k.workspaceId,
        keyId: k.id,
        identityId: null,
        refillAmount: k.refillAmount as number,
        isLegacy: true,
      }));
    });

    const allToRefill = [...creditsToRefill, ...legacyKeysToRefill];

    for (const credit of allToRefill) {
      const resourceType = credit.keyId ? "key" : "identity";
      const resourceId = credit.keyId || credit.identityId;
      await step.do(`refilling ${credit.id} (${resourceType})`, async () => {
        await db.transaction(async (tx) => {
          if (credit.isLegacy) {
            // Update legacy key remaining field
            await tx
              .update(schema.keys)
              .set({
                remaining: credit.refillAmount,
                lastRefillAt: now,
              })
              .where(eq(schema.keys.id, credit.id));
          } else {
            // Update credits table
            await tx
              .update(schema.credits)
              .set({
                remaining: credit.refillAmount,
                refilledAt: now.getTime(),
                updatedAt: now.getTime(),
              })
              .where(eq(schema.credits.id, credit.id));
          }

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
      refillLegacyKeyIds: legacyKeysToRefill.map((k) => k.id),
    };
  }
}
