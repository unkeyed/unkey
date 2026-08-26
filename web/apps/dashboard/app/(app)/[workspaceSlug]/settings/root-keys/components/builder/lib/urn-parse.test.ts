import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { ACTIONS, ALL_INSTANCES, RESOURCE_SCOPES, type ResourceScope } from "./catalogue.types";
import { type Policy, newPolicy, setRowsActions } from "./policy";
import { TEMPLATES } from "./templates";
import { buildUrns } from "./urn";
import { grantsToPolicies } from "./urn-parse";

const ws = "ws_123";

const sorted = (grants: readonly string[]): string[] => [...grants].sort();

const everything = (scope: ResourceScope, instances: string[] = [ALL_INSTANCES]): Policy => {
  const policy = newPolicy(scope);
  return {
    ...policy,
    instances,
    selection: setRowsActions(policy.selection, catalogueRows(CATALOGUES[scope]), ACTIONS),
  };
};

const roundTrip = (policies: Policy[]) => {
  const grants = buildUrns(ws, policies);
  const mapped = grantsToPolicies(ws, grants);
  return { grants, mapped, reemitted: buildUrns(ws, mapped.policies) };
};

describe("grantsToPolicies", () => {
  it("re-emits every template it is given", () => {
    for (const template of TEMPLATES) {
      const { grants, mapped, reemitted } = roundTrip(template.materialise());
      expect(mapped.unmapped, template.id).toEqual([]);
      expect(sorted(reemitted), template.id).toEqual(sorted(grants));
    }
  });

  it("re-emits every scope with every action, for all instances", () => {
    for (const scope of RESOURCE_SCOPES) {
      const { grants, mapped, reemitted } = roundTrip([everything(scope)]);
      expect(mapped.unmapped, scope).toEqual([]);
      expect(sorted(reemitted), scope).toEqual(sorted(grants));
    }
  });

  it("re-emits every scope with every action, for named instances", () => {
    for (const scope of RESOURCE_SCOPES) {
      if (CATALOGUES[scope].instanceNoun === null) {
        continue;
      }
      const { grants, mapped, reemitted } = roundTrip([everything(scope, ["one_1", "two_2"])]);
      expect(mapped.unmapped, scope).toEqual([]);
      expect(sorted(reemitted), scope).toEqual(sorted(grants));
    }
  });

  it("reads back the shape of the card that produced it", () => {
    const policy: Policy = {
      scope: "ratelimit-namespaces",
      instances: [ALL_INSTANCES],
      selection: { namespace: ["read"], override: ["read", "write", "delete"] },
    };
    expect(grantsToPolicies(ws, buildUrns(ws, [policy])).policies).toEqual([policy]);
  });

  it("keeps the decrypt tick separate from write", () => {
    const policy: Policy = {
      scope: "keyspaces",
      instances: ["ks_1"],
      selection: { key: ["decrypt"] },
    };
    expect(grantsToPolicies(ws, buildUrns(ws, [policy])).policies).toEqual([policy]);
    expect(
      grantsToPolicies(ws, buildUrns(ws, [{ ...policy, selection: { key: ["write"] } }])).policies,
    ).toEqual([{ ...policy, selection: { key: ["write"] } }]);
  });

  it("merges named instances that share a selection", () => {
    const grants = buildUrns(ws, [
      { scope: "keyspaces", instances: ["ks_1"], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: ["ks_2"], selection: { keyspace: ["read"] } },
    ]);
    expect(grantsToPolicies(ws, grants).policies).toEqual([
      { scope: "keyspaces", instances: ["ks_1", "ks_2"], selection: { keyspace: ["read"] } },
    ]);
  });

  it("never merges all instances with named ones", () => {
    const grants = buildUrns(ws, [
      { scope: "keyspaces", instances: [ALL_INSTANCES], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: ["ks_1"], selection: { keyspace: ["read"] } },
    ]);
    expect(grantsToPolicies(ws, grants).policies).toEqual([
      { scope: "keyspaces", instances: [ALL_INSTANCES], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: ["ks_1"], selection: { keyspace: ["read"] } },
    ]);
  });

  it("refuses to guess when a tick is only half granted", () => {
    const partial = "unkey:v1:ws_123:keyspaces/ks_1/keys/*#read_key";
    expect(grantsToPolicies(ws, [partial])).toEqual({ policies: [], unmapped: [partial] });
  });

  it("hands back legacy names, alien workspaces and unknown paths untouched", () => {
    const grants = [
      "api.ks_1.read_key",
      "*",
      "unkey:v1:ws_other:identities/*#read_identity",
      "unkey:v1:ws_123:teleporters/*#read_teleporter",
      "unkey:v1:ws_123:identities/*#read_identity",
    ];
    expect(grantsToPolicies(ws, grants)).toEqual({
      policies: [
        { scope: "identities", instances: [ALL_INSTANCES], selection: { identity: ["read"] } },
      ],
      unmapped: [
        "api.ks_1.read_key",
        "*",
        "unkey:v1:ws_other:identities/*#read_identity",
        "unkey:v1:ws_123:teleporters/*#read_teleporter",
      ],
    });
  });

  it("maps nothing from an empty grant list", () => {
    expect(grantsToPolicies(ws, [])).toEqual({ policies: [], unmapped: [] });
  });
});
