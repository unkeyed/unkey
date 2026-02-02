import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { getStripeClient } from "@/lib/stripe";

// Search identities by external ID (like create key does)
export const searchIdentitiesByExternalId = workspaceProcedure
  .input(
    z.object({
      query: z.string().trim().min(1).max(255),
    }),
  )
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    // Search for identities by external ID
    const identities = await db.query.identities.findMany({
      where: (table, { and, eq, like }) =>
        and(
          eq(table.workspaceId, workspaceId),
          eq(table.deleted, false),
          like(table.externalId, `%${query}%`),
        ),
      limit: 10,
      orderBy: (identities, { asc }) => [asc(identities.externalId)],
    });

    return { identities };
  });

// Search end users by external ID (for existing end users)
export const searchEndUsers = workspaceProcedure
  .input(
    z.object({
      query: z.string().trim().min(1).max(255),
    }),
  )
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    // Search for end users by external ID
    const endUsers = await db.query.billingEndUsers.findMany({
      where: (table, { and, eq, like }) =>
        and(
          eq(table.workspaceId, workspaceId),
          like(table.externalId, `%${query}%`),
        ),
      with: {
        pricingModel: {
          columns: {
            id: true,
            name: true,
            currency: true,
          },
        },
      },
      limit: 10,
      orderBy: (users, { asc }) => [asc(users.externalId)],
    });

    return { endUsers };
  });

// Create end user - uses search + create pattern like identity creation
export const createEndUser = workspaceProcedure
  .input(
    z.object({
      externalId: z.string().min(1).max(255),
      pricingModelId: z.string(),
      email: z.string().email().optional(),
      name: z.string().max(255).optional(),
      metadata: z.record(z.string(), z.string()).optional(),
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

    // Verify pricing model exists and belongs to workspace
    const pricingModel = await db.query.pricingModels.findFirst({
      where: eq(schema.pricingModels.id, input.pricingModelId),
    });

    if (!pricingModel || pricingModel.workspaceId !== ctx.workspace.id || !pricingModel.active) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Pricing model not found",
      });
    }

    // Check if end user with this externalId already exists (search pattern)
    const existingEndUser = await db.query.billingEndUsers.findFirst({
      where: eq(schema.billingEndUsers.externalId, input.externalId),
    });

    if (existingEndUser && existingEndUser.workspaceId === ctx.workspace.id) {
      throw new TRPCError({
        code: "CONFLICT",
        message: "An end user with this external ID already exists in your workspace.",
      });
    }

    // Check if identity with this externalId already exists
    let identityId: string;
    const existingIdentity = await db.query.identities.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.workspaceId, ctx.workspace.id), eq(table.externalId, input.externalId)),
    });

    if (existingIdentity) {
      // Use existing identity
      identityId = existingIdentity.id;
    } else {
      // Create new identity (like create key does)
      identityId = newId("identity");
      await db.insert(schema.identities).values({
        id: identityId,
        externalId: input.externalId,
        workspaceId: ctx.workspace.id,
        createdAt: Date.now(),
        updatedAt: null,
        meta: null,
        environment: "",
        deleted: false,
      });
    }

    const now = Date.now();
    const endUserId = newId("billingEndUser");

    // Get connected account for workspace to create customer on connected account
    const billingConnectedAccount = await db.query.billingConnectedAccounts.findFirst({
      where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
    });
    if (!billingConnectedAccount || !billingConnectedAccount.stripeAccountId) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Stripe account not connected",
      });
    }

    // Create real Stripe customer on connected account
    const stripe = getStripeClient();
    const stripeCustomer = await stripe.customers.create(
      {
        email: input.email,
        name: input.name,
        metadata: {
          workspace_id: ctx.workspace.id,
          external_id: input.externalId,
        },
      },
      {
        stripeAccount: billingConnectedAccount.stripeAccountId,
      }
    );

    await db.insert(schema.billingEndUsers).values({
      id: endUserId,
      workspaceId: ctx.workspace.id,
      externalId: input.externalId,
      pricingModelId: input.pricingModelId,
      stripeCustomerId: stripeCustomer.id,
      email: input.email ?? null,
      name: input.name ?? null,
      metadata: input.metadata ?? null,
      createdAtM: now,
      updatedAtM: now,
    });

    return { id: endUserId, stripeCustomerId: stripeCustomer.id, identityId };
  });

// List end users
export const listEndUsers = workspaceProcedure
  .input(
    z
      .object({
        limit: z.number().min(1).max(100).default(50),
        offset: z.number().min(0).default(0),
      })
      .optional(),
  )
  .query(async ({ ctx, input }) => {
    const limit = input?.limit ?? 50;
    const offset = input?.offset ?? 0;

    const endUsers = await db.query.billingEndUsers.findMany({
      where: eq(schema.billingEndUsers.workspaceId, ctx.workspace.id),
      with: {
        pricingModel: true,
      },
      orderBy: (users, { desc }) => [desc(users.createdAtM)],
      limit,
      offset,
    });

    return endUsers;
  });

// Get end user by ID
export const getEndUser = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .query(async ({ ctx, input }) => {
    const endUser = await db.query.billingEndUsers.findFirst({
      where: eq(schema.billingEndUsers.id, input.id),
      with: {
        pricingModel: true,
        invoices: {
          orderBy: (invoices, { desc }) => [desc(invoices.createdAtM)],
          limit: 10,
        },
      },
    });

    if (!endUser || endUser.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "End user not found",
      });
    }

    return endUser;
  });

// Upsert end user by external ID
export const upsertEndUser = workspaceProcedure
  .input(
    z.object({
      externalId: z.string().min(1).max(255),
      pricingModelId: z.string(),
      email: z.string().email().optional(),
      name: z.string().max(255).optional(),
      metadata: z.record(z.string(), z.string()).optional(),
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

    // Verify pricing model exists and belongs to workspace
    const pricingModel = await db.query.pricingModels.findFirst({
      where: eq(schema.pricingModels.id, input.pricingModelId),
    });

    if (!pricingModel || pricingModel.workspaceId !== ctx.workspace.id || !pricingModel.active) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Pricing model not found",
      });
    }

    const now = Date.now();

    // Check for existing end user with same external ID (upsert logic)
    const existing = await db.query.billingEndUsers.findFirst({
      where: eq(schema.billingEndUsers.externalId, input.externalId),
    });

    if (existing && existing.workspaceId === ctx.workspace.id) {
      // Update existing end user
      await db
        .update(schema.billingEndUsers)
        .set({
          pricingModelId: input.pricingModelId,
          email: input.email ?? existing.email,
          name: input.name ?? existing.name,
          metadata: input.metadata ?? existing.metadata,
          updatedAtM: now,
        })
        .where(eq(schema.billingEndUsers.id, existing.id));

      return { id: existing.id, stripeCustomerId: existing.stripeCustomerId, created: false };
    }

    const id = newId("billingEndUser");

    // Create real Stripe customer on connected account
    const stripe = getStripeClient();
    const stripeCustomer = await stripe.customers.create(
      {
        email: input.email,
        name: input.name,
        metadata: {
          workspace_id: ctx.workspace.id,
          external_id: input.externalId,
        },
      },
      {
        stripeAccount: connectedAccount.stripeAccountId,
      }
    );

    await db.insert(schema.billingEndUsers).values({
      id,
      workspaceId: ctx.workspace.id,
      externalId: input.externalId,
      pricingModelId: input.pricingModelId,
      stripeCustomerId: stripeCustomer.id,
      email: input.email ?? null,
      name: input.name ?? null,
      metadata: input.metadata ?? null,
      createdAtM: now,
      updatedAtM: now,
    });

    return { id, stripeCustomerId: stripeCustomer.id, created: true };
  });

// Update end user
export const updateEndUser = workspaceProcedure
  .input(
    z.object({
      id: z.string(),
      pricingModelId: z.string().optional(),
      email: z.string().email().optional(),
      name: z.string().max(255).optional(),
      metadata: z.record(z.string(), z.string()).optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const endUser = await db.query.billingEndUsers.findFirst({
      where: eq(schema.billingEndUsers.id, input.id),
    });

    if (!endUser || endUser.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "End user not found",
      });
    }

    // If changing pricing model, verify it exists
    if (input.pricingModelId) {
      const pricingModel = await db.query.pricingModels.findFirst({
        where: eq(schema.pricingModels.id, input.pricingModelId),
      });

      if (!pricingModel || pricingModel.workspaceId !== ctx.workspace.id || !pricingModel.active) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Pricing model not found",
        });
      }
    }

    await db
      .update(schema.billingEndUsers)
      .set({
        pricingModelId: input.pricingModelId ?? endUser.pricingModelId,
        email: input.email !== undefined ? input.email : endUser.email,
        name: input.name !== undefined ? input.name : endUser.name,
        metadata: input.metadata !== undefined ? input.metadata : endUser.metadata,
        updatedAtM: Date.now(),
      })
      .where(eq(schema.billingEndUsers.id, input.id));

    return { success: true };
  });

// Delete end user
export const deleteEndUser = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const endUser = await db.query.billingEndUsers.findFirst({
      where: eq(schema.billingEndUsers.id, input.id),
    });

    if (!endUser || endUser.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "End user not found",
      });
    }

    // Check for existing invoices
    const invoices = await db.query.billingInvoices.findMany({
      where: eq(schema.billingInvoices.endUserId, input.id),
    });

    if (invoices.length > 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Cannot delete end user with existing invoices",
      });
    }

    await db.delete(schema.billingEndUsers).where(eq(schema.billingEndUsers.id, input.id));

    return { success: true };
  });
