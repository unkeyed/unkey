import { creditsSchema } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
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
          with: {
            credits: true,
          },
        });
        if (!key) {
          throw new TRPCError({
            message:
              "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
            code: "NOT_FOUND",
          });
        }

        const hasOldCredits = key.remaining !== null;
        const hasNewCredits = key.credits !== null;

        // If limits are disabled, set all values to null
        if (input.limit.enabled) {
          // Get appropriate refill values based on the interval
          const { refill } = input.limit.data;
          const refillDay = refill.interval === "monthly" ? refill.refillDay : null;
          const refillAmount = refill.interval !== "none" ? refill.amount : null;

          if (hasOldCredits) {
            await tx
              .update(schema.keys)
              .set({
                remaining: input.limit.data.remaining,
                refillDay,
                refillAmount,
                lastRefillAt: refillAmount ? new Date() : null,
              })
              .where(
                and(eq(schema.keys.id, key.id), eq(schema.keys.workspaceId, ctx.workspace.id)),
              );
          } else if (hasNewCredits) {
            await tx
              .update(schema.credits)
              .set({
                remaining: input.limit.data.remaining,
                refillDay,
                refillAmount,
                refilledAt: Date.now(),
                updatedAt: Date.now(),
              })
              .where(eq(schema.credits.id, key.credits.id));
          } else {
            await tx.insert(schema.credits).values({
              id: newId("credit"),
              workspaceId: ctx.workspace.id,
              keyId: key.id,
              identityId: null,
              remaining: input.limit.data.remaining,
              refillDay,
              refillAmount,
              refilledAt: Date.now(),
              updatedAt: Date.now(),
              createdAt: Date.now(),
            });
          }
        } else {
          if (hasOldCredits) {
            await tx
              .update(schema.keys)
              .set({
                remaining: null,
                refillDay: null,
                refillAmount: null,
                lastRefillAt: null,
              })
              .where(
                and(eq(schema.keys.id, key.id), eq(schema.keys.workspaceId, ctx.workspace.id)),
              );
          } else {
            await tx.delete(schema.credits).where(eq(schema.credits.id, key.credits.id));
          }
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
