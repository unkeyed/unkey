import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";
import { rateCardNameSchema } from "./schemas";

export const updateRateCardInputSchema = z.object({
  rateCardId: z.string(),
  name: rateCardNameSchema.optional(),
  selectable: z.boolean().optional(),
  archived: z.boolean().optional(),
});

/**
 * Renames, toggles end-user selectability, or archives a rate card. Pricing
 * config is immutable after creation — periods already billed reference the
 * card (R18), so a price change is a new card, not an edit.
 */
export const updateRateCard = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(updateRateCardInputSchema)
  .mutation(async ({ input, ctx }) => {
    const card = await db.query.rateCards.findFirst({
      where: (table, { and: andWhere, eq: eqWhere }) =>
        andWhere(eqWhere(table.id, input.rateCardId), eqWhere(table.workspaceId, ctx.workspace.id)),
    });
    if (!card) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Rate card not found." });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.rateCards)
        .set({
          ...(input.name !== undefined ? { name: input.name } : {}),
          ...(input.selectable !== undefined ? { selectable: input.selectable } : {}),
          ...(input.archived !== undefined ? { archived: input.archived } : {}),
        })
        .where(
          and(
            eq(schema.rateCards.id, input.rateCardId),
            eq(schema.rateCards.workspaceId, ctx.workspace.id),
          ),
        );

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Updated rate card "${card.name}" (${card.id})`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
            meta: {
              rateCardId: card.id,
              name: input.name ?? null,
              selectable: input.selectable ?? null,
              archived: input.archived ?? null,
            },
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { rateCardId: card.id };
  });
