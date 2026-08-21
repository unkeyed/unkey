import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { policyIdentity } from "@/lib/collections/deploy/policies.schema";
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
      key: policyIdentity("firewall", "KEBAP"),
      production: { id: "pol_a1" },
      preview: { id: "pol_a2" },
    });
  });

  it("keys an unpaired policy by its identity, not its id, when the identity is unique in its own environment", () => {
    // A row in one environment only is the common case. Its key must stay an
    // identity, because the row actions read a policy from a key.
    const merged = mergePolicies(
      [firewall("pol_a1", "Prod only")],
      [firewall("pol_b1", "Preview only")],
    );

    expect(merged).toHaveLength(2);
    expect(merged.find((m) => m.name === "Prod only")).toMatchObject({
      key: policyIdentity("firewall", "Prod only"),
      production: { id: "pol_a1" },
      preview: null,
    });
    expect(merged.find((m) => m.name === "Preview only")).toMatchObject({
      key: policyIdentity("firewall", "Preview only"),
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

  it("falls back to an id-based key for a true in-environment duplicate", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);

    expect(merged.map((m) => m.key).sort()).toEqual(["production:pol_a1", "production:pol_a2"]);
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
      key: policyIdentity("firewall", "Guard"),
      production: { id: "pol_a1" },
      preview: null,
    });
    expect(merged.find((m) => m.type === "ratelimit")).toMatchObject({
      key: policyIdentity("ratelimit", "Guard"),
      production: null,
      preview: { id: "pol_b1" },
    });
  });

  it("keeps the same name of two types in one environment as two addressable rows", () => {
    const merged = mergePolicies([firewall("pol_a1", "Guard"), ratelimit("pol_a2", "Guard")], []);

    expect(new Set(merged.map((m) => m.key)).size).toBe(2);
    expect(policyInEnv(merged, policyIdentity("firewall", "Guard"), "production")?.id).toBe(
      "pol_a1",
    );
    expect(policyInEnv(merged, policyIdentity("ratelimit", "Guard"), "production")?.id).toBe(
      "pol_a2",
    );
  });
});

describe("name folding", () => {
  it("pairs names that differ only in case or surrounding space", () => {
    const merged = mergePolicies([firewall("pol_a1", "Auth ")], [firewall("pol_b1", "auth")]);

    expect(merged).toHaveLength(1);
    expect(merged[0]).toMatchObject({
      name: "Auth ",
      production: { id: "pol_a1" },
      preview: { id: "pol_b1" },
    });
  });

  it("does not pair two names that fold the same within one environment", () => {
    const merged = mergePolicies([firewall("pol_a1", "Auth"), firewall("pol_a2", "auth")], []);

    expect(merged.map((m) => m.key).sort()).toEqual(["production:pol_a1", "production:pol_a2"]);
  });
});

// The row actions read a policy from a key. A duplicate identity gets an id
// key, so the reader must accept both key shapes.
describe("policyInEnv", () => {
  it("resolves an identity-keyed row in each environment", () => {
    const merged = mergePolicies([firewall("pol_a1", "KEBAP")], [firewall("pol_b1", "KEBAP")]);
    const key = policyIdentity("firewall", "KEBAP");

    expect(policyInEnv(merged, key, "production")?.id).toBe("pol_a1");
    expect(policyInEnv(merged, key, "preview")?.id).toBe("pol_b1");
  });

  it("resolves an id-keyed duplicate-name row that a name lookup would miss", () => {
    const merged = mergePolicies([firewall("pol_a1", "Dup"), firewall("pol_a2", "Dup")], []);

    expect(policyInEnv(merged, "production:pol_a2", "production")?.id).toBe("pol_a2");
    expect(policyInEnv(merged, "production:pol_a2", "preview")).toBeNull();
  });

  it("returns null for an environment the row does not exist in", () => {
    const merged = mergePolicies([firewall("pol_a1", "Prod only")], []);

    expect(policyInEnv(merged, policyIdentity("firewall", "Prod only"), "preview")).toBeNull();
  });

  it("returns null for an unknown key", () => {
    expect(policyInEnv(mergePolicies([], []), "nope", "production")).toBeNull();
  });
});

// The form rejects a spaces-only name, the API accepts one.
describe("blank names", () => {
  it("never yields an empty key, which React cannot use", () => {
    const merged = mergePolicies([firewall("pol_a1", "   ")], [firewall("pol_b1", "   ")]);
    expect(merged.map((m) => m.key)).toEqual(["production:pol_a1", "preview:pol_b1"]);
  });

  it("does not pair two unrelated unnamed policies across environments", () => {
    const merged = mergePolicies([firewall("pol_a1", "   ")], [firewall("pol_b1", "\t")]);
    expect(merged).toHaveLength(2);
    expect(merged.every((m) => m.production === null || m.preview === null)).toBe(true);
  });
});
