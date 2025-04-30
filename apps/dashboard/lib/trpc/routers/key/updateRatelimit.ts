import { ratelimitSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { type Key, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

const baseRatelimitInputSchema = z.object({
  keyId: z.string(),
  workspaceId: z.string(),
});

const ratelimitValidationV2 = ratelimitSchema.extend({
  ratelimitType: z.literal("v2"),
});

const ratelimitValidationV1 = z.object({
  enabled: z.boolean(),
  ratelimitAsync: z.boolean().optional(),
  ratelimitLimit: z.number().int().positive().optional(),
  ratelimitDuration: z.number().int().positive().optional(),
  ratelimitType: z.literal("v1"),
});

export const ratelimitInputSchema = z
  .discriminatedUnion("ratelimitType", [ratelimitValidationV1, ratelimitValidationV2])
  .and(baseRatelimitInputSchema);

type RatelimitInputSchema = z.infer<typeof ratelimitInputSchema>;

export const updateKeyRatelimit = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
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
            "We were unable to update ratelimits on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    if (input.ratelimitType === "v1") {
      return updateRatelimitV1(input, key, {
        audit: ctx.audit,
        userId: ctx.user.id,
        workspaceId: ctx.workspace.id,
      });
    }
    return updateRatelimitV2(input, key, {
      audit: ctx.audit,
      userId: ctx.user.id,
      workspaceId: ctx.workspace.id,
    });
  });

const updateRatelimitV1 = async (
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
  if (input.ratelimitType !== "v1") {
    throw new TRPCError({
      message: "Unsupported rate limit type. Only v1 ratelimits are supported.",
      code: "BAD_REQUEST",
    });
  }
  try {
    await db.transaction(async (tx) => {
      const ratelimitValues = input.enabled
        ? {
            ratelimitAsync: input.ratelimitAsync,
            ratelimitLimit: input.ratelimitLimit,
            ratelimitDuration: input.ratelimitDuration,
          }
        : {
            ratelimitAsync: null,
            ratelimitLimit: null,
            ratelimitDuration: null,
          };
      await tx.update(schema.keys).set(ratelimitValues).where(eq(schema.keys.id, key.id));

      const description = input.enabled
        ? `Changed ratelimit of ${key.id}`
        : `Disabled ratelimit of ${key.id}`;

      const resources: UnkeyAuditLog["resources"] = [
        {
          type: "key",
          id: key.id,
          name: key.name || undefined,
          meta: input.enabled
            ? {
                "ratelimit.async": input.ratelimitAsync,
                "ratelimit.limit": input.ratelimitLimit,
                "ratelimit.duration": input.ratelimitDuration,
              }
            : undefined,
        },
      ];
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspaceId,
        actor: {
          type: "user",
          id: ctx.userId,
        },
        event: "key.update",
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
        "We were unable to update ratelimit on this key. Please try again or contact support@unkey.dev",
    });
  }

  return { keyId: key.id };
};

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
  if (input.ratelimitType !== "v2") {
    throw new TRPCError({
      message: "Unsupported rate limit type. Only v2 ratelimits are supported.",
      code: "BAD_REQUEST",
    });
  }
  try {
    await db.transaction(async (tx) => {
      if (input.ratelimit.enabled && input.ratelimit.data.length > 0) {
        // First, fetch existing ratelimits for this key
        const existingRatelimits = await tx
          .select()
          .from(schema.ratelimits)
          .where(eq(schema.ratelimits.keyId, input.keyId));

        const inputRatelimitIds = new Set(
          input.ratelimit.data.filter((r) => r.id).map((r) => r.id),
        );

        // Delete ratelimits that exist in DB but are not in the input (they were removed)
        for (const existing of existingRatelimits) {
          if (!inputRatelimitIds.has(existing.id)) {
            await tx.delete(schema.ratelimits).where(eq(schema.ratelimits.id, existing.id));
          }
        }

        // Update or insert each ratelimit sequentially
        for (const ratelimit of input.ratelimit.data) {
          if (ratelimit.id) {
            // Update existing
            await tx
              .update(schema.ratelimits)
              .set({
                duration: ratelimit.refillInterval,
                limit: ratelimit.limit,
                name: ratelimit.name,
                updatedAt: Date.now(),
              })
              .where(eq(schema.ratelimits.id, ratelimit.id));
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
      } else if (!input.ratelimit.enabled) {
        // If rate limiting is disabled, remove all rate limit rules for this key
        await tx.delete(schema.ratelimits).where(eq(schema.ratelimits.keyId, input.keyId));
      } else {
        // Rate limiting is enabled but no rules provided (edge case). Should not happen in v2.
        throw new Error("Rate limiting is enabled but no rules were provided");
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
        "We were unable to update ratelimit on this key. Please try again or contact support@unkey.dev",
    });
  }

  return { keyId: key.id };
};
