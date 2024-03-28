import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyRatelimit = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      workspaceId: z.string(),
      enabled: z.boolean(),
      ratelimitType: z.enum(["fast", "consistent"]).nullable(),
      ratelimitLimit: z.number().int().positive().optional(),
      ratelimitRefillRate: z.number().int().positive().optional(),
      ratelimitRefillInterval: z.number().int().positive().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let ratelimitType: "fast" | "consistent" | null = null;
    let ratelimitLimit: number | null = null;
    let ratelimitRefillRate: number | null = null;
    let ratelimitRefillInterval: number | null = null;

    if (input.enabled) {
      if (typeof input.ratelimitType !== "string") {
        throw new TRPCError({
          message: "ratelimitType must be a string",
          code: "BAD_REQUEST",
        });
      }
      ratelimitType = input.ratelimitType;

      if (typeof input.ratelimitLimit !== "number" || input.ratelimitLimit <= 0) {
        throw new TRPCError({
          message: "Limit must be a positive integer",
          code: "BAD_REQUEST",
        });
      }
      ratelimitLimit = input.ratelimitLimit;

      if (typeof input.ratelimitRefillRate !== "number" || input.ratelimitRefillRate <= 0) {
        throw new TRPCError({
          message: "Rate must be a positive integer",
          code: "BAD_REQUEST",
        });
      }
      ratelimitRefillRate = input.ratelimitRefillRate;
      if (typeof input.ratelimitRefillInterval !== "number" || input.ratelimitRefillInterval <= 0) {
        throw new TRPCError({
          message: "Interval must be a positive integer",
          code: "BAD_REQUEST",
        });
      }
      ratelimitRefillInterval = input.ratelimitRefillInterval;
    }
    const key = await db.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
    }
    await db
      .update(schema.keys)
      .set({
        ratelimitType,
        ratelimitLimit,
        ratelimitRefillRate,
        ratelimitRefillInterval,
      })
      .where(eq(schema.keys.id, key.id));
    await ingestAuditLogs({
      workspaceId: key.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "key.update",
      description: `Changed ratelimit of ${key.id}`,
      resources: [
        {
          type: "key",
          id: key.id,
          meta: {
            "ratelimit.type": ratelimitType,
            "ratelimit.limit": ratelimitLimit,
            "ratelimit.refillRate": ratelimitRefillRate,
            "ratelimit.refillInterval": ratelimitRefillInterval,
          },
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
