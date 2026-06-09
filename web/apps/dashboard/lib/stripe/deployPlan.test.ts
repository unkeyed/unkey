import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { detectDeployPlan } from "./deployPlan";

// Minimal subscription stub: detectDeployPlan only reads items[].price.id and
// items[].price.metadata.deploy_plan.
function subWithItems(...items: Array<{ id?: string; deployPlan?: string }>): Stripe.Subscription {
  return {
    id: "sub_test",
    items: {
      data: items.map(({ id, deployPlan }) => ({
        price: {
          id: id ?? "price_x",
          metadata: deployPlan === undefined ? {} : { deploy_plan: deployPlan },
        },
      })),
    },
  } as unknown as Stripe.Subscription;
}

describe("detectDeployPlan", () => {
  it("maps the plan-fee metadata to its plan", () => {
    expect(detectDeployPlan(subWithItems({ deployPlan: "starter" }))).toBe("starter");
    expect(detectDeployPlan(subWithItems({ deployPlan: "pro" }))).toBe("pro");
    expect(detectDeployPlan(subWithItems({ deployPlan: "business" }))).toBe("business");
  });

  it("finds the tagged plan-fee item among other items", () => {
    const sub = subWithItems(
      { id: "price_api_plan" },
      { id: "price_deploy_metered_cpu" },
      { id: "price_deploy_fee", deployPlan: "pro" },
    );
    expect(detectDeployPlan(sub)).toBe("pro");
  });

  it("trims surrounding whitespace in the metadata value", () => {
    expect(detectDeployPlan(subWithItems({ deployPlan: " pro " }))).toBe("pro");
  });

  it("returns null when no item carries Deploy plan metadata", () => {
    expect(detectDeployPlan(subWithItems({ id: "price_api_plan" }))).toBeNull();
  });

  it("returns null for a subscription with no items", () => {
    expect(detectDeployPlan(subWithItems())).toBeNull();
  });

  it("fails closed on an unrecognized plan value and logs a warning", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(detectDeployPlan(subWithItems({ deployPlan: "enterprise" }))).toBeNull();
    expect(warn).toHaveBeenCalledOnce();
    warn.mockRestore();
  });
});
