import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyRemaining = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      limitEnabled: z.boolean(),
      remaining: z.number().int().positive().optional(),
      refill: z
        .object({
          interval: z.enum(["daily", "monthly", "none"]),
          amount: z.number().int().min(1).optional(),
        })
        .optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    if (input.limitEnabled === false || input.remaining === null) {
      input.remaining = undefined;
      input.refill = undefined;
    }
    if (input.refill?.interval === "none") {
      input.refill = undefined;
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
        remaining: input.remaining ?? null,
        refillInterval:
          input.refill?.interval === "none" || input.refill?.interval === undefined
            ? null
            : input.refill?.interval,
        refillAmount: input.refill?.amount ?? null,
        lastRefillAt: input.refill?.interval ? new Date() : null,
      })
      .where(eq(schema.keys.id, key.id));

    await ingestAuditLogs({
      workspaceId: key.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "key.update",
      description: input.limitEnabled
        ? `Changed remaining for ${key.id} to remaining=${input.remaining}, refill=${
            input.refill ? `${input.refill.amount}@${input.refill.interval}` : "none"
          }`
        : `Disabled limit for ${key.id}`,
      resources: [
        {
          type: "key",
          id: key.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
    return true;
  });
