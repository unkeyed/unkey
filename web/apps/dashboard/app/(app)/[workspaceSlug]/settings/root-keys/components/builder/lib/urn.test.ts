import { permissionValidation } from "@unkey/rbac";
import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { appsCatalogue, environmentsCatalogue, projectsCatalogue } from "./catalogue.deploy";
import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import {
  ACTIONS,
  ALL_INSTANCES,
  type Action,
  INSTANCE_TOKEN,
  type PermissionRow,
  RESOURCE_SCOPES,
  type ResourceScope,
  instancePath,
  rowOffers,
} from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import { newPolicy, setRowActions, setRowsActions } from "./policy";
import { buildUrn, buildUrns, rowActionGrants, rowGrants } from "./urn";

const ws = "ws_123";

const rowOf = (scope: (typeof RESOURCE_SCOPES)[number], id: string): PermissionRow => {
  const row = catalogueRows(CATALOGUES[scope]).find((entry) => entry.id === id);
  if (!row) {
    throw new Error(`no row ${id} in ${scope}`);
  }
  return row;
};

const grantNames = (row: PermissionRow, action: Action): string[] =>
  row.actions[action].map((grant) => grant.name);

describe("row actions", () => {
  it("derives the concrete names from the resource noun", () => {
    const row = rowOf("workspace", "identity");
    expect(grantNames(row, "read")).toEqual(["read_identity"]);
    expect(grantNames(row, "write")).toEqual(["create_identity", "update_identity"]);
    expect(grantNames(row, "delete")).toEqual(["delete_identity"]);
  });

  it("keeps the resource noun for every container-less row", () => {
    const rows = [...catalogueRows(identitiesCatalogue), ...catalogueRows(rbacCatalogue)];
    expect(rows.map((row) => grantNames(row, "read"))).toEqual([
      ["read_identity"],
      ["read_role"],
      ["read_permission"],
    ]);
  });

  it("prefers the names a row declares", () => {
    expect(grantNames(rowOf("keyspaces", "key"), "read")).toEqual(["read_key", "verify_key"]);
    expect(grantNames(rowOf("keyspaces", "keyspace"), "write")).toEqual(["update_keyspace"]);
    expect(grantNames(rowOf("ratelimit-namespaces", "override"), "write")).toEqual([
      "set_override",
    ]);
  });

  it("resolves every grant onto the row path unless it declares its own", () => {
    expect(rowOf("keyspaces", "key").actions.write).toEqual([
      { name: "create_key", path: "keyspaces/{instance}" },
      { name: "update_key", path: "keyspaces/{instance}/keys/*" },
    ]);
  });

  it("names the decrypt action only on the rows that declare it", () => {
    expect(grantNames(rowOf("keyspaces", "key"), "decrypt")).toEqual(["decrypt_key"]);
    expect(grantNames(rowOf("workspace", "key"), "decrypt")).toEqual(["decrypt_key"]);
    expect(grantNames(rowOf("keyspaces", "keyspace"), "decrypt")).toEqual([]);
    expect(grantNames(rowOf("workspace", "identity"), "decrypt")).toEqual([]);
  });

  it("names an action for every action a row offers, and none for the rest", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        for (const action of ACTIONS) {
          expect(row.actions[action].length > 0).toBe(rowOffers(row, action));
        }
      }
    }
  });
});

describe("rowOffers", () => {
  it("offers the three coarse actions on every row", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        expect(rowOffers(row, "read")).toBe(true);
        expect(rowOffers(row, "write")).toBe(true);
        expect(rowOffers(row, "delete")).toBe(true);
      }
    }
  });

  it("offers decrypt on the api keys row alone", () => {
    const decrypting = RESOURCE_SCOPES.flatMap((scope) =>
      catalogueRows(CATALOGUES[scope])
        .filter((row) => rowOffers(row, "decrypt"))
        .map((row) => `${scope}:${row.id}`),
    );
    expect(decrypting).toEqual(["workspace:key", "keyspaces:key"]);
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

describe("catalogue grammar", () => {
  it("emits urns the canonical grammar accepts, for one instance and for all", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        for (const action of ACTIONS) {
          for (const instance of ["id_1", ALL_INSTANCES]) {
            for (const grant of rowActionGrants(row, action, instance)) {
              const urn = buildUrn(ws, grant.path, grant.action);
              expect(permissionValidation.safeParse(urn).success, urn).toBe(true);
            }
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

  it("covers every family with wildcards on the workspace scope", () => {
    const all = (scope: ResourceScope) => ({
      ...newPolicy(scope),
      selection: everything(catalogueRows(CATALOGUES[scope])),
    });
    expect(buildUrns(ws, [all("workspace")])).toEqual(
      buildUrns(ws, [
        all("projects"),
        all("keyspaces"),
        all("ratelimit-namespaces"),
        all("identities"),
        all("rbac"),
      ]),
    );
  });

  it("leaves no instance token in the workspace catalogue", () => {
    for (const row of catalogueRows(workspaceCatalogue)) {
      expect(row.path).not.toContain(INSTANCE_TOKEN);
      for (const action of ACTIONS) {
        for (const grant of row.actions[action]) {
          expect(grant.path).not.toContain(INSTANCE_TOKEN);
        }
      }
    }
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
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/ks_2/keys/*#read_key",
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
    const selection = everything(catalogueRows(keyspacesCatalogue));
    expect(buildUrns(ws, [{ ...newPolicy("keyspaces"), instances: ["ks_1"], selection }])).toEqual([
      "unkey:v1:ws_123:keyspaces/ks_1#read_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1#update_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1#delete_keyspace",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/ks_1#create_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#update_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#delete_key",
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#decrypt_key",
    ]);
  });

  it("emits the decrypt grant only when it is ticked", () => {
    const decrypting = {
      ...newPolicy("keyspaces"),
      instances: ["ks_1"],
      selection: setRowActions({}, "key", ["decrypt"]),
    };
    expect(buildUrns(ws, [decrypting])).toEqual([
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#decrypt_key",
    ]);

    const writing = {
      ...newPolicy("keyspaces"),
      instances: ["ks_1"],
      selection: setRowActions({}, "key", ["read", "write", "delete"]),
    };
    expect(buildUrns(ws, [writing])).not.toContain(
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#decrypt_key",
    );
  });

  it("never lets a bulk selection reach an action a row does not offer", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      instances: ["ks_1"],
      selection: setRowsActions({}, catalogueRows(keyspacesCatalogue), ["read", "write", "delete"]),
    };
    expect(policy.selection.keyspace).toEqual(["read", "write", "delete"]);
    expect(policy.selection.key).toEqual(["read", "write", "delete"]);
    expect(buildUrns(ws, [policy])).not.toContain(
      "unkey:v1:ws_123:keyspaces/ks_1/keys/*#decrypt_key",
    );
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

const everything = (rows: readonly PermissionRow[]) => setRowsActions({}, rows, ACTIONS);

describe.each([
  { scope: "projects", rowId: "project", path: "projects/{instance}" },
  { scope: "apps", rowId: "app", path: "projects/*/apps/{instance}" },
  {
    scope: "environments",
    rowId: "environment",
    path: "projects/*/apps/*/environments/{instance}",
  },
] as const)("the $scope level of the deploy tree", ({ scope, rowId, path }) => {
  it("carries the instance token and wildcards every other level", () => {
    expect(rowOf(scope, rowId).path).toBe(path);
  });

  it("hangs the environment rows off the environment below it", () => {
    const environmentPath = rowOf(scope, "environment").path;
    expect(environmentPath.startsWith(path)).toBe(true);
    expect(rowOf(scope, "deployment").path).toBe(`${environmentPath}/deployments/*`);
    expect(rowOf(scope, "domain").path).toBe(`${environmentPath}/domains/*`);
    expect(rowOf(scope, "variable").path).toBe(environmentPath);
  });

  it("never offers a create action whose only target is a wider scope", () => {
    const names = catalogueRows(CATALOGUES[scope]).flatMap((row) =>
      ACTIONS.flatMap((action) => grantNames(row, action)),
    );
    expect(names).not.toContain(`create_${rowId}`);
  });
});

describe("deploy hierarchy grants", () => {
  it("moves each create action to the parent that already exists", () => {
    expect(rowOf("projects", "app").actions.write).toEqual([
      { name: "create_app", path: "projects/{instance}" },
      { name: "update_app", path: "projects/{instance}/apps/*" },
      { name: "connect_repository", path: "projects/{instance}/apps/*" },
    ]);
    expect(rowOf("apps", "environment").actions.write).toEqual([
      { name: "create_environment", path: "projects/*/apps/{instance}" },
      { name: "update_environment", path: "projects/*/apps/{instance}/environments/*" },
    ]);
    expect(rowOf("environments", "domain").actions.write).toEqual([
      { name: "create_domain", path: "projects/*/apps/*/environments/{instance}" },
      { name: "verify_domain", path: "projects/*/apps/*/environments/{instance}" },
    ]);
  });

  it("leaves create_deployment on the deployment collection", () => {
    expect(rowOf("environments", "deployment").actions.write[0]).toEqual({
      name: "create_deployment",
      path: "projects/*/apps/*/environments/{instance}/deployments/*",
    });
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
      "unkey:v1:ws_123:projects/proj_1/apps/*#update_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*#connect_repository",
      "unkey:v1:ws_123:projects/proj_2#create_app",
      "unkey:v1:ws_123:projects/proj_2/apps/*#update_app",
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

describe("rowGrants", () => {
  it("lists every path and action a row emits for the instance picked", () => {
    expect(rowGrants(rowOf("keyspaces", "key"), ["ks_1"])).toEqual([
      { path: "keyspaces/ks_1/keys/*", action: "read_key" },
      { path: "keyspaces/ks_1/keys/*", action: "verify_key" },
      { path: "keyspaces/ks_1", action: "create_key" },
      { path: "keyspaces/ks_1/keys/*", action: "update_key" },
      { path: "keyspaces/ks_1/keys/*", action: "delete_key" },
      { path: "keyspaces/ks_1/keys/*", action: "decrypt_key" },
    ]);
  });

  it("repeats them for every instance picked", () => {
    expect(rowGrants(rowOf("keyspaces", "keyspace"), ["ks_1", "ks_2"])).toEqual([
      { path: "keyspaces/ks_1", action: "read_keyspace" },
      { path: "keyspaces/ks_2", action: "read_keyspace" },
      { path: "keyspaces/ks_1", action: "update_keyspace" },
      { path: "keyspaces/ks_2", action: "update_keyspace" },
      { path: "keyspaces/ks_1", action: "delete_keyspace" },
      { path: "keyspaces/ks_2", action: "delete_keyspace" },
    ]);
  });

  it("wildcards the instance segment for all instances", () => {
    expect(rowGrants(rowOf("projects", "app"), [ALL_INSTANCES])).toEqual([
      { path: "projects/*/apps/*", action: "read_app" },
      { path: "projects/*", action: "create_app" },
      { path: "projects/*/apps/*", action: "update_app" },
      { path: "projects/*/apps/*", action: "connect_repository" },
      { path: "projects/*/apps/*", action: "delete_app" },
    ]);
  });
});
