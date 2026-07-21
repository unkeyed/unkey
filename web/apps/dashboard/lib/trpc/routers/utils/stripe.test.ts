import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import {
  expandableId,
  handleStripeError,
  isStripeNotFound,
  retrieveWorkspaceCheckoutSession,
  throwMaskedStripeError,
} from "./stripe";

const WORKSPACE_ID = "ws_victim";

// Partial literals cast through `unknown`, matching the stub pattern in
// lib/stripe/linkDeploySubscription.test.ts — a full Checkout.Session literal
// is impractical to construct.
function session(overrides: Partial<Stripe.Checkout.Session> = {}): Stripe.Checkout.Session {
  return {
    id: "cs_1",
    client_reference_id: WORKSPACE_ID,
    customer: "cus_1",
    ...overrides,
  } as unknown as Stripe.Checkout.Session;
}

function stubStripe(retrieved: Stripe.Checkout.Session | Error) {
  const retrieve = vi.fn(async () => {
    if (retrieved instanceof Error) {
      throw retrieved;
    }
    return retrieved;
  });
  return {
    stripe: { checkout: { sessions: { retrieve } } } as unknown as Stripe,
    retrieve,
  };
}

// Runs fn and returns what it throws. Fails the test if it does not throw.
function caught(fn: () => unknown): unknown {
  try {
    fn();
  } catch (error) {
    return error;
  }
  throw new Error("expected the call to throw");
}

// Asserts the value is a TRPCError and narrows it for further assertions.
function asTRPCError(value: unknown): TRPCError {
  expect(value).toBeInstanceOf(TRPCError);
  if (!(value instanceof TRPCError)) {
    throw new Error("unreachable: asserted above");
  }
  return value;
}

async function rejection(promise: Promise<unknown>): Promise<TRPCError> {
  return asTRPCError(await promise.catch((err: unknown) => err));
}

async function expectNotFound(promise: Promise<unknown>, message: string) {
  const error = await rejection(promise);
  expect(error.code).toBe("NOT_FOUND");
  expect(error.message).toBe(message);
}

describe("retrieveWorkspaceCheckoutSession", () => {
  it("retrieves the given session id and returns the session when it was created for this workspace", async () => {
    const { stripe, retrieve } = stubStripe(session());

    const result = await retrieveWorkspaceCheckoutSession({
      stripe,
      sessionId: "cs_1",
      workspaceId: WORKSPACE_ID,
      notFoundMessage: "Customer not found",
    });

    expect(retrieve).toHaveBeenCalledWith("cs_1");
    expect(expandableId(result.customer)).toBe("cus_1");
  });

  /**
   * Guarantees that a session id belonging to another workspace cannot be
   * used to reach that workspace's billing objects (ENG-2927). The rejection
   * must be NOT_FOUND, not FORBIDDEN, so it does not confirm the session
   * exists.
   */
  it("rejects a session belonging to another workspace with NOT_FOUND", async () => {
    const { stripe } = stubStripe(session({ client_reference_id: "ws_attacker" }));

    await expectNotFound(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
      "Customer not found",
    );
  });

  /**
   * A session with no `client_reference_id` (created outside the billing
   * flow) must not be treated as belonging to the workspace.
   */
  it("rejects a session with no client_reference_id", async () => {
    const { stripe } = stubStripe(session({ client_reference_id: null }));

    await expectNotFound(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
      "Customer not found",
    );
  });

  /**
   * A nonexistent session id (stripe-node throws resource_missing, never
   * resolves null) must be indistinguishable from an ownership mismatch, or
   * the error would tell a caller probing ids which sessions exist.
   */
  it("maps a nonexistent session id onto the same NOT_FOUND as a mismatch", async () => {
    const { stripe } = stubStripe(
      new Stripe.errors.StripeInvalidRequestError({
        type: "invalid_request_error",
        code: "resource_missing",
        statusCode: 404,
        message: "No such checkout.session: 'cs_missing'",
      }),
    );

    await expectNotFound(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_missing",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
      "Customer not found",
    );
  });

  /**
   * A non-404 StripeInvalidRequestError is a genuine request defect and must
   * keep its BAD_REQUEST mapping — widening the mask to the whole class would
   * hide such bugs as "not found".
   */
  it("does not mask a non-404 StripeInvalidRequestError as NOT_FOUND", async () => {
    const { stripe } = stubStripe(
      new Stripe.errors.StripeInvalidRequestError({
        type: "invalid_request_error",
        code: "parameter_invalid_string",
        statusCode: 400,
        message: "Invalid string: sessionId",
      }),
    );

    const error = await rejection(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
    );

    expect(error.code).toBe("BAD_REQUEST");
    expect(error.message).not.toBe("Customer not found");
  });

  it("maps non-not-found Stripe errors through handleStripeError", async () => {
    const { stripe } = stubStripe(
      new Stripe.errors.StripeAuthenticationError({
        type: "invalid_request_error",
        message: "Invalid API key provided",
      }),
    );

    const error = await rejection(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
    );

    expect(error.code).toBe("UNAUTHORIZED");
  });

  /**
   * Non-Stripe errors must not reach the client verbatim — tRPC's default
   * error formatter forwards the thrown message.
   */
  it("masks non-Stripe errors as a generic INTERNAL_SERVER_ERROR with cause", async () => {
    const networkError = new Error("socket hang up");
    const { stripe } = stubStripe(networkError);

    const error = await rejection(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
    );

    expect(error.code).toBe("INTERNAL_SERVER_ERROR");
    expect(error.message).toBe("Failed to retrieve checkout session");
    expect(error.cause).toBe(networkError);
  });
});

describe("handleStripeError", () => {
  /**
   * Pins the deliberate branch order: the StripeInvalidRequestError class
   * wins over the resource_missing code, because resource_missing also covers
   * a missing sub-resource in request params (e.g. a bogus payment-method
   * id), which must stay BAD_REQUEST.
   */
  it("maps a resource_missing StripeInvalidRequestError to BAD_REQUEST", () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "No such PaymentMethod: 'pm_x'",
    });

    expect(asTRPCError(caught(() => handleStripeError(error))).code).toBe("BAD_REQUEST");
  });

  it("maps a 404 outside the invalid-request class to NOT_FOUND", () => {
    const error = new Stripe.errors.StripeAPIError({
      type: "api_error",
      statusCode: 404,
      message: "Not found",
    });

    expect(asTRPCError(caught(() => handleStripeError(error))).code).toBe("NOT_FOUND");
  });

  /**
   * A restricted API key can mask an inaccessible resource as missing; the
   * permission class must win, or a key misconfiguration would be
   * misdiagnosed as a missing billing resource.
   */
  it("maps a 404-shaped StripePermissionError to FORBIDDEN", () => {
    const error = new Stripe.errors.StripePermissionError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "This API key does not have access to this resource",
    });

    expect(asTRPCError(caught(() => handleStripeError(error))).code).toBe("FORBIDDEN");
  });

  it("maps auth and rate-limit errors to their codes", () => {
    const authError = new Stripe.errors.StripeAuthenticationError({
      type: "invalid_request_error",
      message: "Invalid API key provided",
    });
    const rateLimitError = new Stripe.errors.StripeRateLimitError({
      type: "rate_limit_error",
      message: "Too many requests",
    });

    expect(asTRPCError(caught(() => handleStripeError(authError))).code).toBe("UNAUTHORIZED");
    expect(asTRPCError(caught(() => handleStripeError(rateLimitError))).code).toBe(
      "TOO_MANY_REQUESTS",
    );
  });
});

describe("throwMaskedStripeError", () => {
  /**
   * A not-found shaped error must surface the caller's masked message, never
   * Stripe's "No such ..." text, so probed ids stay indistinguishable.
   */
  it("masks not-found shaped errors with the caller's message", () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "No such setupintent: 'seti_x'",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.code).toBe("NOT_FOUND");
    expect(thrown.message).toBe("Setup intent not found");
  });

  it("delegates non-not-found errors to handleStripeError", () => {
    const error = new Stripe.errors.StripeAuthenticationError({
      type: "invalid_request_error",
      message: "Invalid API key provided",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.code).toBe("UNAUTHORIZED");
  });

  /**
   * A permission error fires identically for every id, so surfacing FORBIDDEN
   * reveals nothing to an id prober — masking it would misreport a key
   * misconfiguration as missing billing data.
   */
  it("does not mask a 404-shaped StripePermissionError", () => {
    const error = new Stripe.errors.StripePermissionError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "This API key does not have access to this resource",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.code).toBe("FORBIDDEN");
    expect(thrown.message).not.toBe("Setup intent not found");
  });
});

describe("isStripeNotFound", () => {
  it("matches on statusCode 404 or code resource_missing", () => {
    expect(
      isStripeNotFound(
        new Stripe.errors.StripeInvalidRequestError({
          type: "invalid_request_error",
          code: "resource_missing",
          message: "No such customer: 'cus_x'",
        }),
      ),
    ).toBe(true);
    expect(
      isStripeNotFound(
        new Stripe.errors.StripeAPIError({
          type: "api_error",
          statusCode: 404,
          message: "Not found",
        }),
      ),
    ).toBe(true);
    expect(
      isStripeNotFound(
        new Stripe.errors.StripeInvalidRequestError({
          type: "invalid_request_error",
          code: "parameter_invalid_string",
          statusCode: 400,
          message: "Invalid string",
        }),
      ),
    ).toBe(false);
  });
});

describe("expandableId", () => {
  it("reads the id from both the bare and expanded shapes", () => {
    expect(expandableId("cus_1")).toBe("cus_1");
    expect(expandableId({ id: "cus_1" })).toBe("cus_1");
  });

  it("returns null for absent values", () => {
    expect(expandableId(null)).toBeNull();
    expect(expandableId(undefined)).toBeNull();
  });
});
