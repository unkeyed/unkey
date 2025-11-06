import { TRPCError } from "@trpc/server";
import Stripe from "stripe";

export const mapProduct = (p: Stripe.Product) => {
  if (!p.default_price) {
    throw new Error(`Product ${p.id} is missing default_price`);
  }

  const price = typeof p.default_price === "string" ? null : (p.default_price as Stripe.Price);

  if (!price) {
    throw new Error(`Product ${p.id} default_price must be expanded`);
  }

  if (price.unit_amount === null || price.unit_amount === undefined) {
    throw new Error(`Product ${p.id} price is missing unit_amount`);
  }

  const quotaRaw = p.metadata?.quota_requests_per_month;

  // Validate that the metadata value is a non-empty string of digits
  if (!quotaRaw || typeof quotaRaw !== "string" || !/^\d+$/.test(quotaRaw)) {
    throw new Error(
      `Product ${p.id} metadata.quota_requests_per_month must be a non-empty string of digits, got: ${quotaRaw}`,
    );
  }

  // Parse into integer only after regex validation
  const quotaValue = Number.parseInt(quotaRaw, 10);

  // Ensure the parsed integer is >= 0
  if (quotaValue < 0) {
    throw new Error(
      `Product ${p.id} metadata.quota_requests_per_month must be >= 0, got: ${quotaRaw}`,
    );
  }

  return {
    id: p.id,
    name: p.name,
    priceId: price.id,
    dollar: price.unit_amount / 100,
    quotas: {
      requestsPerMonth: quotaValue,
    },
  };
};

export const handleStripeError = (error: Stripe.errors.StripeError) => {
  const stripeError = error;
  let code: TRPCError["code"];

  // Map Stripe error types to TRPC error codes
  if (error instanceof Stripe.errors.StripeAuthenticationError) {
    code = "UNAUTHORIZED";
  } else if (error instanceof Stripe.errors.StripeRateLimitError) {
    code = "TOO_MANY_REQUESTS";
  } else if (error instanceof Stripe.errors.StripeInvalidRequestError) {
    code = "BAD_REQUEST";
  } else if (error instanceof Stripe.errors.StripePermissionError) {
    code = "FORBIDDEN";
  } else if (stripeError.statusCode === 404 || stripeError.code === "resource_missing") {
    code = "NOT_FOUND";
  } else if (error instanceof Stripe.errors.StripeAPIError) {
    code = "INTERNAL_SERVER_ERROR";
  } else if (error instanceof Stripe.errors.StripeConnectionError) {
    code = "INTERNAL_SERVER_ERROR";
  } else {
    // Default for other Stripe errors
    code = "INTERNAL_SERVER_ERROR";
  }

  throw new TRPCError({
    code,
    message: `Stripe error: ${stripeError.message}`,
  });
};
