import { describe, expect, it } from "vitest";
import { availableProducts, paywallCopy } from "./paywall-copy";

describe("paywallCopy", () => {
  it("offers both products for team members", () => {
    expect(paywallCopy("team").products).toEqual(["compute", "api"]);
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
});
