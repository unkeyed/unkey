import { CATALOGUES } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/catalogue";
import {
  ACTIONS,
  type ScopeCatalogue,
  offeredActions,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/catalogue.types";
import { describe, expect, it } from "vitest";
import { grantableCatalogues, uncoveredPermissions } from "./grants";

const ws = "ws_test";
const urn = (resource: string, action: string) => `unkey:v1:${ws}:${resource}#${action}`;

describe("uncoveredPermissions", () => {
  it("covers everything with the admin wildcard", () => {
    const requested = [urn("rootKeys/*", "read"), urn("projects/*/apps/*", "write")];
    expect(uncoveredPermissions([urn("**", "*")], requested)).toEqual([]);
  });

  it("matches one segment per star", () => {
    const granted = [urn("projects/*", "read")];
    expect(uncoveredPermissions(granted, [urn("projects/*", "read")])).toEqual([]);
    expect(uncoveredPermissions(granted, [urn("projects/*/apps/*", "read")])).toEqual([
      urn("projects/*/apps/*", "read"),
    ]);
  });

  it("requires the same action unless the grant is a wildcard", () => {
    const requested = [urn("flags", "write")];
    expect(uncoveredPermissions([urn("flags", "read")], requested)).toEqual(requested);
  });

  it("rejects other workspaces", () => {
    const requested = [urn("flags", "read")];
    expect(uncoveredPermissions(["unkey:v1:ws_other:**#*"], requested)).toEqual(requested);
  });

  it("returns resources the role lacks", () => {
    const requested = [urn("flags", "read"), urn("rootKeys/*", "read")];
    expect(uncoveredPermissions([urn("flags", "read")], requested)).toEqual([
      urn("rootKeys/*", "read"),
    ]);
  });
});

describe("uncoveredPermissions with scoped grants", () => {
  const scoped = [urn("projects/proj_1/**", "read")];

  it("covers the base path and everything below it", () => {
    const requested = [urn("projects/proj_1", "read"), urn("projects/proj_1/apps/app_1", "read")];
    expect(uncoveredPermissions(scoped, requested)).toEqual([]);
  });

  it("does not cover a sibling project", () => {
    const requested = [urn("projects/proj_2/apps/app_1", "read")];
    expect(uncoveredPermissions(scoped, requested)).toEqual(requested);
  });

  it("ignores grants that are not URNs", () => {
    const requested = [urn("flags", "read")];
    expect(uncoveredPermissions(["api.*.read_api"], requested)).toEqual(requested);
  });

  it("treats requests that are not URNs as uncovered", () => {
    expect(uncoveredPermissions([urn("**", "*")], ["api.*.read_api"])).toEqual(["api.*.read_api"]);
  });
});

describe("grantableCatalogues", () => {
  const offers = (catalogue: ScopeCatalogue) =>
    catalogue.groups.flatMap((group) =>
      group.rows.map((row) => `${group.id}/${row.id}:${offeredActions(row).join("+")}`),
    );

  it("keeps the whole catalogue for the admin wildcard", () => {
    expect(grantableCatalogues(CATALOGUES, ws, [urn("**", "*")])).toEqual(CATALOGUES);
  });

  it("keeps only the read actions a viewer holds", () => {
    const viewer = [
      urn("projects/*/keyspaces/*", "read"),
      urn("projects/*/keyspaces/*/logs", "read"),
      urn("projects/*/keyspaces/*/keys/*", "read"),
    ];
    expect(offers(grantableCatalogues(CATALOGUES, ws, viewer).keyspaces)).toEqual([
      "keyspaces/keyspace:read",
      "keyspaces/keyspace_log:read",
      "keyspaces/key:read",
    ]);
  });

  it("drops groups the role cannot reach, such as root keys", () => {
    const developer = [
      ...ACTIONS.map((action) => urn("projects/**", action)),
      urn("flags", "read"),
      urn("flags", "write"),
    ];
    const workspace = grantableCatalogues(CATALOGUES, ws, developer).workspace;
    expect(workspace.groups.map((group) => group.id)).toEqual([
      "projects",
      "apps",
      "environments",
      "deployments",
      "gateway",
      "keyspaces",
      "ratelimits",
      "access",
      "portals",
      "flags",
    ]);
    expect(offers(workspace).slice(-1)).toEqual(["flags/flags:read+write"]);
  });

  it("empties a scope with nothing grantable", () => {
    const catalogues = grantableCatalogues(CATALOGUES, ws, [urn("flags", "read")]);
    expect(catalogues.identities.groups).toEqual([]);
  });

  it("grants nothing from another workspace", () => {
    const catalogues = grantableCatalogues(CATALOGUES, ws, ["unkey:v1:ws_other:**#*"]);
    expect(catalogues.workspace.groups).toEqual([]);
  });
});
