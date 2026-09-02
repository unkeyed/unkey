import { describe, expect, it } from "vitest";
import type { Policy } from "./schema";
import { fromPolicy, policyFormSchema, toPolicy } from "./schema";

// Exercises the keyauth credits override: the form accepts 0 and positive
// integers, rejects negatives, and round-trips the value to/from the wire.
function keyauthWithCredits(credits: unknown) {
  return {
    type: "keyauth" as const,
    name: "p",
    environmentId: "__all__",
    matchConditions: [],
    keyspaceIds: ["ks_1"],
    locations: [],
    permissionQuery: "",
    ratelimits: [],
    credits,
  };
}

describe("keyauth credits override", () => {
  it("accepts a credits override of 0", () => {
    const r = policyFormSchema.safeParse(keyauthWithCredits(0));
    expect(r.success).toBe(true);
  });

  it("accepts a positive credits override", () => {
    const r = policyFormSchema.safeParse(keyauthWithCredits(3));
    expect(r.success).toBe(true);
  });

  it("accepts an omitted credits override", () => {
    const r = policyFormSchema.safeParse(keyauthWithCredits(undefined));
    expect(r.success).toBe(true);
  });

  it("rejects a negative credits override", () => {
    const r = policyFormSchema.safeParse(keyauthWithCredits(-1));
    expect(r.success).toBe(false);
  });

  it("serializes a 0 credits override onto the wire", () => {
    const parsed = policyFormSchema.parse(keyauthWithCredits(0));
    const wire = toPolicy(parsed) as Extract<Policy, { type: "keyauth" }>;
    expect(wire.keyauth.credits).toBe(0);
  });

  it("omits credits from the wire when unset", () => {
    const parsed = policyFormSchema.parse(keyauthWithCredits(undefined));
    const wire = toPolicy(parsed) as Extract<Policy, { type: "keyauth" }>;
    expect(wire.keyauth.credits).toBeUndefined();
  });

  it("round-trips a credits override through the wire form", () => {
    const parsed = policyFormSchema.parse(keyauthWithCredits(3));
    const wire = toPolicy(parsed) as Policy;
    const back = fromPolicy(wire, "__all__");
    expect(back.type).toBe("keyauth");
    if (back.type === "keyauth") {
      expect(back.credits).toBe(3);
    }
  });
});
