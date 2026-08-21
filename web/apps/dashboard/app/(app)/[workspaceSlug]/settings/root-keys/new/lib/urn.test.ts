import { describe, expect, it } from "vitest";
import { catalogueRows } from "./catalogue";
import { workspaceCatalogue } from "./catalogue.workspace";
import { newPolicy, setRowActions } from "./policy";
import { buildUrn, buildUrns, isValidResourcePath, urnActions } from "./urn";

const ws = "ws_123";

describe("urnActions", () => {
  it("maps the coarse actions onto concrete action names", () => {
    expect(urnActions("identity", "read")).toEqual(["read_identity"]);
    expect(urnActions("identity", "write")).toEqual(["create_identity", "update_identity"]);
    expect(urnActions("identity", "delete")).toEqual(["delete_identity"]);
  });

  it("keeps the resource noun for every row in the workspace catalogue", () => {
    expect(
      catalogueRows(workspaceCatalogue).map((row) => urnActions(row.resource, "read")),
    ).toEqual([["read_identity"], ["read_role"], ["read_permission"], ["read_vault_key"]]);
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

  it("holds for every path in the workspace catalogue", () => {
    for (const row of catalogueRows(workspaceCatalogue)) {
      expect(isValidResourcePath(row.path)).toBe(true);
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
});
