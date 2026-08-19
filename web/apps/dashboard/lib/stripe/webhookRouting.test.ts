import { describe, expect, it } from "vitest";
import { keepsTeamAfterDelete } from "./webhookRouting";

describe("keepsTeamAfterDelete", () => {
  it("API delete keeps team while a team-enabled Deploy plan is active", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Free", plan: "pro" })).toBe(true);
    expect(keepsTeamAfterDelete("api", { tier: "Free", plan: "business" })).toBe(true);
  });

  it("API delete drops team on Starter or when no Deploy plan remains", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Pro", plan: "starter" })).toBe(false);
    expect(keepsTeamAfterDelete("api", { tier: "Pro", plan: null })).toBe(false);
  });

  it("Deploy delete keeps team while the API tier is paid", () => {
    expect(keepsTeamAfterDelete("compute", { tier: "Pro", plan: "pro" })).toBe(true);
  });

  it("Deploy delete drops team when the API tier is Free", () => {
    expect(keepsTeamAfterDelete("compute", { tier: "Free", plan: "pro" })).toBe(false);
  });

  it("Deploy delete treats a null tier as Free", () => {
    expect(keepsTeamAfterDelete("compute", { tier: null, plan: "pro" })).toBe(false);
  });
});
