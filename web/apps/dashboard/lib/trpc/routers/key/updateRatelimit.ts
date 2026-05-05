import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { type Key, and, db, eq, schema } from "@/lib/db";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

const baseRatelimitInputSchema = z.object({
  keyId: z.string(),
});

export const ratelimitInputSchema = ratelimitSchema.and(baseRatelimitInputSchema);

type RatelimitInputSchema = z.infer<typeof ratelimitInputSchema>;

export const updateKeyRatelimit = workspaceProcedure
  .input(ratelimitInputSchema)
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.keyId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update ratelimits on this key. Please try again or contact support@unkey.com",
        });
      });
    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.com.",
        code: "NOT_FOUND",
      });
    }

    return updateRatelimitV2(input, key, {
      audit: ctx.audit,
      userId: ctx.user.id,
      workspaceId: ctx.workspace.id,
    });
  });

const updateRatelimitV2 = async (
  input: RatelimitInputSchema,
  key: Key,
  ctx: {
    workspaceId: string;
    userId: string;
    audit: {
      location: string;
      userAgent: string | undefined;
    };
  },
) => {
  try {
    await db.transaction(async (tx) => {
      if (input.ratelimit.enabled && input.ratelimit.data.length > 0) {
        // Scope the existing-row lookup to the verified key AND workspace,
        // so any subsequent UPDATE/DELETE that re-uses these ids is implicitly
        // limited to rows the caller owns.
        const existingRatelimits = await tx
          .select()
          .from(schema.ratelimits)
          .where(
            and(
              eq(schema.ratelimits.keyId, input.keyId),
              eq(schema.ratelimits.workspaceId, ctx.workspaceId),
            ),
          );
        const existingRatelimitIds = new Set(existingRatelimits.map((r) => r.id));

        const inputRatelimitIds = new Set(
          input.ratelimit.data.filter((r) => r.id).map((r) => r.id),
        );

        // Delete ratelimits that exist in DB but are not in the input (they were removed).
        // Filter by workspace + key to defend against any future code path that
        // might pass an unverified id here.
        for (const existing of existingRatelimits) {
          if (!inputRatelimitIds.has(existing.id)) {
            await tx
              .delete(schema.ratelimits)
              .where(
                and(
                  eq(schema.ratelimits.id, existing.id),
                  eq(schema.ratelimits.keyId, input.keyId),
                  eq(schema.ratelimits.workspaceId, ctx.workspaceId),
                ),
              );
          }
        }

        // Update or insert each ratelimit sequentially
        for (const ratelimit of input.ratelimit.data) {
          if (ratelimit.id) {
            // Reject any client-supplied id that doesn't belong to this key.
            // Without this, an attacker could pass a foreign workspace's
            // ratelimit id and rewrite its limit/duration/autoApply fields.
            if (!existingRatelimitIds.has(ratelimit.id)) {
              throw new TRPCError({
                code: "NOT_FOUND",
                message: "Ratelimit not found",
              });
            }
            await tx
              .update(schema.ratelimits)
              .set({
                duration: ratelimit.refillInterval,
                limit: ratelimit.limit,
                name: ratelimit.name,
                updatedAt: Date.now(),
                autoApply: ratelimit.autoApply,
              })
              .where(
                and(
                  eq(schema.ratelimits.id, ratelimit.id),
                  eq(schema.ratelimits.keyId, input.keyId),
                  eq(schema.ratelimits.workspaceId, ctx.workspaceId),
                ),
              );
          } else {
            // Create new
            await tx.insert(schema.ratelimits).values({
              id: newId("ratelimit"),
              keyId: input.keyId,
              duration: ratelimit.refillInterval,
              limit: ratelimit.limit,
              name: ratelimit.name,
              workspaceId: ctx.workspaceId,
              createdAt: Date.now(),
              updatedAt: null,
            });
          }
        }
      } else if (input.ratelimit.enabled) {
        // Rate limiting is enabled but no rules provided (edge case). Should not happen in v2.
        throw new Error("Rate limiting is enabled but no rules were provided");
      } else {
        // If rate limiting is disabled, remove all rate limit rules for this key
        await tx
          .delete(schema.ratelimits)
          .where(
            and(
              eq(schema.ratelimits.keyId, input.keyId),
              eq(schema.ratelimits.workspaceId, ctx.workspaceId),
            ),
          );
      }
      const description = input.ratelimit.enabled
        ? `Updated rate limits for key ${key.id} (${input.ratelimit.data.length} rules)`
        : `Disabled rate limits for key ${key.id}`;

      const ratelimitMeta = input.ratelimit.enabled
        ? {
            "ratelimit.enabled": true,
            "ratelimit.rules_count": input.ratelimit.data.length,
            ...input.ratelimit.data.reduce((acc, rule, index) => {
              return {
                // biome-ignore lint/performance/noAccumulatingSpread: <explanation>
                ...acc,
                [`ratelimit.rule.${index}.name`]: rule.name,
                [`ratelimit.rule.${index}.limit`]: rule.limit,
                [`ratelimit.rule.${index}.interval`]: rule.refillInterval,
              };
            }, {}),
          }
        : {
            "ratelimit.enabled": false,
          };

      const resources: UnkeyAuditLog["resources"] = [
        {
          type: "ratelimit",
          id: key.id,
          name: key.name || undefined,
          meta: ratelimitMeta,
        },
      ];

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspaceId,
        actor: {
          type: "user",
          id: ctx.userId,
        },
        event: "ratelimit.update",
        description,
        resources,
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });
  } catch (err) {
    console.error(err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "We were unable to update ratelimit on this key. Please try again or contact support@unkey.com",
    });
  }

  return { keyId: key.id };
};
