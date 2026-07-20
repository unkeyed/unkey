import { describe, expect, it } from "vitest";
import { keepsTeamAfterDelete, matchSubscriptionColumn } from "./webhookRouting";

describe("matchSubscriptionColumn", () => {
  it("matches the API column", () => {
    expect(
      matchSubscriptionColumn(
        { stripeSubscriptionId: "sub_api", stripeDeploySubscriptionId: "sub_deploy" },
        "sub_api",
      ),
    ).toBe("api");
  });

  it("matches the Deploy column", () => {
    expect(
      matchSubscriptionColumn(
        { stripeSubscriptionId: "sub_api", stripeDeploySubscriptionId: "sub_deploy" },
        "sub_deploy",
      ),
    ).toBe("deploy");
  });

  it("returns null when neither column matches", () => {
    expect(
      matchSubscriptionColumn(
        { stripeSubscriptionId: "sub_api", stripeDeploySubscriptionId: "sub_deploy" },
        "sub_other",
      ),
    ).toBeNull();
  });

  it("returns null when both columns are empty", () => {
    expect(
      matchSubscriptionColumn(
        { stripeSubscriptionId: null, stripeDeploySubscriptionId: null },
        "sub_api",
      ),
    ).toBeNull();
  });

  it("prefers the API column on a degenerate tie", () => {
    expect(
      matchSubscriptionColumn(
        { stripeSubscriptionId: "sub_same", stripeDeploySubscriptionId: "sub_same" },
        "sub_same",
      ),
    ).toBe("api");
  });
});

describe("keepsTeamAfterDelete", () => {
  it("API delete keeps team while a Deploy plan is active", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Free", plan: "pro" })).toBe(true);
  });

  it("API delete drops team when no Deploy plan remains", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Pro", plan: null })).toBe(false);
  });

  it("Deploy delete keeps team while the API tier is paid", () => {
    expect(keepsTeamAfterDelete("deploy", { tier: "Pro", plan: "pro" })).toBe(true);
  });

  it("Deploy delete drops team when the API tier is Free", () => {
    expect(keepsTeamAfterDelete("deploy", { tier: "Free", plan: "pro" })).toBe(false);
  });

  it("Deploy delete treats a null tier as Free", () => {
    expect(keepsTeamAfterDelete("deploy", { tier: null, plan: "pro" })).toBe(false);
  });
});
