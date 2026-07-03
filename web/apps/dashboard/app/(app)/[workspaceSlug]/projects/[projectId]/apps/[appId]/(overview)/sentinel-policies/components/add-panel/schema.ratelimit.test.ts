import { describe, expect, it } from "vitest";
import { policyFormSchema } from "./schema";

// Exercises the keyauth ratelimit override validation. The Go verify path honors
// three override shapes (cost alone, inline limit+duration, and limit+duration+cost)
// and silently ignores a partial limit/duration pair, so the form must accept the
// former and reject the latter.
function keyauthWithRatelimit(rl: Record<string, unknown>) {
  return {
    type: "keyauth" as const,
    name: "p",
    environmentId: "__all__",
    matchConditions: [],
    keySpaceIds: ["ks_1"],
    locations: [],
    permissionQuery: "",
    ratelimits: [{ id: "1", name: "expensive", ...rl }],
  };
}

describe("keyauth ratelimit override", () => {
  it("accepts cost-only override", () => {
    const r = policyFormSchema.safeParse(keyauthWithRatelimit({ override: true, cost: 5 }));
    expect(r.success).toBe(true);
  });

  it("accepts limit + duration override", () => {
    const r = policyFormSchema.safeParse(
      keyauthWithRatelimit({ override: true, limit: 100, duration: 60000 }),
    );
    expect(r.success).toBe(true);
  });

  it("accepts limit + duration + cost override", () => {
    const r = policyFormSchema.safeParse(
      keyauthWithRatelimit({ override: true, limit: 100, duration: 60000, cost: 2 }),
    );
    expect(r.success).toBe(true);
  });

  it("accepts a bare named reference when override is off", () => {
    const r = policyFormSchema.safeParse(keyauthWithRatelimit({ override: false }));
    expect(r.success).toBe(true);
  });

  it("rejects a partial inline override (limit without duration)", () => {
    const r = policyFormSchema.safeParse(keyauthWithRatelimit({ override: true, limit: 100 }));
    expect(r.success).toBe(false);
  });

  it("rejects a partial inline override (duration without limit)", () => {
    const r = policyFormSchema.safeParse(keyauthWithRatelimit({ override: true, duration: 60000 }));
    expect(r.success).toBe(false);
  });

  it("rejects an override toggled on with no values", () => {
    const r = policyFormSchema.safeParse(keyauthWithRatelimit({ override: true }));
    expect(r.success).toBe(false);
  });
});
