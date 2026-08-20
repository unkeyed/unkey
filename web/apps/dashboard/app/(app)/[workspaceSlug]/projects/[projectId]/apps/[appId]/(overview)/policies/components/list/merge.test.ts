import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { describe, expect, it } from "vitest";
import { mergePolicies, policyInEnv } from "./merge";

function firewall(id: string, name: string): PolicyRow {
  return {
    id,
    name,
    enabled: true,
    type: "firewall",
    firewall: { action: "ACTION_DENY" },
    environmentId: "env_1",
    projectId: "proj_KEBAP",
    appId: "app_KEBAP",
  };
}

describe("mergePolicies", () => {
  it("pairs policies with the same name across environments", () => {
    const merged = mergePolicies([firewall("pol_a1", "KEBAP")], [firewall("pol_a2", "KEBAP")]);

    expect(merged).toHaveLength(1);
    expect(merged[0]).toMatchObject({
      key: "KEBAP",
      envA: { id: "pol_a1" },
      envB: { id: "pol_a2" },
    });
  });

  it("keys an unpaired policy by its name, not its id, when the name is unique in its own environment", () => {
    // A row in one environment only is the common case. Its key must stay a
    // name, because the row actions read a policy from a key.
    const merged = mergePolicies(
      [firewall("pol_a1", "Prod only")],
      [firewall("pol_b1", "Preview only")],
    );

    expect(merged).toHaveLength(2);
    expect(merged.find((m) => m.name === "Prod only")).toMatchObject({
      key: "Prod only",
      envA: { id: "pol_a1" },
      envB: null,
    });
    expect(merged.find((m) => m.name === "Preview only")).toMatchObject({
      key: "Preview only",
      envA: null,
      envB: { id: "pol_b1" },
    });
  });

  it("does not pair a name that occurs more than once within one environment", () => {
    const merged = mergePolicies(
      [firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")],
      [firewall("pol_b1", "Dup")],
    );

    // All three stay unpaired. The name occurs two times in list A, so it
    // cannot resolve to the single entry in list B.
    expect(merged).toHaveLength(3);
    expect(merged.every((m) => m.envA === null || m.envB === null)).toBe(true);
  });

  it("falls back to an id-based key for a true in-environment duplicate", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);

    expect(merged.map((m) => m.key).sort()).toEqual(["envA:pol_a1", "envA:pol_a2"]);
  });

  it("appends envB-only policies after all envA rows", () => {
    const merged = mergePolicies(
      [firewall("pol_a1", "A"), firewall("pol_a2", "B")],
      [firewall("pol_b1", "B"), firewall("pol_b2", "Only in B")],
    );

    expect(merged.map((m) => m.name)).toEqual(["A", "B", "Only in B"]);
  });
});

// The row actions read a policy from a key. A duplicate name gets an id key,
// so the reader must accept both key shapes.
describe("policyInEnv", () => {
  it("resolves a name-keyed row in each environment", () => {
    const merged = mergePolicies([firewall("pol_a1", "KEBAP")], [firewall("pol_b1", "KEBAP")]);

    expect(policyInEnv(merged, "KEBAP", "envA")?.id).toBe("pol_a1");
    expect(policyInEnv(merged, "KEBAP", "envB")?.id).toBe("pol_b1");
  });

  it("resolves an id-keyed duplicate-name row that a name lookup would miss", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);

    expect(policyInEnv(merged, "envA:pol_a2", "envA")?.id).toBe("pol_a2");
    expect(policyInEnv(merged, "envA:pol_a2", "envB")).toBeNull();
  });

  it("returns null for an environment the row does not exist in", () => {
    const merged = mergePolicies([firewall("pol_a1", "Prod only")], []);

    expect(policyInEnv(merged, "Prod only", "envB")).toBeNull();
  });

  it("returns null for an unknown key", () => {
    expect(policyInEnv(mergePolicies([], []), "nope", "envA")).toBeNull();
  });
});

describe("blank names", () => {
  it("never yields an empty key, which React cannot use", () => {
    const merged = mergePolicies([firewall("pol_a1", "")], [firewall("pol_b1", "")]);
    expect(merged.map((m) => m.key)).toEqual(["envA:pol_a1", "envB:pol_b1"]);
  });

  it("does not pair two unrelated unnamed policies across environments", () => {
    const merged = mergePolicies([firewall("pol_a1", "")], [firewall("pol_b1", "")]);
    expect(merged).toHaveLength(2);
    expect(merged.every((m) => m.envA === null || m.envB === null)).toBe(true);
  });
});
