import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { appsCatalogue } from "./catalogue.apps";
import { environmentsCatalogue } from "./catalogue.environments";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { projectsCatalogue } from "./catalogue.projects";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import {
  ACTIONS,
  type PermissionRow,
  type PermissionSelection,
  RESOURCE_SCOPES,
} from "./catalogue.types";
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

const everything = (rows: readonly PermissionRow[]) =>
  rows.reduce<PermissionSelection>(
    (acc, row) => setRowActions(acc, row.id, ["read", "write", "delete"]),
    {},
  );

describe("deploy hierarchy paths", () => {
  it("nests each scope one level deeper than the last", () => {
    expect(rowOf("projects", "project").path).toBe("projects/{instance}");
    expect(rowOf("apps", "app").path).toBe("projects/*/apps/{instance}");
    expect(rowOf("environments", "environment").path).toBe(
      "projects/*/apps/*/environments/{instance}",
    );
  });

  it("wildcards every level below the picked one", () => {
    expect(rowOf("projects", "deployment").path).toBe(
      "projects/{instance}/apps/*/environments/*/deployments/*",
    );
    expect(rowOf("apps", "deployment").path).toBe(
      "projects/*/apps/{instance}/environments/*/deployments/*",
    );
    expect(rowOf("environments", "deployment").path).toBe(
      "projects/*/apps/*/environments/{instance}/deployments/*",
    );
  });

  it("keeps the variables row on the environment itself", () => {
    expect(rowOf("projects", "variable").path).toBe(rowOf("projects", "environment").path);
    expect(rowOf("apps", "variable").path).toBe(rowOf("apps", "environment").path);
    expect(rowOf("environments", "variable").path).toBe(rowOf("environments", "environment").path);
  });

  it("moves each create action to the parent that already exists", () => {
    expect(urnActions(rowOf("projects", "app"), "write")).toEqual([
      { name: "create_app", path: "projects/{instance}" },
      { name: "update_app" },
      { name: "connect_repository" },
    ]);
    expect(urnActions(rowOf("apps", "environment"), "write")).toEqual([
      { name: "create_environment", path: "projects/*/apps/{instance}" },
      { name: "update_environment" },
    ]);
    expect(urnActions(rowOf("environments", "domain"), "write")).toEqual([
      { name: "create_domain", path: "projects/*/apps/*/environments/{instance}" },
      { name: "verify_domain", path: "projects/*/apps/*/environments/{instance}" },
    ]);
  });

  it("leaves create_deployment on the deployment collection", () => {
    expect(urnActions(rowOf("environments", "deployment"), "write")[0]).toEqual({
      name: "create_deployment",
    });
  });

  it("never offers a create action whose only target is a wider scope", () => {
    const names = (scope: (typeof RESOURCE_SCOPES)[number]) =>
      catalogueRows(CATALOGUES[scope]).flatMap((row) =>
        ACTIONS.flatMap((action) => urnActions(row, action).map((grant) => grant.name)),
      );
    expect(names("projects")).not.toContain("create_project");
    expect(names("apps")).not.toContain("create_app");
    expect(names("environments")).not.toContain("create_environment");
  });
});

describe("buildUrns on the projects scope", () => {
  it("hangs every descendant off the project it was picked for", () => {
    const policy = {
      ...newPolicy("projects"),
      instances: ["proj_1"],
      selection: everything(catalogueRows(projectsCatalogue)),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/proj_1#read_project",
      "unkey:v1:ws_123:projects/proj_1#update_project",
      "unkey:v1:ws_123:projects/proj_1#delete_project",
      "unkey:v1:ws_123:projects/proj_1/apps/*#read_app",
      "unkey:v1:ws_123:projects/proj_1#create_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*#update_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*#connect_repository",
      "unkey:v1:ws_123:projects/proj_1/apps/*#delete_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/proj_1/apps/*#create_environment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#update_environment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#delete_environment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#create_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#start_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#stop_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#promote_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#rollback_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#delete_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#create_domain",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#verify_domain",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/domains/*#delete_domain",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#read_environment_variables",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#create_variable",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#set_environment_variables",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#remove_environment_variables",
    ]);
  });

  it("wildcards the project segment for all projects", () => {
    const policy = {
      ...newPolicy("projects"),
      selection: setRowActions(setRowActions({}, "project", ["read"]), "deployment", ["read"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*#read_project",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#read_deployment",
    ]);
  });

  it("repeats every grant for each project picked", () => {
    const policy = {
      ...newPolicy("projects"),
      instances: ["proj_1", "proj_2"],
      selection: setRowActions({}, "app", ["write"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/proj_1#create_app",
      "unkey:v1:ws_123:projects/proj_2#create_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*#update_app",
      "unkey:v1:ws_123:projects/proj_2/apps/*#update_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*#connect_repository",
      "unkey:v1:ws_123:projects/proj_2/apps/*#connect_repository",
    ]);
  });
});

describe("buildUrns on the apps scope", () => {
  it("hangs every descendant off the app it was picked for", () => {
    const policy = {
      ...newPolicy("apps"),
      instances: ["app_1"],
      selection: everything(catalogueRows(appsCatalogue)),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/app_1#read_app",
      "unkey:v1:ws_123:projects/*/apps/app_1#update_app",
      "unkey:v1:ws_123:projects/*/apps/app_1#connect_repository",
      "unkey:v1:ws_123:projects/*/apps/app_1#delete_app",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#read_environment",
      "unkey:v1:ws_123:projects/*/apps/app_1#create_environment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#update_environment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#delete_environment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#create_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#start_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#stop_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#promote_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#rollback_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#delete_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#create_domain",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#verify_domain",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/domains/*#delete_domain",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#read_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#create_variable",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#set_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*#remove_environment_variables",
    ]);
  });

  it("wildcards the app segment for all apps and leaves the project wildcard alone", () => {
    const policy = {
      ...newPolicy("apps"),
      selection: setRowActions({}, "environment", ["read", "write"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/*/apps/*#create_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#update_environment",
    ]);
  });
});

describe("buildUrns on the environments scope", () => {
  it("hangs every descendant off the environment it was picked for", () => {
    const policy = {
      ...newPolicy("environments"),
      instances: ["env_1"],
      selection: everything(catalogueRows(environmentsCatalogue)),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#read_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#update_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#delete_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#create_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#start_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#stop_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#promote_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#rollback_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#delete_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/domains/*#read_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#create_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#verify_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/domains/*#delete_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#read_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#create_variable",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#set_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1#remove_environment_variables",
    ]);
  });

  it("wildcards only the environment segment for all environments", () => {
    const policy = {
      ...newPolicy("environments"),
      selection: setRowActions({}, "variable", ["read"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment_variables",
    ]);
  });

  it("keeps the three deploy scopes apart in one key", () => {
    const project = {
      ...newPolicy("projects"),
      instances: ["proj_1"],
      selection: setRowActions({}, "deployment", ["read"]),
    };
    const app = {
      ...newPolicy("apps"),
      instances: ["app_1"],
      selection: setRowActions({}, "deployment", ["read"]),
    };
    const environment = {
      ...newPolicy("environments"),
      instances: ["env_1"],
      selection: setRowActions({}, "deployment", ["read"]),
    };
    expect(buildUrns(ws, [project, app, environment])).toEqual([
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/app_1/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/env_1/deployments/*#read_deployment",
    ]);
  });
});
