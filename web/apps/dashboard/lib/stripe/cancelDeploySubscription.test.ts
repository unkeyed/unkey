import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { cancelDeploySubscription } from "./cancelDeploySubscription";
import type { DeployBillingConfig } from "./deployBilling";

const CONFIG: DeployBillingConfig = {
  planFeePriceIds: { starter: "price_starter", pro: "price_pro", business: "price_business" },
  meteredPriceIds: ["price_cpu", "price_memory", "price_egress", "price_disk", "price_active_keys"],
  allDeployPriceIds: new Set([
    "price_starter",
    "price_pro",
    "price_business",
    "price_cpu",
    "price_memory",
    "price_egress",
    "price_disk",
    "price_active_keys",
  ]),
};

function item(id: string, priceId: string) {
  return { id, price: { id: priceId } };
}

// fakeStripe returns a Stripe stub whose retrieve yields the given items and
// whose update records its calls for assertion.
function fakeStripe(items: Array<{ id: string; price: { id: string } }>) {
  const update = vi.fn().mockResolvedValue({});
  const stripe = {
    subscriptions: {
      retrieve: vi.fn().mockResolvedValue({ id: "sub_1", items: { data: items } }),
      update,
    },
  } as unknown as Stripe;
  return { stripe, update };
}

describe("cancelDeploySubscription", () => {
  it("returns 'none' and mutates nothing when there are no Deploy items", async () => {
    const { stripe, update } = fakeStripe([item("si_api", "price_api")]);
    const result = await cancelDeploySubscription(stripe, "sub_1", CONFIG);
    expect(result).toBe("none");
    expect(update).not.toHaveBeenCalled();
  });

  it("cancels a Deploy-only subscription at period end", async () => {
    const { stripe, update } = fakeStripe([
      item("si_fee", "price_pro"),
      item("si_cpu", "price_cpu"),
      item("si_mem", "price_memory"),
    ]);
    const result = await cancelDeploySubscription(stripe, "sub_1", CONFIG);
    expect(result).toBe("deploy_only");
    expect(update).toHaveBeenCalledWith("sub_1", { cancel_at_period_end: true });
  });

  it("drops only the plan-fee item(s) from a mixed subscription, keeping metered items", async () => {
    const { stripe, update } = fakeStripe([
      item("si_api", "price_api"),
      item("si_fee", "price_pro"),
      item("si_cpu", "price_cpu"),
    ]);
    const result = await cancelDeploySubscription(stripe, "sub_1", CONFIG);
    expect(result).toBe("mixed");
    expect(update).toHaveBeenCalledWith("sub_1", {
      items: [{ id: "si_fee", deleted: true }],
      proration_behavior: "none",
    });
  });
});
