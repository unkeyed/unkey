import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { isDuplicateKeyError } from "@/lib/utils/db-errors";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";
import { currencySchema, rateCardConfigSchema, rateCardNameSchema } from "./schemas";

export const createRateCardInputSchema = z.object({
  name: rateCardNameSchema,
  currency: currencySchema.default("USD"),
  config: rateCardConfigSchema,
  selectable: z.boolean().default(false),
});

export const createRateCard = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(createRateCardInputSchema)
  .mutation(async ({ input, ctx }) => {
    const rateCardId = newId("rateCard");

    try {
      await db.transaction(async (tx) => {
        await tx.insert(schema.rateCards).values({
          id: rateCardId,
          workspaceId: ctx.workspace.id,
          name: input.name,
          currency: input.currency,
          config: input.config,
          selectable: input.selectable,
          archived: false,
          createdAt: Date.now(),
          updatedAt: null,
        });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Created rate card "${input.name}" (${rateCardId})`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: ctx.workspace.name || "Unknown workspace",
              meta: { rateCardId, selectable: input.selectable },
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });

      return { rateCardId, name: input.name };
    } catch (err) {
      if (isDuplicateKeyError(err)) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `A rate card named "${input.name}" already exists in your workspace.`,
        });
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We encountered an issue while creating the rate card. Please try again.",
      });
    }
  });
