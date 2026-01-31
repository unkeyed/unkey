import { z } from "zod";

// Stripe Connected Account
export const connectedAccountSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  stripeAccountId: z.string(),
  scope: z.string(),
  connectedAt: z.number(),
  disconnectedAt: z.number().nullable(),
});

export type ConnectedAccount = z.infer<typeof connectedAccountSchema>;

// Pricing Model
export const tieredPricingSchema = z.object({
  tiers: z.array(
    z.object({
      upTo: z.number(),
      unitPrice: z.number(),
    }),
  ),
});

export const pricingModelSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  name: z.string(),
  currency: z.string(),
  verificationUnitPrice: z.number(),
  ratelimitUnitPrice: z.number(),
  tieredPricing: tieredPricingSchema.nullable(),
  version: z.number(),
  active: z.boolean(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
});

export type PricingModel = z.infer<typeof pricingModelSchema>;

// End User
export const endUserSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  externalId: z.string(),
  pricingModelId: z.string(),
  stripeCustomerId: z.string(),
  stripeSubscriptionId: z.string().nullable(),
  email: z.string().nullable(),
  name: z.string().nullable(),
  metadata: z.record(z.string(), z.string()).nullable(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
});

export type EndUser = z.infer<typeof endUserSchema>;

// Invoice
export const invoiceStatusSchema = z.enum(["draft", "open", "paid", "void", "uncollectible"]);

export const invoiceSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  endUserId: z.string(),
  stripeInvoiceId: z.string(),
  billingPeriodStart: z.number(),
  billingPeriodEnd: z.number(),
  verificationCount: z.number(),
  ratelimitCount: z.number(),
  totalAmount: z.number(),
  currency: z.string(),
  status: invoiceStatusSchema,
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
});

export type Invoice = z.infer<typeof invoiceSchema>;

// Usage
export const usageSchema = z.object({
  verifications: z.number(),
  rateLimits: z.number(),
});

export type Usage = z.infer<typeof usageSchema>;

// Revenue Analytics
export const revenueDataPointSchema = z.object({
  date: z.string(),
  revenue: z.number(),
  invoiceCount: z.number(),
});

export type RevenueDataPoint = z.infer<typeof revenueDataPointSchema>;
