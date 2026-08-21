import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { ACTIONS, type PermissionRow, RESOURCE_SCOPES } from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import { ALL_INSTANCES, newPolicy, setRowActions } from "./policy";
import { buildUrn, buildUrns, instancePath, isValidResourcePath, urnActions } from "./urn";

const ws = "ws_123";

const rowOf = (scope: (typeof RESOURCE_SCOPES)[number], id: string): PermissionRow => {
  const row = catalogueRows(CATALOGUES[scope]).find((entry) => entry.id === id);
  if (!row) {
    throw new Error(`no row ${id} in ${scope}`);
  }
  return row;
};

describe("urnActions", () => {
  it("derives the concrete names from the resource noun", () => {
    const row = rowOf("workspace", "identity");
    expect(urnActions(row, "read")).toEqual([{ name: "read_identity" }]);
    expect(urnActions(row, "write")).toEqual([
      { name: "create_identity" },
      { name: "update_identity" },
    ]);
    expect(urnActions(row, "delete")).toEqual([{ name: "delete_identity" }]);
  });

  it("keeps the resource noun for every row in the workspace catalogue", () => {
    expect(catalogueRows(workspaceCatalogue).map((row) => urnActions(row, "read"))).toEqual([
      [{ name: "read_identity" }],
      [{ name: "read_role" }],
      [{ name: "read_permission" }],
      [{ name: "read_vault_key" }],
    ]);
  });

  it("prefers the names a row declares", () => {
    expect(urnActions(rowOf("keyspaces", "key"), "read")).toEqual([
      { name: "read_key" },
      { name: "verify_key" },
    ]);
    expect(urnActions(rowOf("keyspaces", "keyspace"), "write")).toEqual([
      { name: "update_keyspace" },
    ]);
    expect(urnActions(rowOf("ratelimit-namespaces", "override"), "write")).toEqual([
      { name: "set_override" },
    ]);
  });

  it("lets a create action move to its parent path", () => {
    expect(urnActions(rowOf("keyspaces", "key"), "write")).toEqual([
      { name: "create_key", path: "keyspaces/{instance}" },
      { name: "update_key" },
    ]);
  });

  it("names one action for every row and coarse action in every catalogue", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        for (const action of ACTIONS) {
          expect(urnActions(row, action).length).toBeGreaterThan(0);
        }
      }
    }
  });
});

describe("instancePath", () => {
  it("substitutes a concrete instance id", () => {
    expect(instancePath("keyspaces/{instance}/keys/*", "ks_1")).toBe("keyspaces/ks_1/keys/*");
  });

  it("substitutes a single-segment wildcard for all instances", () => {
    expect(instancePath("ratelimits/namespaces/{instance}", ALL_INSTANCES)).toBe(
      "ratelimits/namespaces/*",
    );
  });

  it("leaves a path without the token alone", () => {
    expect(instancePath("rbac/roles/*", "ks_1")).toBe("rbac/roles/*");
  });
});

describe("buildUrn", () => {
  it("composes the versioned urn with the action suffix", () => {
    expect(buildUrn(ws, "rbac/roles/*", "create_role")).toBe(
      "unkey:v1:ws_123:rbac/roles/*#create_role",
    );
  });

  it("passes a trailing descendant pattern through unchanged", () => {
    expect(buildUrn(ws, "projects/proj_123/**", "delete_deployment")).toBe(
      "unkey:v1:ws_123:projects/proj_123/**#delete_deployment",
    );
  });
});

describe("isValidResourcePath", () => {
  it("accepts single-segment wildcards anywhere", () => {
    expect(isValidResourcePath("identities/*")).toBe(true);
    expect(isValidResourcePath("keyspaces/*/keys/*")).toBe(true);
  });

  it("accepts a descendant wildcard only in trailing position", () => {
    expect(isValidResourcePath("**")).toBe(true);
    expect(isValidResourcePath("projects/proj_123/**")).toBe(true);
    expect(isValidResourcePath("**/deployments/*")).toBe(false);
    expect(isValidResourcePath("projects/proj_123/**/deployments/*")).toBe(false);
  });

  it("rejects empty and partial-segment wildcards", () => {
    expect(isValidResourcePath("")).toBe(false);
    expect(isValidResourcePath("identities/")).toBe(false);
    expect(isValidResourcePath("identities/id_*")).toBe(false);
  });

  it("holds for every catalogue path, for one instance and for all", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        for (const action of ACTIONS) {
          for (const grant of urnActions(row, action)) {
            expect(isValidResourcePath(instancePath(grant.path ?? row.path, "id_1"))).toBe(true);
            expect(isValidResourcePath(instancePath(grant.path ?? row.path, ALL_INSTANCES))).toBe(
              true,
            );
          }
        }
      }
    }
  });
});

describe("buildUrns", () => {
  it("returns nothing for a policy with no grants", () => {
    expect(buildUrns(ws, [newPolicy()])).toEqual([]);
  });

  it("expands a container-less row into one urn per concrete action", () => {
    const policy = { ...newPolicy(), selection: setRowActions({}, "role", ["read", "write"]) };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:rbac/roles/*#read_role",
      "unkey:v1:ws_123:rbac/roles/*#create_role",
      "unkey:v1:ws_123:rbac/roles/*#update_role",
    ]);
  });

  it("covers the whole workspace catalogue", () => {
    const rows = catalogueRows(workspaceCatalogue);
    const selection = rows.reduce(
      (acc, row) => setRowActions(acc, row.id, ["read", "write", "delete"]),
      {},
    );
    expect(buildUrns(ws, [{ ...newPolicy(), selection }])).toEqual([
      "unkey:v1:ws_123:identities/*#read_identity",
      "unkey:v1:ws_123:identities/*#create_identity",
      "unkey:v1:ws_123:identities/*#update_identity",
      "unkey:v1:ws_123:identities/*#delete_identity",
      "unkey:v1:ws_123:rbac/roles/*#read_role",
      "unkey:v1:ws_123:rbac/roles/*#create_role",
      "unkey:v1:ws_123:rbac/roles/*#update_role",
      "unkey:v1:ws_123:rbac/roles/*#delete_role",
      "unkey:v1:ws_123:rbac/permissions/*#read_permission",
      "unkey:v1:ws_123:rbac/permissions/*#create_permission",
      "unkey:v1:ws_123:rbac/permissions/*#update_permission",
      "unkey:v1:ws_123:rbac/permissions/*#delete_permission",
      "unkey:v1:ws_123:vault/keys/*#read_vault_key",
      "unkey:v1:ws_123:vault/keys/*#create_vault_key",
      "unkey:v1:ws_123:vault/keys/*#update_vault_key",
      "unkey:v1:ws_123:vault/keys/*#delete_vault_key",
    ]);
  });

  it("deduplicates urns shared by two policies", () => {
    const policy = { ...newPolicy(), selection: setRowActions({}, "identity", ["read"]) };
    expect(buildUrns(ws, [policy, policy])).toEqual(["unkey:v1:ws_123:identities/*#read_identity"]);
  });

  it("carries the workspace id it was given", () => {
    const policy = { ...newPolicy(), selection: setRowActions({}, "identity", ["delete"]) };
    expect(buildUrns("ws_other", [policy])).toEqual([
      "unkey:v1:ws_other:identities/*#delete_identity",
    ]);
  });

  it("puts every keyspace grant on the keyspace it was picked for", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      instances: ["ks_1"],
      selection: setRowActions(setRowActions({}, "keyspace", ["read"]), "key", [
        "read",
        "write",
        "delete",
      ]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:keyspaces/ks_1#read_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/ks_1#create_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#update_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#delete_key",
    ]);
  });

  it("repeats a keyspace grant for every instance picked", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      instances: ["ks_1", "ks_2"],
      selection: setRowActions({}, "key", ["read"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/ks_2/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/ks_2/keys/*#verify_key",
    ]);
  });

  it("wildcards the id segment for all keyspaces", () => {
    const policy = { ...newPolicy("keyspaces"), selection: setRowActions({}, "key", ["read"]) };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
    ]);
  });

  it("covers the keyspace catalogue", () => {
    const selection = catalogueRows(keyspacesCatalogue).reduce(
      (acc, row) => setRowActions(acc, row.id, ["read", "write", "delete"]),
      {},
    );
    expect(buildUrns(ws, [{ ...newPolicy("keyspaces"), instances: ["ks_1"], selection }])).toEqual([
      "unkey:v1:ws_123:keyspaces/ks_1#read_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1#update_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1#delete_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/ks_1#create_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#update_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#delete_key",
    ]);
  });

  it("covers the ratelimit namespace catalogue", () => {
    const selection = catalogueRows(ratelimitNamespacesCatalogue).reduce(
      (acc, row) => setRowActions(acc, row.id, ["read", "write", "delete"]),
      {},
    );
    expect(
      buildUrns(ws, [{ ...newPolicy("ratelimit-namespaces"), instances: ["rlns_1"], selection }]),
    ).toEqual([
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1#read_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1#limit",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1#update_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1#delete_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1/overrides/*#read_override",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1/overrides/*#set_override",
      "unkey:v1:ws_123:ratelimits/namespaces/rlns_1/overrides/*#delete_override",
    ]);
  });

  it("keeps two scopes in one key apart", () => {
    const keyspace = {
      ...newPolicy("keyspaces"),
      selection: setRowActions({}, "key", ["read"]),
    };
    const namespace = {
      ...newPolicy("ratelimit-namespaces"),
      selection: setRowActions({}, "override", ["read"]),
    };
    expect(buildUrns(ws, [keyspace, namespace])).toEqual([
      "unkey:v1:ws_123:keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#read_override",
    ]);
  });
});
