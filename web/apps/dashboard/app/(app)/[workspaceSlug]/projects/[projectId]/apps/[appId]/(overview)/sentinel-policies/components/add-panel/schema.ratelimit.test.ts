import { describe, expect, it } from "vitest";
import { fromSentinelPolicy, policyFormSchema, toSentinelPolicy } from "./schema";
import type { PolicyFormValues, SentinelPolicy } from "./schema";

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

function ratelimitForm(
  identifiers: { id: string; source: "remoteIp" | "header" | "path"; value: string }[],
): PolicyFormValues {
  return {
    type: "ratelimit",
    name: "rl",
    environmentId: "__all__",
    matchConditions: [],
    limit: 100,
    windowMs: 60000,
    identifiers,
  };
}

// Every write must serialize the identifiers array, even with one row, so
// stored policies converge on the target shape and the deprecated single
// identifier can be removed later.
describe("ratelimit identifier serialization", () => {
  it("serializes one row to a one-entry identifiers array", () => {
    const wire = toSentinelPolicy(ratelimitForm([{ id: "1", source: "remoteIp", value: "" }]));
    expect(wire).not.toHaveProperty("ratelimit.identifier");
    expect(wire).toMatchObject({
      type: "ratelimit",
      ratelimit: { limit: 100, windowMs: 60000, identifiers: [{ remoteIp: {} }] },
    });
  });

  it("serializes multiple rows in order", () => {
    const wire = toSentinelPolicy(
      ratelimitForm([
        { id: "1", source: "header", value: "x-client-id" },
        { id: "2", source: "path", value: "" },
      ]),
    );
    expect(wire).toMatchObject({
      ratelimit: { identifiers: [{ header: { name: "x-client-id" } }, { path: {} }] },
    });
  });

  it("deserializes the deprecated single identifier into one row", () => {
    const stored: SentinelPolicy = {
      id: "pol_1",
      name: "rl",
      enabled: true,
      type: "ratelimit",
      ratelimit: { limit: 100, windowMs: 60000, identifier: { path: {} } },
    };
    const form = fromSentinelPolicy(stored, "__all__");
    expect(form).toMatchObject({
      type: "ratelimit",
      identifiers: [{ source: "path", value: "" }],
    });
  });

  it("deserializes the identifiers array into rows", () => {
    const stored: SentinelPolicy = {
      id: "pol_1",
      name: "rl",
      enabled: true,
      type: "ratelimit",
      ratelimit: {
        limit: 100,
        windowMs: 60000,
        identifiers: [{ authenticatedSubject: {} }, { header: { name: "x-tenant" } }],
      },
    };
    const form = fromSentinelPolicy(stored, "__all__");
    expect(form).toMatchObject({
      type: "ratelimit",
      identifiers: [
        { source: "authenticatedSubject", value: "" },
        { source: "header", value: "x-tenant" },
      ],
    });
  });
});
