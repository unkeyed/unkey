import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { describe, expect, it } from "vitest";
import { duplicateNameError, otherMatchKeys } from "./duplicate-name";
import { mergePolicies } from "./merge";

function row(type: "keyauth" | "ratelimit", name: string, index: number): PolicyRow {
  const base = { id: `pol_${index}`, name, enabled: true, environmentId: "env_prod" };
  const scope = { projectId: "proj_KEBAP", appId: "app_KEBAP", _order: index };
  return type === "keyauth"
    ? { ...base, ...scope, type, keyauth: { keyspaces: ["ks_1"] } }
    : {
        ...base,
        ...scope,
        type,
        ratelimit: { limit: 1, windowMs: 1000, identifiers: [{ remoteIp: {} }] },
      };
}

const merged = mergePolicies([row("keyauth", "Checkout", 0), row("ratelimit", "Public", 1)], []);

describe("duplicateNameError", () => {
  it("accepts an edited policy that keeps its own name", () => {
    const others = otherMatchKeys(merged, merged[0].key);
    expect(duplicateNameError({ type: "keyauth", name: "Checkout" }, others)).toBe(null);
  });

  it("rejects an edited policy renamed onto another policy", () => {
    const others = otherMatchKeys(merged, merged[0].key);
    expect(duplicateNameError({ type: "ratelimit", name: "Public" }, others)).toBe(
      'A Rate Limit policy named "Public" already exists. Open it to add it to another environment.',
    );
  });

  it("rejects a new policy that repeats a name of the same type", () => {
    expect(
      duplicateNameError({ type: "keyauth", name: " Checkout " }, otherMatchKeys(merged, null)),
    ).not.toBe(null);
  });
});
