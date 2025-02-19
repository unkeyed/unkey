import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
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
          amount: z.number().int().min(1).optional(),
          refillDay: z.number().int().min(1).max(31).nullable().optional(),
        })
        .optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    if (input.limitEnabled === false || input.remaining === null) {
      input.remaining = undefined;
      input.refill = undefined;
    }

    await db
      .transaction(async (tx) => {
        const key = await tx.query.keys.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.id, input.keyId),
              isNull(table.deletedAt),
            ),
        });
        if (!key) {
          throw new TRPCError({
            message:
              "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
            code: "NOT_FOUND",
          });
        }
        await tx
          .update(schema.keys)
          .set({
            remaining: input.remaining ?? null,
            refillDay: input.refill?.refillDay ?? null,
            refillAmount: input.refill?.amount ?? null,
            lastRefillAt: input.refill?.amount ? new Date() : null,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update remaining on this key. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: input.limitEnabled
            ? `Changed remaining for ${key.id} to remaining=${input.remaining}, refill=${
                input.refill ? `${input.refill.amount}@${input.refill.refillDay}` : "none"
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update remaining limits on this key. Please try again or contact support@unkey.dev",
        });
      });
  });
