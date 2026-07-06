import { insertAuditLogs } from "@/lib/audit";
import { db, schema, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

export const setDefaultRateCardInputSchema = z.object({
  /** Null clears the workspace default. */
  rateCardId: z.string().nullable(),
});

export const setDefaultRateCard = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(setDefaultRateCardInputSchema)
  .mutation(async ({ input, ctx }) => {
    if (input.rateCardId !== null) {
      const rateCardId: string = input.rateCardId;
      const card = await db.query.rateCards.findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.id, rateCardId),
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.archived, false),
          ),
      });
      if (!card) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Rate card not found." });
      }
    }

    await db.transaction(async (tx) => {
      await tx
        .insert(schema.workspaceBillingSettings)
        .values({
          id: newId("rateCard"),
          workspaceId: ctx.workspace.id,
          defaultRateCardId: input.rateCardId,
          stripeConnectEncrypted: null,
          stripeConnectEncryptionKeyId: null,
          createdAt: Date.now(),
          updatedAt: null,
        })
        .onDuplicateKeyUpdate({
          set: { defaultRateCardId: input.rateCardId, updatedAt: sql`${Date.now()}` },
        });

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Set default rate card to ${input.rateCardId ?? "none"}`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
            meta: { defaultRateCardId: input.rateCardId },
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { defaultRateCardId: input.rateCardId };
  });
