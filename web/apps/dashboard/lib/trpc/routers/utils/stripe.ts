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
  notFoundMessage: string;
};

/**
 * A missing session and one owned by another workspace both raise NOT_FOUND
 * with `notFoundMessage`, so a caller cannot probe which session ids exist.
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
    // Must match the missing-session response so the id is not confirmed. The
    // cause separates the two in logs and carries no ids (cross-tenant PII).
    throw new TRPCError({
      code: "NOT_FOUND",
      message: args.notFoundMessage,
      cause: new Error("checkout session client_reference_id does not match the workspace"),
    });
  }

  return session;
};

/**
 * As [retrieveWorkspaceCheckoutSession], but also rejects an unfinished
 * checkout: its customer may have no payment method, so binding billing to it
 * would leave the workspace pointing at a customer that can never be charged.
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

// Both signals are needed: some endpoints send `resource_missing` with a 400,
// others send a bare 404 with no code.
export const isStripeNotFound = (error: StripeError): boolean => {
  return error.statusCode === 404 || error.code === "resource_missing";
};

// Branch order matters: invalid-request and permission are checked before
// `isStripeNotFound`, because `resource_missing` also fires for a bad
// sub-resource in the parameters (a bad request, not a missing target) and for
// a restricted key (a permission problem).
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

// Deliberately carries no "contact support" line. Callers own that: the ones
// that render this to a user append it themselves, and a message that already
// had it forced them to sniff the prose to avoid printing it twice.
const STRIPE_REQUEST_FAILED = "The billing provider rejected the request. Please try again.";

/**
 * Answers with `message` and Stripe's mapped status, keeping Stripe's own text
 * (which names objects and keys) off the wire; the raw error stays on `cause`.
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
 * `notFoundMessage` so a probed id looks the same as one the workspace does
 * not own. Permission errors stay unmasked: they fire identically for every id
 * so leak nothing, and masking them would report a misconfigured key as
 * missing data.
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
