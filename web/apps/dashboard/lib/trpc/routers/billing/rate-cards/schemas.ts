import { billingTier, calculateTieredPrices } from "@unkey/billing";
import { z } from "zod";

/**
 * Tiers for one metered dimension. calculateTieredPrices with 0 units runs
 * the full tier validation (contiguity, single unbounded tail) without
 * computing anything, so CRUD validation and billing math share one rule set.
 */
const dimensionTiers = z
  .array(billingTier)
  .min(1)
  .superRefine((tiers, ctx) => {
    const result = calculateTieredPrices(tiers, 0);
    if (result.err) {
      ctx.addIssue({
        code: "custom",
        message: result.err.message,
      });
    }
  });

/**
 * Tiered prices per metered dimension; omitted dimensions are not billed.
 * Matches the RateCardConfig type stored in rate_cards.config.
 */
export const rateCardConfigSchema = z
  .object({
    verifications: dimensionTiers.optional(),
    credits: dimensionTiers.optional(),
    ratelimits: dimensionTiers.optional(),
  })
  .refine(
    (config) => config.verifications || config.credits || config.ratelimits,
    "A rate card must price at least one dimension",
  );

export const rateCardNameSchema = z
  .string()
  .transform((s) => s.trim())
  .refine((s) => s.length >= 1, "Name is required")
  .refine((s) => s.length <= 255, "Name cannot exceed 255 characters");

export const currencySchema = z
  .string()
  .regex(/^[A-Z]{3}$/, "Currency must be a 3-letter ISO 4217 code");
