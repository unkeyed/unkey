import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import { availableProducts, defaultProduct, paywallCopy } from "./paywall-copy";

describe("paywallCopy", () => {
  it("offers both products for team members", () => {
    expect(paywallCopy("team").products).toEqual(["compute", "api"]);
  });

  it("offers only Compute plans that include team members on the team wall", () => {
    const plans = paywallCopy("team").computePlans;
    expect(plans).toEqual(["pro", "business"]);
    expect(plans.every((plan) => limitsByPlan[plan].teamEnabled)).toBe(true);
  });

  it("limits deploy and custom domains to Compute plans", () => {
    expect(paywallCopy("deploy").products).toEqual(["compute"]);
    expect(paywallCopy("custom-domains").products).toEqual(["compute"]);
  });

  it("limits the API quota wall to API plans", () => {
    expect(paywallCopy("api-limit").products).toEqual(["api"]);
  });

  it("drops Compute when Compute billing is off", () => {
    expect(availableProducts(["compute", "api"], { computeEnabled: false })).toEqual(["api"]);
    expect(availableProducts(["compute", "api"], { computeEnabled: true })).toEqual([
      "compute",
      "api",
    ]);
    expect(availableProducts(["compute"], { computeEnabled: false })).toEqual([]);
  });

  it("opens on the product the workspace already uses", () => {
    const both = ["compute", "api"] as const;
    expect(defaultProduct([...both], { compute: 1200, api: 0 })).toBe("compute");
    expect(defaultProduct([...both], { compute: 1200, api: 90_000 })).toBe("compute");
    expect(defaultProduct([...both], { compute: 0, api: 90_000 })).toBe("api");
    expect(defaultProduct([...both], { compute: 0, api: 0 })).toBe("compute");
  });

  it("falls back to the first offered product", () => {
    expect(defaultProduct(["api"], { compute: 1200, api: 0 })).toBe("api");
    expect(defaultProduct([], { compute: 0, api: 0 })).toBeUndefined();
  });

  it("offers cancel and change controls only on the billing page pickers", () => {
    expect(paywallCopy("compute-plan")).toMatchObject({ products: ["compute"], manage: true });
    expect(paywallCopy("api-plan")).toMatchObject({ products: ["api"], manage: true });
    expect(paywallCopy("team").manage).toBe(false);
    expect(paywallCopy("choose-plan")).toMatchObject({
      products: ["compute", "api"],
      manage: false,
    });
  });
});
