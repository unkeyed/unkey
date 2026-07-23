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
 * Reads the id from a Stripe expandable field, which is either a bare id or
 * the expanded object. Returns null when absent, so callers decide what a
 * missing relation means.
 */
export const expandableId = (value: string | { id: string } | null | undefined): string | null => {
  if (!value) {
    return null;
  }
  return typeof value === "string" ? value : value.id;
};

type RetrieveWorkspaceCheckoutSessionArgs = {
  stripe: Stripe;
  sessionId: string;
  workspaceId: string;
  // Surfaced to the client on both a missing and a foreign session, so it must
  // name the resource the caller asked for, not the session.
  notFoundMessage: string;
};

/**
 * Returns the session only if it belongs to this workspace. The session id
 * comes from the URL, so this is the authorization boundary for the /success
 * flow (ENG-2927).
 *
 * A missing session and one owned by another workspace both raise NOT_FOUND
 * with `notFoundMessage`, so a caller cannot probe which ids exist.
 *
 * Says nothing about whether checkout finished. Callers that write billing
 * state want [retrieveCompletedWorkspaceCheckoutSession] instead.
 */
export const retrieveWorkspaceCheckoutSession = async (
  args: RetrieveWorkspaceCheckoutSessionArgs,
): Promise<Stripe.Checkout.Session> => {
  let session: Stripe.Checkout.Session;
  try {
    session = await args.stripe.checkout.sessions.retrieve(args.sessionId);
  } catch (error) {
    if (error instanceof Stripe.errors.StripeError) {
      throwMaskedStripeError(error, args.notFoundMessage);
    }

    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to retrieve checkout session",
      cause: error,
    });
  }

  if (session.client_reference_id !== args.workspaceId) {
    // Same response as a missing session, so the id is not confirmed. The
    // cause separates them in logs: this one is an attack signal. No ids in
    // it, they would be cross-tenant PII.
    throw new TRPCError({
      code: "NOT_FOUND",
      message: args.notFoundMessage,
      cause: new Error("checkout session client_reference_id does not match the workspace"),
    });
  }

  return session;
};

/**
 * As [retrieveWorkspaceCheckoutSession], and additionally requires that
 * checkout finished.
 *
 * Callers bind billing to this session's customer, and an open or expired
 * session's customer may have no payment method, which would leave the
 * workspace pointing at a customer that can never be charged. Ownership is
 * proven before this check, so it can say what is wrong rather than masking
 * as not-found.
 */
export const retrieveCompletedWorkspaceCheckoutSession = async (
  args: RetrieveWorkspaceCheckoutSessionArgs,
): Promise<Stripe.Checkout.Session> => {
  const session = await retrieveWorkspaceCheckoutSession(args);

  if (session.status !== "complete") {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Checkout has not been completed",
    });
  }

  return session;
};

type StripeError = InstanceType<typeof Stripe.errors.StripeError>;

/**
 * Reports whether Stripe says a resource is missing. Both signals are needed:
 * some endpoints send `resource_missing` with a 400, others send a bare 404
 * with no code.
 */
export const isStripeNotFound = (error: StripeError): boolean => {
  return error.statusCode === 404 || error.code === "resource_missing";
};

/**
 * Maps a Stripe error onto the tRPC status to answer with.
 *
 * Branch order matters. Invalid-request wins over `isStripeNotFound` because
 * `resource_missing` also fires for a bad sub-resource in the parameters, such
 * as a bogus payment-method id, which is a bad request and not a missing
 * target. Permission wins too, so a restricted key is not reported as missing
 * data.
 */
export const stripeErrorCode = (error: StripeError): TRPCError["code"] => {
  if (error instanceof Stripe.errors.StripeAuthenticationError) {
    return "UNAUTHORIZED";
  }
  if (error instanceof Stripe.errors.StripeRateLimitError) {
    return "TOO_MANY_REQUESTS";
  }
  if (error instanceof Stripe.errors.StripeInvalidRequestError) {
    return "BAD_REQUEST";
  }
  if (error instanceof Stripe.errors.StripePermissionError) {
    return "FORBIDDEN";
  }
  if (isStripeNotFound(error)) {
    return "NOT_FOUND";
  }
  return "INTERNAL_SERVER_ERROR";
};

/**
 * Legacy helper that surfaces Stripe's own text ("Stripe error: <message>").
 * That text names the objects involved and, on auth failures, an identifying
 * key, so new code must use [throwRedactedStripeError] or
 * [throwMaskedStripeError] instead. Retained only for `getSetupIntent`, which
 * is being reworked separately in ENG-3080 (PR #6829).
 */
export const handleStripeError = (error: StripeError): never => {
  throw new TRPCError({
    code: stripeErrorCode(error),
    message: `Stripe error: ${error.message}`,
  });
};

/**
 * Replaces Stripe's text on failures the user cannot act on. Names no Stripe
 * object and no key.
 */
const STRIPE_REQUEST_FAILED =
  "The billing provider rejected the request. Please try again or contact support@unkey.com if this issue persists.";

/**
 * Throws the caller's message with the status Stripe's error maps to. Keeps
 * Stripe's text off the wire, since it names the objects involved ("No such
 * PaymentMethod: 'pm_x'") and, on auth failures, an identifying API key. The
 * original error stays on `cause` for the logs.
 *
 * The type sits on the binding, not the arrow: TypeScript only treats a call
 * as never-returning when the callee is a const with an explicit type.
 */
export const throwRedactedStripeError: (error: StripeError, message: string) => never = (
  error,
  message,
) => {
  throw new TRPCError({
    code: stripeErrorCode(error),
    message,
    cause: error,
  });
};

/**
 * Redacts like [throwRedactedStripeError], and also collapses not-found onto
 * the caller's message so a probed id looks the same as one the workspace
 * does not own.
 *
 * Permission errors stay unmasked. They fire identically for every id, so
 * they leak nothing, and calling a misconfigured key "missing data" sends
 * on-call hunting the wrong bug.
 */
export const throwMaskedStripeError: (error: StripeError, notFoundMessage: string) => never = (
  error,
  notFoundMessage,
) => {
  if (!(error instanceof Stripe.errors.StripePermissionError) && isStripeNotFound(error)) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: notFoundMessage,
      cause: error,
    });
  }
  return throwRedactedStripeError(error, STRIPE_REQUEST_FAILED);
};
