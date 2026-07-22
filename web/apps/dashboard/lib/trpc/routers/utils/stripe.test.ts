import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import {
  expandableId,
  isStripeNotFound,
  retrieveCompletedWorkspaceCheckoutSession,
  retrieveWorkspaceCheckoutSession,
  stripeErrorCode,
  throwMaskedStripeError,
  throwRedactedStripeError,
} from "./stripe";

const WORKSPACE_ID = "ws_victim";

const incompleteStatuses: Stripe.Checkout.Session.Status[] = ["open", "expired"];

// Partial literals cast through `unknown`, as in
// lib/stripe/linkDeploySubscription.test.ts. A full Checkout.Session literal
// is impractical to construct.
function session(overrides: Partial<Stripe.Checkout.Session> = {}): Stripe.Checkout.Session {
  return {
    id: "cs_1",
    client_reference_id: WORKSPACE_ID,
    customer: "cus_1",
    status: "complete",
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

// Asserts the value is an Error and narrows it, so `cause` assertions need no
// cast.
function asError(value: unknown): Error {
  expect(value).toBeInstanceOf(Error);
  if (!(value instanceof Error)) {
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
   * Another workspace's session id must not reach that workspace's billing
   * objects (ENG-2927). NOT_FOUND rather than FORBIDDEN, so the rejection
   * does not confirm the session exists.
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
   * The response cannot separate an ownership violation from a stale id, so
   * the cause must, or the attack looks like noise in the logs. No ids in it,
   * they would be cross-tenant PII.
   */
  it("marks an ownership mismatch on the cause without naming any id", async () => {
    const { stripe } = stubStripe(session({ client_reference_id: "ws_attacker" }));

    const error = await rejection(
      retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
    );

    const cause = asError(error.cause);
    expect(cause.message).toContain("client_reference_id");
    expect(cause.message).not.toContain("cs_1");
    expect(cause.message).not.toContain("ws_attacker");
  });

  /**
   * Reads report a session's status, so they must be able to see one that is
   * not complete. Only the billing writes require completion, via
   * [retrieveCompletedWorkspaceCheckoutSession].
   */
  it.each(incompleteStatuses)("returns a %s session the workspace owns", async (status) => {
    const { stripe } = stubStripe(session({ status }));

    const result = await retrieveWorkspaceCheckoutSession({
      stripe,
      sessionId: "cs_1",
      workspaceId: WORKSPACE_ID,
      notFoundMessage: "Customer not found",
    });

    expect(result.status).toBe(status);
  });

  /**
   * A session with no `client_reference_id` was created outside the billing
   * flow and belongs to nobody.
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
   * A nonexistent id must look like an ownership mismatch, or the error tells
   * a caller probing ids which sessions exist. stripe-node throws
   * resource_missing here, it never resolves null.
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
   * A non-404 StripeInvalidRequestError is a real request defect. Masking the
   * whole class would hide such bugs as "not found".
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

  it("maps non-not-found Stripe errors through stripeErrorCode", async () => {
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
   * tRPC's default formatter forwards the thrown message, so a non-Stripe
   * error must not reach the client verbatim.
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

describe("retrieveCompletedWorkspaceCheckoutSession", () => {
  it("returns a complete session the workspace owns", async () => {
    const { stripe } = stubStripe(session());

    const result = await retrieveCompletedWorkspaceCheckoutSession({
      stripe,
      sessionId: "cs_1",
      workspaceId: WORKSPACE_ID,
      notFoundMessage: "Customer not found",
    });

    expect(result.id).toBe("cs_1");
  });

  /**
   * Callers bind billing to this session's customer, and an abandoned
   * checkout's customer may have no payment method.
   */
  it.each(incompleteStatuses)("rejects a %s session the workspace owns", async (status) => {
    const { stripe } = stubStripe(session({ status }));

    const error = await rejection(
      retrieveCompletedWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
    );

    expect(error.code).toBe("PRECONDITION_FAILED");
    expect(error.message).toBe("Checkout has not been completed");
  });

  /**
   * Completion is checked after ownership, so a foreign session is still
   * reported as missing and never as incomplete, which would confirm it
   * exists.
   */
  it("reports a foreign incomplete session as not found", async () => {
    const { stripe } = stubStripe(session({ client_reference_id: "ws_attacker", status: "open" }));

    await expectNotFound(
      retrieveCompletedWorkspaceCheckoutSession({
        stripe,
        sessionId: "cs_1",
        workspaceId: WORKSPACE_ID,
        notFoundMessage: "Customer not found",
      }),
      "Customer not found",
    );
  });
});

describe("stripeErrorCode", () => {
  /**
   * Pins the branch order. The invalid-request class wins over the
   * resource_missing code, which also covers a bad sub-resource in the
   * params, such as a bogus payment-method id.
   */
  it("maps a resource_missing StripeInvalidRequestError to BAD_REQUEST", () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "No such PaymentMethod: 'pm_x'",
    });

    expect(stripeErrorCode(error)).toBe("BAD_REQUEST");
  });

  /**
   * Defensive branch only. stripe-node routes every 404 through
   * `StripeError.generate`, which dispatches on `type` and returns
   * StripeInvalidRequestError, so this shape never arrives from a real
   * response.
   */
  it("maps a 404 outside the invalid-request class to NOT_FOUND", () => {
    const error = new Stripe.errors.StripeAPIError({
      type: "api_error",
      statusCode: 404,
      message: "Not found",
    });

    expect(stripeErrorCode(error)).toBe("NOT_FOUND");
  });

  /**
   * A restricted key reports an inaccessible resource as missing: 403 with
   * code resource_missing. The permission class must win over that code, or a
   * key misconfiguration reads as a missing billing resource.
   */
  it("maps a resource_missing StripePermissionError to FORBIDDEN", () => {
    const error = new Stripe.errors.StripePermissionError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 403,
      message: "This API key does not have access to this resource",
    });

    expect(stripeErrorCode(error)).toBe("FORBIDDEN");
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

    expect(stripeErrorCode(authError)).toBe("UNAUTHORIZED");
    expect(stripeErrorCode(rateLimitError)).toBe("TOO_MANY_REQUESTS");
  });
});

describe("throwMaskedStripeError", () => {
  /**
   * Surfaces the caller's message, never Stripe's "No such ..." text, so
   * probed ids stay indistinguishable.
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

  /**
   * Masking hides outages too. A wrong-mode key turns every retrieve into a
   * not-found, which reads like users passing stale ids unless the Stripe
   * error survives on the cause.
   */
  it("keeps the original Stripe error on the cause", () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "No such setupintent: 'seti_x'",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.cause).toBe(error);
  });

  /**
   * The non-not-found path is the one that leaks. A wrong key makes Stripe
   * answer 401 naming the key, and the client renders whatever the server
   * throws.
   */
  it("redacts non-not-found errors instead of forwarding Stripe's text", () => {
    const error = new Stripe.errors.StripeAuthenticationError({
      type: "invalid_request_error",
      message: "Invalid API key provided: sk_live_************************abcd",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.code).toBe("UNAUTHORIZED");
    expect(thrown.message).not.toContain("sk_live_");
    expect(thrown.message).not.toContain("Invalid API key");
    expect(thrown.cause).toBe(error);
  });

  /**
   * A permission error fires identically for every id, so FORBIDDEN reveals
   * nothing to an id prober, while masking it would report a key
   * misconfiguration as missing billing data.
   */
  it("does not mask a resource_missing StripePermissionError", () => {
    const error = new Stripe.errors.StripePermissionError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 403,
      message: "This API key does not have access to this resource",
    });

    const thrown = asTRPCError(
      caught(() => throwMaskedStripeError(error, "Setup intent not found")),
    );
    expect(thrown.code).toBe("FORBIDDEN");
    expect(thrown.message).not.toBe("Setup intent not found");
  });
});

describe("throwRedactedStripeError", () => {
  /**
   * On `customers.update` Stripe's message names the customer and the payment
   * method, and the client renders it verbatim. Only the text is replaced,
   * the status still follows the error class.
   */
  it("replaces Stripe's message but keeps its status and the error as cause", () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 404,
      message: "No such PaymentMethod: 'pm_x'",
    });

    const thrown = asTRPCError(caught(() => throwRedactedStripeError(error, "Update failed")));
    expect(thrown.code).toBe("BAD_REQUEST");
    expect(thrown.message).toBe("Update failed");
    expect(thrown.message).not.toContain("pm_x");
    expect(thrown.cause).toBe(error);
  });

  it("carries the status through for non-not-found errors", () => {
    const error = new Stripe.errors.StripeRateLimitError({
      type: "rate_limit_error",
      message: "Too many requests",
    });

    expect(asTRPCError(caught(() => throwRedactedStripeError(error, "Update failed"))).code).toBe(
      "TOO_MANY_REQUESTS",
    );
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
