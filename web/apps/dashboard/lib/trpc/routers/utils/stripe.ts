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

/**
 * Extracts the id from a Stripe relation, which is either a bare id string
 * or, when the field was expanded, the full object.
 */
export const expandableId = (value: string | { id: string } | null | undefined): string | null => {
  if (!value) {
    return null;
  }
  return typeof value === "string" ? value : value.id;
};

/**
 * Retrieves a Stripe checkout session and verifies it was created for this
 * workspace (billing sessions carry the workspace id in
 * `client_reference_id`). Session ids are attacker-supplied, so this check is
 * what stands between a client-supplied id and another workspace's billing
 * data. A nonexistent session and a foreign session throw the same NOT_FOUND,
 * so the response cannot be used to probe which ids exist.
 */
export const retrieveWorkspaceCheckoutSession = async (args: {
  stripe: Stripe;
  sessionId: string;
  workspaceId: string;
  /** Shown to the caller on rejection. Must not confirm the session exists. */
  notFoundMessage: string;
}): Promise<Stripe.Checkout.Session> => {
  let session: Stripe.Checkout.Session;
  try {
    session = await args.stripe.checkout.sessions.retrieve(args.sessionId);
  } catch (error) {
    // Stripe throws (rather than returning null) for unknown ids.
    if (error instanceof Stripe.errors.StripeError) {
      throwMaskedStripeError(error, args.notFoundMessage);
    }
    // Non-Stripe errors must not reach the client verbatim — tRPC's default
    // error formatter would forward the raw message.
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to retrieve checkout session",
      cause: error,
    });
  }

  if (session.client_reference_id !== args.workspaceId) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: args.notFoundMessage,
    });
  }

  return session;
};

// stripe-node v22 exports error classes as values only; derive the type.
type StripeError = InstanceType<typeof Stripe.errors.StripeError>;

/**
 * A missing or deleted resource. Must be checked BEFORE `handleStripeError`,
 * where resource_missing (a StripeInvalidRequestError) maps to BAD_REQUEST.
 */
export const isStripeNotFound = (error: StripeError): boolean => {
  return error.statusCode === 404 || error.code === "resource_missing";
};

export const handleStripeError = (error: StripeError): never => {
  let code: TRPCError["code"];

  // The class checks deliberately precede the not-found check:
  // resource_missing also covers a missing sub-resource in request params
  // (e.g. a bogus payment-method id), which must stay BAD_REQUEST, and a
  // 404-shaped permission error must stay FORBIDDEN.
  if (error instanceof Stripe.errors.StripeAuthenticationError) {
    code = "UNAUTHORIZED";
  } else if (error instanceof Stripe.errors.StripeRateLimitError) {
    code = "TOO_MANY_REQUESTS";
  } else if (error instanceof Stripe.errors.StripeInvalidRequestError) {
    code = "BAD_REQUEST";
  } else if (error instanceof Stripe.errors.StripePermissionError) {
    code = "FORBIDDEN";
  } else if (isStripeNotFound(error)) {
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
    message: `Stripe error: ${error.message}`,
  });
};

/**
 * Maps a Stripe error for an endpoint that must not reveal whether a probed
 * id exists: not-found shaped errors throw NOT_FOUND with the caller's masked
 * message (never Stripe's "No such ..." text), everything else goes through
 * `handleStripeError`. Permission errors are exempt from the mask even when
 * 404-shaped: they fire identically for every id (revealing nothing to an id
 * prober), and masking them would misreport a restricted-key
 * misconfiguration as missing billing data.
 */
export const throwMaskedStripeError = (error: StripeError, notFoundMessage: string): never => {
  if (!(error instanceof Stripe.errors.StripePermissionError) && isStripeNotFound(error)) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: notFoundMessage,
    });
  }
  // `return` proves to the compiler that this function cannot fall through.
  return handleStripeError(error);
};
