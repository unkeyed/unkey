import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { cancelDeploySubscription } from "./cancelDeploySubscription";

// fakeStripe returns a Stripe stub whose retrieve yields a subscription with the
// given status and whose update records its calls for assertion.
function fakeStripe(status: Stripe.Subscription.Status) {
  const update = vi.fn().mockResolvedValue({});
  const stripe = {
    subscriptions: {
      retrieve: vi.fn().mockResolvedValue({ id: "sub_1", status }),
      update,
    },
  } as unknown as Stripe;
  return { stripe, update };
}

describe("cancelDeploySubscription", () => {
  it("cancels a live subscription at period end", async () => {
    const { stripe, update } = fakeStripe("active");
    await cancelDeploySubscription(stripe, "sub_1");
    expect(update).toHaveBeenCalledWith("sub_1", { cancel_at_period_end: true });
  });

  it("cancels a trialing subscription at period end", async () => {
    const { stripe, update } = fakeStripe("trialing");
    await cancelDeploySubscription(stripe, "sub_1");
    expect(update).toHaveBeenCalledWith("sub_1", { cancel_at_period_end: true });
  });

  it("no-ops on an already-canceled subscription", async () => {
    const { stripe, update } = fakeStripe("canceled");
    await cancelDeploySubscription(stripe, "sub_1");
    expect(update).not.toHaveBeenCalled();
  });

  it("no-ops on an incomplete_expired subscription", async () => {
    const { stripe, update } = fakeStripe("incomplete_expired");
    await cancelDeploySubscription(stripe, "sub_1");
    expect(update).not.toHaveBeenCalled();
  });
});
