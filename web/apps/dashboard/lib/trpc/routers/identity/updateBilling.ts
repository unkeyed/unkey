import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const updateIdentityBillingInputSchema = z.object({
  identityId: z.string(),
  /** Binds the identity to a provider; only stripe_connect is billed today. */
  billingProvider: z.enum(["none", "stripe_connect", "export"]),
  /** Provider-side customer reference (e.g. Stripe customer on the connected account). */
  billingExternalCustomerId: z.string().trim().max(256).nullable(),
  /** Assigned rate card; null falls back to the workspace default. */
  rateCardId: z.string().nullable(),
});

/**
 * Sets an end-user identity's billing: which provider it bills through, the
 * provider customer it maps to, and the rate card assigned to it. Admin-gated
 * like the other billing mutations; validates the identity and (when set) the
 * rate card belong to the caller's workspace.
 */
export const updateIdentityBilling = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(updateIdentityBillingInputSchema)
  .mutation(async ({ input, ctx }) => {
    const identity = await db.query.identities.findFirst({
      where: (table, { and: andW, eq: eqW }) =>
        andW(eqW(table.id, input.identityId), eqW(table.workspaceId, ctx.workspace.id)),
    });
    if (!identity) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Identity not found." });
    }

    if (input.rateCardId !== null) {
      const rateCardId = input.rateCardId;
      const card = await db.query.rateCards.findFirst({
        where: (table, { and: andW, eq: eqW }) =>
          andW(
            eqW(table.id, rateCardId),
            eqW(table.workspaceId, ctx.workspace.id),
            eqW(table.archived, false),
          ),
      });
      if (!card) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Rate card not found in this workspace.",
        });
      }
    }

    const customerId =
      input.billingExternalCustomerId && input.billingExternalCustomerId.trim() !== ""
        ? input.billingExternalCustomerId.trim()
        : null;

    await db.transaction(async (tx) => {
      await tx
        .update(schema.identities)
        .set({
          billingProvider: input.billingProvider,
          billingExternalCustomerId: customerId,
          rateCardId: input.rateCardId,
        })
        .where(
          and(
            eq(schema.identities.id, input.identityId),
            eq(schema.identities.workspaceId, ctx.workspace.id),
          ),
        );

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "identity.update",
        description: `Updated billing for identity ${input.identityId} (provider=${input.billingProvider}, rateCard=${input.rateCardId ?? "workspace default"})`,
        resources: [
          {
            type: "identity",
            id: input.identityId,
            name: identity.externalId,
            meta: {
              billingProvider: input.billingProvider,
              rateCardId: input.rateCardId,
              hasProviderCustomer: customerId !== null,
            },
          },
        ],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });

    return { success: true };
  });
