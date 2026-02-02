import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { tieredPricingSchema } from "./types";

// List pricing models
export const listPricingModels = workspaceProcedure.query(async ({ ctx }) => {
  const models = await db.query.pricingModels.findMany({
    where: eq(schema.pricingModels.workspaceId, ctx.workspace.id),
    orderBy: (models, { desc }) => [desc(models.createdAtM)],
  });

  return models.filter((m) => m.active);
});

// Get pricing model by ID
export const getPricingModel = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .query(async ({ ctx, input }) => {
    const model = await db.query.pricingModels.findFirst({
      where: eq(schema.pricingModels.id, input.id),
    });

    if (!model || model.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Pricing model not found",
      });
    }

    return model;
  });

// Create pricing model
export const createPricingModel = workspaceProcedure
  .input(
    z.object({
      name: z.string().min(1).max(255),
      currency: z.string().length(3),
      verificationUnitPrice: z.number().multipleOf(0.00001).nonnegative(),
      keyAccessUnitPrice: z.number().multipleOf(0.00001).nonnegative(),
      creditUnitPrice: z.number().multipleOf(0.00001).nonnegative(),
      tieredPricing: tieredPricingSchema.nullable().optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Check if workspace has billing beta access
    if (!ctx.workspace.betaFeatures.billing) {
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "Billing feature is not enabled for this workspace",
      });
    }

    // Check for connected account
    const connectedAccount = await db.query.stripeConnectedAccounts.findFirst({
      where: eq(schema.stripeConnectedAccounts.workspaceId, ctx.workspace.id),
    });

    if (!connectedAccount || connectedAccount.disconnectedAt) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Please connect your Stripe account first",
      });
    }

    // Enforce single currency per workspace
    const existingModels = await db.query.pricingModels.findMany({
      where: eq(schema.pricingModels.workspaceId, ctx.workspace.id),
    });

    const activeModels = existingModels.filter((m) => m.active);
    if (activeModels.length > 0) {
      const existingCurrency = activeModels[0].currency;
      if (existingCurrency !== input.currency.toUpperCase()) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `All pricing models must use the same currency. This workspace uses ${existingCurrency}.`,
        });
      }
    }

    const id = newId("pricingModel");
    const now = Date.now();

    await db.insert(schema.pricingModels).values({
      id,
      workspaceId: ctx.workspace.id,
      name: input.name,
      currency: input.currency.toUpperCase(),
      verificationUnitPrice: input.verificationUnitPrice,
      keyAccessUnitPrice: input.keyAccessUnitPrice,
      creditUnitPrice: input.creditUnitPrice,
      tieredPricing: input.tieredPricing ?? null,
      version: 1,
      active: true,
      createdAtM: now,
      updatedAtM: now,
    });

    return { id };
  });

// Update pricing model (creates new version)
export const updatePricingModel = workspaceProcedure
  .input(
    z.object({
      id: z.string(),
      name: z.string().min(1).max(255).optional(),
      verificationUnitPrice: z.number().min(0).optional(),
      keyAccessUnitPrice: z.number().min(0).optional(),
      creditUnitPrice: z.number().min(0).optional(),
      tieredPricing: tieredPricingSchema.nullable().optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const model = await db.query.pricingModels.findFirst({
      where: eq(schema.pricingModels.id, input.id),
    });

    if (!model || model.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Pricing model not found",
      });
    }

    const now = Date.now();

    await db
      .update(schema.pricingModels)
      .set({
        name: input.name ?? model.name,
        verificationUnitPrice: input.verificationUnitPrice ?? model.verificationUnitPrice,
        keyAccessUnitPrice: input.keyAccessUnitPrice ?? model.keyAccessUnitPrice,
        creditUnitPrice: input.creditUnitPrice ?? model.creditUnitPrice,
        tieredPricing:
          input.tieredPricing !== undefined ? input.tieredPricing : model.tieredPricing,
        version: model.version + 1,
        updatedAtM: now,
      })
      .where(eq(schema.pricingModels.id, input.id));

    return { success: true };
  });

// Delete pricing model (soft delete)
export const deletePricingModel = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const model = await db.query.pricingModels.findFirst({
      where: eq(schema.pricingModels.id, input.id),
    });

    if (!model || model.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Pricing model not found",
      });
    }

    // Check if any end users are using this pricing model
    const endUsers = await db.query.billingEndUsers.findMany({
      where: eq(schema.billingEndUsers.pricingModelId, input.id),
    });

    if (endUsers.length > 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Cannot delete pricing model with active end users",
      });
    }

    await db
      .update(schema.pricingModels)
      .set({
        active: false,
        updatedAtM: Date.now(),
      })
      .where(eq(schema.pricingModels.id, input.id));

    return { success: true };
  });

// Get workspace currency (from existing pricing models)
export const getWorkspaceCurrency = workspaceProcedure.query(async ({ ctx }) => {
  const model = await db.query.pricingModels.findFirst({
    where: eq(schema.pricingModels.workspaceId, ctx.workspace.id),
  });

  return model?.currency ?? null;
});
