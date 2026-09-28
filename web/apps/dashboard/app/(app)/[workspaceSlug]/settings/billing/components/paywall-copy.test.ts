import { describe, expect, it } from "vitest";
import { paywallCopy } from "./paywall-copy";

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
});
