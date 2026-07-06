import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

export const listRateCards = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const [cards, settings] = await Promise.all([
      db.query.rateCards.findMany({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.archived, false)),
        orderBy: (table, { asc }) => [asc(table.name)],
      }),
      db.query.workspaceBillingSettings.findFirst({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
      }),
    ]);

    return {
      rateCards: cards.map((card) => ({
        id: card.id,
        name: card.name,
        currency: card.currency,
        config: card.config,
        selectable: card.selectable,
        isDefault: settings?.defaultRateCardId === card.id,
      })),
      defaultRateCardId: settings?.defaultRateCardId ?? null,
    };
  });
