import { creditsSchema } from "@/app/(app)/[workspaceId]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const updateKeyRemaining = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    creditsSchema.extend({
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    await db
      .transaction(async (tx) => {
        const key = await tx.query.keys.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.id, input.keyId),
              isNull(table.deletedAtM),
            ),
        });
        if (!key) {
          throw new TRPCError({
            message:
              "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
            code: "NOT_FOUND",
          });
        }

        // If limits are disabled, set all values to null
        if (input.limit.enabled) {
          // Get appropriate refill values based on the interval
          const { refill } = input.limit.data;
          const refillDay = refill.interval === "monthly" ? refill.refillDay : null;
          const refillAmount = refill.interval !== "none" ? refill.amount : null;

          await tx
            .update(schema.keys)
            .set({
              remaining: input.limit.data.remaining,
              refillDay,
              refillAmount,
              lastRefillAt: refillAmount ? new Date() : null,
            })
            .where(and(eq(schema.keys.id, key.id), eq(schema.keys.workspaceId, ctx.workspace.id)))
            .catch((_err) => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message:
                  "We were unable to update remaining on this key. Please try again or contact support@unkey.dev",
              });
            });
        } else {
          await tx
            .update(schema.keys)
            .set({
              remaining: null,
              refillDay: null,
              refillAmount: null,
              lastRefillAt: null,
            })
            .where(and(eq(schema.keys.id, key.id), eq(schema.keys.workspaceId, ctx.workspace.id)))
            .catch((_err) => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message:
                  "We were unable to update remaining on this key. Please try again or contact support@unkey.dev",
              });
            });
        }

        // Create audit log
        let description = "";
        if (input.limit.enabled) {
          const { refill } = input.limit.data;
          const refillInfo =
            refill.interval !== "none"
              ? `${refill.amount}@${refill.interval === "monthly" ? refill.refillDay : "daily"}`
              : "none";

          description = `Changed remaining for ${key.id} to remaining=${input.limit.data.remaining}, refill=${refillInfo}`;
        } else {
          description = `Disabled limit for ${key.id}`;
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description,
          resources: [
            {
              type: "key",
              id: key.id,
              name: key.name || undefined,
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
    return { keyId: input.keyId };
  });
