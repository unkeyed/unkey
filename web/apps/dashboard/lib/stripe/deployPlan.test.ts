import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { computeQuotasForPlan, detectDeployPlan } from "./deployPlan";

// Minimal subscription stub. detectDeployPlan reads items[].price.metadata.plan.
function subWithItems(...items: Array<{ id?: string; plan?: string }>): Stripe.Subscription {
  return {
    id: "sub_test",
    items: {
      data: items.map(({ id, plan }) => ({
        price: {
          id: id ?? "price_x",
          metadata: {
            ...(plan === undefined ? {} : { plan }),
          },
        },
      })),
    },
  } as unknown as Stripe.Subscription;
}

describe("detectDeployPlan", () => {
  it("maps the plan-fee metadata to its plan", () => {
    expect(detectDeployPlan(subWithItems({ plan: "starter" }))).toBe("starter");
    expect(detectDeployPlan(subWithItems({ plan: "pro" }))).toBe("pro");
    expect(detectDeployPlan(subWithItems({ plan: "business" }))).toBe("business");
  });

  it("finds the tagged plan-fee item among other items", () => {
    const sub = subWithItems(
      { id: "price_api_plan" },
      { id: "price_metered_cpu" },
      { id: "price_plan_fee", plan: "pro" },
    );
    expect(detectDeployPlan(sub)).toBe("pro");
  });

  it("trims surrounding whitespace in the metadata value", () => {
    expect(detectDeployPlan(subWithItems({ plan: " pro " }))).toBe("pro");
  });

  it("returns null when no item carries plan metadata", () => {
    expect(detectDeployPlan(subWithItems({ id: "price_api_plan" }))).toBeNull();
  });

  it("returns null for a subscription with no items", () => {
    expect(detectDeployPlan(subWithItems())).toBeNull();
  });

  it("fails closed on an unrecognized plan value and logs a warning", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(detectDeployPlan(subWithItems({ plan: "enterprise" }))).toBeNull();
    expect(warn).toHaveBeenCalledOnce();
    warn.mockRestore();
  });
});

describe("computeQuotasForPlan", () => {
  it("returns the advertised per-instance CPU and memory limits", () => {
    expect(computeQuotasForPlan("starter")).toEqual({
      maxCpuMillicoresPerInstance: 2_000,
      maxMemoryMibPerInstance: 2_048,
    });
    expect(computeQuotasForPlan("pro")).toEqual({
      maxCpuMillicoresPerInstance: 8_000,
      maxMemoryMibPerInstance: 8_192,
    });
    expect(computeQuotasForPlan("business")).toEqual({
      maxCpuMillicoresPerInstance: 16_000,
      maxMemoryMibPerInstance: 32_768,
    });
  });

  it("returns the default Compute limits without a plan", () => {
    expect(computeQuotasForPlan(null)).toEqual({
      maxCpuMillicoresPerInstance: 2_000,
      maxMemoryMibPerInstance: 4_096,
    });
  });
});
