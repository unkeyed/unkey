import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { ACTIONS, ALL_INSTANCES, RESOURCE_SCOPES, type ResourceScope } from "./catalogue.types";
import { type Policy, newPolicy, setRowsActions } from "./policy";
import { TEMPLATES } from "./templates";
import { buildUrns } from "./urn";
import { grantsToPolicies } from "./urn-parse";

const ws = "ws_123";

const KEYSPACE = "projects/proj_1/keyspaces/ks_1";
const KEYSPACE_2 = "projects/proj_2/keyspaces/ks_2";

// Two instances per scope, spelled as the whole path down to each one.
const NAMED_INSTANCES: Partial<Record<ResourceScope, string[]>> = {
  projects: ["proj_1", "proj_2"],
  apps: ["projects/proj_1/apps/app_1", "projects/proj_2/apps/app_2"],
  environments: [
    "projects/proj_1/apps/app_1/environments/env_1",
    "projects/proj_2/apps/app_2/environments/env_2",
  ],
  keyspaces: [KEYSPACE, KEYSPACE_2],
  "ratelimit-namespaces": [
    "projects/proj_1/ratelimits/namespaces/rlns_1",
    "projects/proj_2/ratelimits/namespaces/rlns_2",
  ],
};

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
      const instances = NAMED_INSTANCES[scope];
      if (instances === undefined) {
        throw new Error(`no named instances for ${scope}`);
      }
      const { grants, mapped, reemitted } = roundTrip([everything(scope, instances)]);
      expect(mapped.unmapped, scope).toEqual([]);
      expect(sorted(reemitted), scope).toEqual(sorted(grants));
    }
  });

  it("reads back the shape of the card that produced it", () => {
    const policy: Policy = {
      scope: "ratelimit-namespaces",
      instances: ["projects/proj_1/ratelimits/namespaces/rlns_1"],
      selection: { ratelimit_namespace: ["read"], ratelimit_override: ["read", "write", "delete"] },
    };
    expect(grantsToPolicies(ws, buildUrns(ws, [policy])).policies).toEqual([policy]);
  });

  it("gives a fully wildcarded key back as one workspace card", () => {
    const policy: Policy = {
      scope: "keyspaces",
      instances: [ALL_INSTANCES],
      selection: { key: ["verify"] },
    };
    expect(grantsToPolicies(ws, buildUrns(ws, [policy])).policies).toEqual([
      { scope: "workspace", instances: [ALL_INSTANCES], selection: { key: ["verify"] } },
    ]);
  });

  it("keeps the decrypt tick separate from write", () => {
    const policy: Policy = {
      scope: "keyspaces",
      instances: [KEYSPACE],
      selection: { key: ["decrypt"] },
    };
    expect(grantsToPolicies(ws, buildUrns(ws, [policy])).policies).toEqual([policy]);
    expect(
      grantsToPolicies(ws, buildUrns(ws, [{ ...policy, selection: { key: ["write"] } }])).policies,
    ).toEqual([{ ...policy, selection: { key: ["write"] } }]);
  });

  it("merges named instances that share a selection", () => {
    const grants = buildUrns(ws, [
      { scope: "keyspaces", instances: [KEYSPACE], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: [KEYSPACE_2], selection: { keyspace: ["read"] } },
    ]);
    expect(grantsToPolicies(ws, grants).policies).toEqual([
      {
        scope: "keyspaces",
        instances: [KEYSPACE, KEYSPACE_2],
        selection: { keyspace: ["read"] },
      },
    ]);
  });

  it("never merges all instances with named ones", () => {
    const grants = buildUrns(ws, [
      { scope: "keyspaces", instances: [ALL_INSTANCES], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: [KEYSPACE], selection: { keyspace: ["read"] } },
    ]);
    expect(grantsToPolicies(ws, grants).policies).toEqual([
      { scope: "workspace", instances: [ALL_INSTANCES], selection: { keyspace: ["read"] } },
      { scope: "keyspaces", instances: [KEYSPACE], selection: { keyspace: ["read"] } },
    ]);
  });

  it("refuses to guess at an action no row offers", () => {
    const stray = `unkey:v1:ws_123:${KEYSPACE}/keys/*#encrypt_key`;
    expect(grantsToPolicies(ws, [stray])).toEqual({ policies: [], unmapped: [stray] });
  });

  it("hands back legacy names, alien workspaces and unknown paths untouched", () => {
    const grants = [
      "api.ks_1.read_key",
      "*",
      "unkey:v1:ws_other:projects/*/identities/*#read_identity",
      "unkey:v1:ws_123:teleporters/*#read_teleporter",
      "unkey:v1:ws_123:projects/*/identities/*#read_identity",
    ];
    expect(grantsToPolicies(ws, grants)).toEqual({
      policies: [
        { scope: "workspace", instances: [ALL_INSTANCES], selection: { identity: ["read"] } },
      ],
      unmapped: [
        "api.ks_1.read_key",
        "*",
        "unkey:v1:ws_other:projects/*/identities/*#read_identity",
        "unkey:v1:ws_123:teleporters/*#read_teleporter",
      ],
    });
  });

  it("maps nothing from an empty grant list", () => {
    expect(grantsToPolicies(ws, [])).toEqual({ policies: [], unmapped: [] });
  });
});
