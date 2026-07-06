import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

/**
 * Current end-user billing configuration for one identity: its billing provider
 * binding, the provider-side customer reference, the workspace-assigned rate
 * card (null = fall back to the workspace default), and any card the end-user
 * self-selected (read-only here; takes precedence over the assignment).
 */
export const getIdentityBilling = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ identityId: z.string() }))
  .query(async ({ ctx, input }) => {
    const identity = await db.query.identities.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.identityId), eq(table.workspaceId, ctx.workspace.id)),
      columns: {
        billingProvider: true,
        billingExternalCustomerId: true,
        rateCardId: true,
        selectedRateCardId: true,
      },
    });
    if (!identity) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Identity not found" });
    }
    return {
      billingProvider: identity.billingProvider,
      billingExternalCustomerId: identity.billingExternalCustomerId,
      rateCardId: identity.rateCardId,
      selectedRateCardId: identity.selectedRateCardId,
    };
  });
