import { describe, expect, it } from "vitest";
import { paywallCopy } from "./paywall-copy";

describe("paywallCopy", () => {
  it("offers both products for team members", () => {
    expect(paywallCopy("team", null).products).toEqual(["compute", "api"]);
  });

  it("tells starter workspaces what they lack", () => {
    expect(paywallCopy("team", "starter").description).toMatch(/^Starter doesn't include/);
    expect(paywallCopy("custom-domains", "starter").description).toMatch(/^Starter includes 1/);
    expect(paywallCopy("team", null).description).not.toMatch(/Starter/);
  });

  it("limits deploy and custom domains to Compute plans", () => {
    expect(paywallCopy("deploy", null).products).toEqual(["compute"]);
    expect(paywallCopy("custom-domains", "starter").products).toEqual(["compute"]);
  });

  it("limits the API quota wall to API plans", () => {
    expect(paywallCopy("api-limit", "pro").products).toEqual(["api"]);
  });
});
