import Stripe from "stripe";
import { describe, expect, it } from "vitest";
import { isStripeNotFound, retrieveSubscriptionOrNull } from "./notFound";

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

describe("retrieveSubscriptionOrNull", () => {
  function failingStripe(error: Error): Stripe {
    const stripe = new Stripe("sk_test_x");
    stripe.subscriptions.retrieve = async () => {
      throw error;
    };
    return stripe;
  }

  it("returns null when the subscription id is missing", async () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      param: "id",
      statusCode: 404,
      message: "No such subscription: 'sub_x'",
    });

    expect(await retrieveSubscriptionOrNull(failingStripe(error), "sub_x")).toBe(null);
  });

  it("rethrows a resource_missing permission error", async () => {
    const error = new Stripe.errors.StripePermissionError({
      type: "invalid_request_error",
      code: "resource_missing",
      statusCode: 403,
      message: "This API key does not have access to this resource",
    });

    await expect(retrieveSubscriptionOrNull(failingStripe(error), "sub_x")).rejects.toBe(error);
  });

  it("rethrows a resource_missing error on a param sub-resource", async () => {
    const error = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      param: "expand[0]",
      statusCode: 404,
      message: "No such object",
    });

    await expect(retrieveSubscriptionOrNull(failingStripe(error), "sub_x")).rejects.toBe(error);
  });
});
