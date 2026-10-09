import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { describe, expect, it } from "vitest";
import { type MergedPolicy, mergePolicies, policyInEnv } from "./merge";

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

function keyOf(merged: MergedPolicy[], id: string) {
  const row = merged.find((m) => m.production?.id === id || m.preview?.id === id);
  if (!row) {
    throw new Error(`no row holds ${id}`);
  }
  return row.key;
}

function ratelimit(id: string, name: string): PolicyRow {
  return {
    id,
    name,
    enabled: true,
    type: "ratelimit",
    ratelimit: { limit: 100, windowMs: 60000, identifiers: [{ remoteIp: {} }] },
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
      key: policyMatchKey("firewall", "KEBAP"),
      production: { id: "pol_a1" },
      preview: { id: "pol_a2" },
    });
  });

  it("keys an unpaired policy by its match key, not its id, when the match key is unique in its own environment", () => {
    // A row in one environment only is the common case. Its key must stay a
    // match key, because the row actions read a policy from a key.
    const merged = mergePolicies(
      [firewall("pol_a1", "Prod only")],
      [firewall("pol_b1", "Preview only")],
    );

    expect(merged).toHaveLength(2);
    expect(merged.find((m) => m.name === "Prod only")).toMatchObject({
      key: policyMatchKey("firewall", "Prod only"),
      production: { id: "pol_a1" },
      preview: null,
    });
    expect(merged.find((m) => m.name === "Preview only")).toMatchObject({
      key: policyMatchKey("firewall", "Preview only"),
      production: null,
      preview: { id: "pol_b1" },
    });
  });

  it("does not pair a name that occurs more than once within one environment", () => {
    const merged = mergePolicies(
      [firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")],
      [firewall("pol_b1", "Dup")],
    );

    // All three stay unpaired. The name occurs two times in production, so it
    // cannot resolve to the single entry in preview.
    expect(merged).toHaveLength(3);
    expect(merged.every((m) => m.production === null || m.preview === null)).toBe(true);
  });

  it("falls back to a position key for a true in-environment duplicate", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);

    expect(merged.map((m) => m.key)).toEqual([
      "production#0#firewall:Dup",
      "production#1#firewall:Dup",
    ]);
  });

  it("appends preview-only policies after all production rows", () => {
    const merged = mergePolicies(
      [firewall("pol_a1", "A"), firewall("pol_a2", "B")],
      [firewall("pol_b1", "B"), firewall("pol_b2", "Only in B")],
    );

    expect(merged.map((m) => m.name)).toEqual(["A", "B", "Only in B"]);
  });
});

describe("policy type", () => {
  it("does not pair the same name across environments when the types differ", () => {
    const merged = mergePolicies([firewall("pol_a1", "Guard")], [ratelimit("pol_b1", "Guard")]);

    expect(merged).toHaveLength(2);
    expect(merged.find((m) => m.type === "firewall")).toMatchObject({
      key: policyMatchKey("firewall", "Guard"),
      production: { id: "pol_a1" },
      preview: null,
    });
    expect(merged.find((m) => m.type === "ratelimit")).toMatchObject({
      key: policyMatchKey("ratelimit", "Guard"),
      production: null,
      preview: { id: "pol_b1" },
    });
  });

  it("keeps the same name of two types in one environment as two addressable rows", () => {
    const merged = mergePolicies([firewall("pol_a1", "Guard"), ratelimit("pol_a2", "Guard")], []);

    expect(new Set(merged.map((m) => m.key)).size).toBe(2);
    expect(policyInEnv(merged, keyOf(merged, "pol_a1"), "production")?.id).toBe("pol_a1");
    expect(policyInEnv(merged, keyOf(merged, "pol_a2"), "production")?.id).toBe("pol_a2");
  });
});

describe("name folding", () => {
  it("pairs names that differ only in surrounding space", () => {
    const merged = mergePolicies([firewall("pol_a1", "Auth ")], [firewall("pol_b1", "Auth")]);

    expect(merged).toHaveLength(1);
    expect(merged[0]).toMatchObject({
      name: "Auth ",
      production: { id: "pol_a1" },
      preview: { id: "pol_b1" },
    });
  });

  // Case is left alone: the API accepts both, and they are visibly different.
  it("keeps names that differ only in case apart", () => {
    const merged = mergePolicies([firewall("pol_a1", "Auth")], [firewall("pol_b1", "auth")]);

    expect(merged).toHaveLength(2);
  });
});

describe("policyInEnv", () => {
  it("resolves a row keyed by its match key in each environment", () => {
    const merged = mergePolicies([firewall("pol_a1", "KEBAP")], [firewall("pol_b1", "KEBAP")]);
    const key = keyOf(merged, "pol_a1");

    expect(key).toBe(policyMatchKey("firewall", "KEBAP"));
    expect(policyInEnv(merged, key, "production")?.id).toBe("pol_a1");
    expect(policyInEnv(merged, key, "preview")?.id).toBe("pol_b1");
  });

  it("resolves a position-keyed duplicate-name row that a name lookup would miss", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);
    const key = keyOf(merged, "pol_a2");

    expect(key).toBe("production#1#firewall:Dup");
    expect(policyInEnv(merged, key, "production")?.id).toBe("pol_a2");
    expect(policyInEnv(merged, key, "preview")).toBeNull();
  });

  it("keeps a position key across new ids, and drops it once another policy takes the place", () => {
    const key = mergePolicies([firewall("pol_a1", "A"), firewall("pol_a2", "")], [])[1].key;

    const rewritten = mergePolicies([firewall("pol_b1", "A"), firewall("pol_b2", "")], []);
    expect(policyInEnv(rewritten, key, "production")?.id).toBe("pol_b2");

    const shifted = mergePolicies([firewall("pol_c1", ""), firewall("pol_c2", "A")], []);
    expect(policyInEnv(shifted, key, "production")).toBeNull();
  });

  it("returns null for an environment the row does not exist in", () => {
    const merged = mergePolicies([firewall("pol_a1", "Prod only")], []);

    expect(policyInEnv(merged, keyOf(merged, "pol_a1"), "preview")).toBeNull();
  });

  it("returns null for an unknown key", () => {
    const key = keyOf(mergePolicies([firewall("pol_a1", "A")], []), "pol_a1");

    expect(policyInEnv(mergePolicies([], []), key, "production")).toBeNull();
  });
});

// The form rejects a spaces-only name, the API accepts one.
describe("blank names", () => {
  it("keys a blank name by its place", () => {
    const merged = mergePolicies([firewall("pol_a1", "   ")], [firewall("pol_b1", "   ")]);
    expect(merged.map((m) => m.key)).toEqual(["production#0#firewall:", "preview#0#firewall:"]);
  });

  it("does not pair two unrelated unnamed policies across environments", () => {
    const merged = mergePolicies([firewall("pol_a1", "   ")], [firewall("pol_b1", "\t")]);
    expect(merged).toHaveLength(2);
    expect(merged.every((m) => m.production === null || m.preview === null)).toBe(true);
  });
});
