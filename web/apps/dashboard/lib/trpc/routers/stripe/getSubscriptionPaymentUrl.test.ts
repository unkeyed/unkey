import type { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/stripe", () => ({ getStripeClient: vi.fn() }));
vi.mock("../../trpc", () => ({
  requireWorkspaceAdmin: {},
  workspaceProcedure: {
    use: () => ({ mutation: () => ({}) }),
  },
}));

import { resolveSubscriptionPaymentUrl } from "./getSubscriptionPaymentUrl";

const PAYMENT_URL = "https://invoice.stripe.com/i/acct_1/test";

function subscription(overrides: Partial<Stripe.Subscription> = {}): Stripe.Subscription {
  return {
    id: "sub_1",
    customer: "cus_1",
    status: "incomplete",
    latest_invoice: { hosted_invoice_url: PAYMENT_URL },
    ...overrides,
  } as unknown as Stripe.Subscription;
}

function stripeWith(result: Stripe.Subscription | Error): Stripe {
  return {
    subscriptions: {
      retrieve: vi.fn(async () => {
        if (result instanceof Error) {
          throw result;
        }
        return result;
      }),
    },
  } as unknown as Stripe;
}

describe("resolveSubscriptionPaymentUrl", () => {
  it.each(["incomplete", "past_due", "unpaid"] as const)(
    "returns the hosted invoice for a %s subscription",
    async (status) => {
      const stripe = stripeWith(subscription({ status }));

      await expect(resolveSubscriptionPaymentUrl(stripe, "sub_1", "cus_1")).resolves.toBe(
        PAYMENT_URL,
      );
    },
  );

  it("does not reveal another customer's invoice", async () => {
    await expect(
      resolveSubscriptionPaymentUrl(
        stripeWith(subscription({ customer: "cus_other" })),
        "sub_1",
        "cus_1",
      ),
    ).rejects.toMatchObject({ code: "NOT_FOUND" } satisfies Partial<TRPCError>);
  });

  it("rejects a subscription without a recoverable invoice", async () => {
    await expect(
      resolveSubscriptionPaymentUrl(
        stripeWith(subscription({ status: "active" })),
        "sub_1",
        "cus_1",
      ),
    ).rejects.toMatchObject({ code: "PRECONDITION_FAILED" } satisfies Partial<TRPCError>);
  });

  it("maps a deleted Stripe subscription to not found", async () => {
    const missing = new Stripe.errors.StripeInvalidRequestError({
      type: "invalid_request_error",
      code: "resource_missing",
      message: "No such subscription",
    });

    await expect(
      resolveSubscriptionPaymentUrl(stripeWith(missing), "sub_missing", "cus_1"),
    ).rejects.toMatchObject({ code: "NOT_FOUND" } satisfies Partial<TRPCError>);
  });
});
