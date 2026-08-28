import { permissionValidation } from "@unkey/rbac";
import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import {
  ACTIONS,
  ALL_INSTANCES,
  type Action,
  CRUD_ACTIONS,
  INSTANCE_TOKEN,
  type PermissionRow,
  RESOURCE_SCOPES,
  type ResourceScope,
  instancePath,
  resolveInstance,
  rowOffers,
} from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import { newPolicy, setRowActions, setRowsActions } from "./policy";
import { buildUrn, buildUrns, rowActionGrants } from "./urn";

const ws = "ws_123";

const KEYSPACE = "projects/proj_1/keyspaces/ks_1";
const NAMESPACE = "projects/proj_1/ratelimits/namespaces/rlns_1";
const APP = "projects/proj_1/apps/app_1";
const ENVIRONMENT = "projects/proj_1/apps/app_1/environments/env_1";

const NAMED_INSTANCES: Record<ResourceScope, string> = {
  workspace: "*",
  projects: "proj_1",
  apps: APP,
  environments: ENVIRONMENT,
  keyspaces: KEYSPACE,
  "ratelimit-namespaces": NAMESPACE,
  identities: "*",
  rbac: "*",
};

const rowOf = (scope: ResourceScope, id: string): PermissionRow => {
  const row = catalogueRows(CATALOGUES[scope]).find((entry) => entry.id === id);
  if (!row) {
    throw new Error(`no row ${id} in ${scope}`);
  }
  return row;
};

const grantNames = (row: PermissionRow, action: Action): string[] =>
  row.actions[action].map((grant) => grant.name);

const everything = (rows: readonly PermissionRow[]) => setRowsActions({}, rows, ACTIONS);

describe("row actions", () => {
  it("derives one resource-suffixed name per coarse action", () => {
    const row = rowOf("identities", "identity");
    expect(grantNames(row, "read")).toEqual(["read_identity"]);
    expect(grantNames(row, "write")).toEqual(["write_identity"]);
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
    expect(grantNames(rowOf("keyspaces", "key"), "verify")).toEqual(["verify_key"]);
    expect(grantNames(rowOf("keyspaces", "keyspace_log"), "read")).toEqual(["read_keyspace_logs"]);
    expect(grantNames(rowOf("ratelimit-namespaces", "ratelimit_namespace"), "limit")).toEqual([
      "limit_ratelimit_namespace",
    ]);
  });

  it("resolves every grant onto the row path", () => {
    expect(rowOf("keyspaces", "key").actions.write).toEqual([
      { name: "write_key", path: `${INSTANCE_TOKEN}/keys/*` },
    ]);
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
  it("offers read, write and delete on every row that is not a log", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        const mutable = !row.id.endsWith("_log");
        expect(rowOffers(row, "read"), `${scope}:${row.id}:read`).toBe(true);
        expect(rowOffers(row, "write"), `${scope}:${row.id}:write`).toBe(mutable);
        expect(rowOffers(row, "delete"), `${scope}:${row.id}:delete`).toBe(mutable);
      }
    }
  });

  it("reads a log row and nothing else", () => {
    const logs = RESOURCE_SCOPES.flatMap((scope) =>
      catalogueRows(CATALOGUES[scope]).filter((row) => row.id.endsWith("_log")),
    );
    expect(logs.length).toBeGreaterThan(0);
    for (const row of logs) {
      expect(ACTIONS.filter((action) => rowOffers(row, action))).toEqual(["read"]);
    }
  });

  it("offers verify and decrypt on the keys row alone, and limit on the namespace row", () => {
    const offering = (action: Action) =>
      RESOURCE_SCOPES.flatMap((scope) =>
        catalogueRows(CATALOGUES[scope])
          .filter((row) => rowOffers(row, action))
          .map((row) => `${scope}:${row.id}`),
      );
    expect(offering("verify")).toEqual(["workspace:key", "projects:key", "keyspaces:key"]);
    expect(offering("decrypt")).toEqual(["workspace:key", "projects:key", "keyspaces:key"]);
    expect(offering("limit")).toEqual([
      "workspace:ratelimit_namespace",
      "projects:ratelimit_namespace",
      "ratelimit-namespaces:ratelimit_namespace",
    ]);
  });
});

describe("instancePath", () => {
  it("substitutes the whole path down to the instance", () => {
    expect(instancePath(`${INSTANCE_TOKEN}/keys/*`, KEYSPACE)).toBe(`${KEYSPACE}/keys/*`);
  });

  it("substitutes the wildcarded path for all instances", () => {
    expect(
      instancePath(
        `${INSTANCE_TOKEN}/overrides/*`,
        resolveInstance(ratelimitNamespacesCatalogue, ALL_INSTANCES),
      ),
    ).toBe("projects/*/ratelimits/namespaces/*/overrides/*");
  });

  it("leaves a path without the token alone", () => {
    expect(instancePath("projects/*/rbac/roles/*", KEYSPACE)).toBe("projects/*/rbac/roles/*");
  });
});

describe("buildUrn", () => {
  it("composes the versioned urn with the action suffix", () => {
    expect(buildUrn(ws, "projects/*/rbac/roles/*", "write_role")).toBe(
      "unkey:v1:ws_123:projects/*/rbac/roles/*#write_role",
    );
  });

  it("passes a trailing descendant pattern through unchanged", () => {
    expect(buildUrn(ws, "projects/proj_123/**", "delete_deployment")).toBe(
      "unkey:v1:ws_123:projects/proj_123/**#delete_deployment",
    );
  });
});

describe("catalogue grammar", () => {
  it("emits urns the canonical catalog accepts, for one instance and for all", () => {
    for (const scope of RESOURCE_SCOPES) {
      const catalogue = CATALOGUES[scope];
      for (const row of catalogueRows(catalogue)) {
        for (const action of ACTIONS) {
          for (const instance of [
            NAMED_INSTANCES[scope],
            resolveInstance(catalogue, ALL_INSTANCES),
          ]) {
            for (const grant of rowActionGrants(row, action, instance)) {
              const urn = buildUrn("ws_1234abcd", grant.path, grant.action);
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
    const policy = {
      ...newPolicy("rbac"),
      selection: setRowActions({}, "role", ["read", "write"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/rbac/roles/*#read_role",
      "unkey:v1:ws_123:projects/*/rbac/roles/*#write_role",
    ]);
  });

  it("covers every family with wildcards on the workspace scope", () => {
    const all = (scope: ResourceScope) => ({
      ...newPolicy(scope),
      selection: everything(catalogueRows(CATALOGUES[scope])),
    });
    const workspace = buildUrns(ws, [all("workspace")]);
    for (const urn of buildUrns(ws, [all("projects")])) {
      expect(workspace).toContain(urn);
    }
    expect(workspace).toContain("unkey:v1:ws_123:github/apps/*#write_github_app");
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
    const policy = {
      ...newPolicy("identities"),
      selection: setRowActions({}, "identity", ["read"]),
    };
    expect(buildUrns(ws, [policy, policy])).toEqual([
      "unkey:v1:ws_123:projects/*/identities/*#read_identity",
    ]);
  });

  it("carries the workspace id it was given", () => {
    const policy = {
      ...newPolicy("identities"),
      selection: setRowActions({}, "identity", ["delete"]),
    };
    expect(buildUrns("ws_other", [policy])).toEqual([
      "unkey:v1:ws_other:projects/*/identities/*#delete_identity",
    ]);
  });

  it("puts every keyspace grant on the keyspace it was picked for", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      instances: [KEYSPACE],
      selection: setRowActions(setRowActions({}, "keyspace", ["read"]), "key", [...CRUD_ACTIONS]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      `unkey:v1:ws_123:${KEYSPACE}#read_keyspace`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#read_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#write_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#delete_key`,
    ]);
  });

  it("repeats a keyspace grant for every instance picked", () => {
    const second = "projects/proj_2/keyspaces/ks_2";
    const policy = {
      ...newPolicy("keyspaces"),
      instances: [KEYSPACE, second],
      selection: setRowActions({}, "key", ["read"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#read_key`,
      `unkey:v1:ws_123:${second}/keys/*#read_key`,
    ]);
  });

  it("wildcards the whole keyspace path for all keyspaces", () => {
    const policy = { ...newPolicy("keyspaces"), selection: setRowActions({}, "key", ["read"]) };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read_key",
    ]);
  });

  it("covers the keyspace catalogue", () => {
    const selection = everything(catalogueRows(keyspacesCatalogue));
    expect(
      buildUrns(ws, [{ ...newPolicy("keyspaces"), instances: [KEYSPACE], selection }]),
    ).toEqual([
      `unkey:v1:ws_123:${KEYSPACE}#read_keyspace`,
      `unkey:v1:ws_123:${KEYSPACE}#write_keyspace`,
      `unkey:v1:ws_123:${KEYSPACE}#delete_keyspace`,
      `unkey:v1:ws_123:${KEYSPACE}/logs#read_keyspace_logs`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#read_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#write_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#delete_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#verify_key`,
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#decrypt_key`,
    ]);
  });

  it("emits verify and decrypt only when they are ticked", () => {
    const narrow = {
      ...newPolicy("keyspaces"),
      instances: [KEYSPACE],
      selection: setRowActions({}, "key", ["verify"]),
    };
    expect(buildUrns(ws, [narrow])).toEqual([`unkey:v1:ws_123:${KEYSPACE}/keys/*#verify_key`]);

    const writing = {
      ...newPolicy("keyspaces"),
      instances: [KEYSPACE],
      selection: setRowActions({}, "key", [...CRUD_ACTIONS]),
    };
    expect(buildUrns(ws, [writing])).not.toContain(
      `unkey:v1:ws_123:${KEYSPACE}/keys/*#decrypt_key`,
    );
  });

  it("never lets a bulk selection reach an action a row does not offer", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      instances: [KEYSPACE],
      selection: setRowsActions({}, catalogueRows(keyspacesCatalogue), CRUD_ACTIONS),
    };
    expect(policy.selection.keyspace).toEqual([...CRUD_ACTIONS]);
    expect(policy.selection.keyspace_log).toEqual(["read"]);
    expect(buildUrns(ws, [policy])).not.toContain(`unkey:v1:ws_123:${KEYSPACE}/keys/*#decrypt_key`);
  });

  it("covers the ratelimit namespace catalogue", () => {
    const selection = everything(catalogueRows(ratelimitNamespacesCatalogue));
    expect(
      buildUrns(ws, [{ ...newPolicy("ratelimit-namespaces"), instances: [NAMESPACE], selection }]),
    ).toEqual([
      `unkey:v1:ws_123:${NAMESPACE}#read_ratelimit_namespace`,
      `unkey:v1:ws_123:${NAMESPACE}#write_ratelimit_namespace`,
      `unkey:v1:ws_123:${NAMESPACE}#delete_ratelimit_namespace`,
      `unkey:v1:ws_123:${NAMESPACE}#limit_ratelimit_namespace`,
      `unkey:v1:ws_123:${NAMESPACE}/logs#read_ratelimit_logs`,
      `unkey:v1:ws_123:${NAMESPACE}/overrides/*#read_ratelimit_override`,
      `unkey:v1:ws_123:${NAMESPACE}/overrides/*#write_ratelimit_override`,
      `unkey:v1:ws_123:${NAMESPACE}/overrides/*#delete_ratelimit_override`,
    ]);
  });

  it("keeps two scopes in one key apart", () => {
    const keyspace = {
      ...newPolicy("keyspaces"),
      selection: setRowActions({}, "key", ["read"]),
    };
    const namespace = {
      ...newPolicy("ratelimit-namespaces"),
      selection: setRowActions({}, "ratelimit_override", ["read"]),
    };
    expect(buildUrns(ws, [keyspace, namespace])).toEqual([
      "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
    ]);
  });
});

describe.each([
  { scope: "projects", rowId: "project", path: `projects/${INSTANCE_TOKEN}` },
  { scope: "apps", rowId: "app", path: INSTANCE_TOKEN },
  { scope: "environments", rowId: "environment", path: INSTANCE_TOKEN },
] as const)("the $scope level of the deploy tree", ({ scope, rowId, path }) => {
  it("anchors its own level on the instance token", () => {
    expect(rowOf(scope, rowId).path).toBe(path);
  });

  it("hangs the environment rows off the environment below it", () => {
    const environmentPath = rowOf(scope, "environment").path;
    expect(environmentPath.startsWith(path)).toBe(true);
    expect(rowOf(scope, "deployment").path).toBe(`${environmentPath}/deployments/*`);
    expect(rowOf(scope, "domain").path).toBe(`${environmentPath}/domains/*`);
    expect(rowOf(scope, "variable").path).toBe(`${environmentPath}/variables/*`);
    expect(rowOf(scope, "gateway_policy").path).toBe(`${environmentPath}/gateway/policies/*`);
  });

  it("never lifts a grant off the row it belongs to", () => {
    for (const row of catalogueRows(CATALOGUES[scope])) {
      for (const action of ACTIONS) {
        for (const grant of row.actions[action]) {
          expect(grant.path).toBe(row.path);
        }
      }
    }
  });
});

describe("buildUrns on the projects scope", () => {
  it("hangs every descendant off the project it was picked for", () => {
    const policy = {
      ...newPolicy("projects"),
      instances: ["proj_1"],
      selection: catalogueRows(CATALOGUES.projects).reduce(
        (acc, row) => setRowActions(acc, row.id, ["read"]),
        {},
      ),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/proj_1#read_project",
      "unkey:v1:ws_123:projects/proj_1/apps/*#read_app",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/variables/*#read_environment_variable",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*/logs#read_deployment_logs",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/gateway/logs#read_gateway_logs",
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/gateway/policies/*#read_gateway_policy",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/*#read_keyspace",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/*/logs#read_keyspace_logs",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/*#read_ratelimit_namespace",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/*/logs#read_ratelimit_logs",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
      "unkey:v1:ws_123:projects/proj_1/identities/*#read_identity",
      "unkey:v1:ws_123:projects/proj_1/rbac/roles/*#read_role",
      "unkey:v1:ws_123:projects/proj_1/rbac/permissions/*#read_permission",
    ]);
  });

  it("wildcards the project segment for all projects", () => {
    const policy = {
      ...newPolicy("projects"),
      selection: setRowActions({}, "deployment", ["write"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#write_deployment",
    ]);
  });

  it("repeats every grant for each project picked", () => {
    const policy = {
      ...newPolicy("projects"),
      instances: ["proj_1", "proj_2"],
      selection: setRowActions({}, "key", ["decrypt"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/proj_1/keyspaces/*/keys/*#decrypt_key",
      "unkey:v1:ws_123:projects/proj_2/keyspaces/*/keys/*#decrypt_key",
    ]);
  });
});

describe("buildUrns on the apps scope", () => {
  it("hangs every descendant off the app it was picked for", () => {
    const policy = {
      ...newPolicy("apps"),
      instances: [APP],
      selection: catalogueRows(CATALOGUES.apps).reduce(
        (acc, row) => setRowActions(acc, row.id, ["read"]),
        {},
      ),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      `unkey:v1:ws_123:${APP}#read_app`,
      `unkey:v1:ws_123:${APP}/environments/*#read_environment`,
      `unkey:v1:ws_123:${APP}/environments/*/variables/*#read_environment_variable`,
      `unkey:v1:ws_123:${APP}/environments/*/domains/*#read_domain`,
      `unkey:v1:ws_123:${APP}/environments/*/deployments/*#read_deployment`,
      `unkey:v1:ws_123:${APP}/environments/*/deployments/*/logs#read_deployment_logs`,
      `unkey:v1:ws_123:${APP}/environments/*/gateway/logs#read_gateway_logs`,
      `unkey:v1:ws_123:${APP}/environments/*/gateway/policies/*#read_gateway_policy`,
    ]);
  });

  it("wildcards project and app together for all apps", () => {
    const policy = { ...newPolicy("apps"), selection: setRowActions({}, "app", ["read"]) };
    expect(buildUrns(ws, [policy])).toEqual(["unkey:v1:ws_123:projects/*/apps/*#read_app"]);
  });

  it("offers nothing that lives beside the app rather than under it", () => {
    const ids = catalogueRows(CATALOGUES.apps).map((row) => row.id);
    expect(ids).not.toContain("keyspace");
    expect(ids).not.toContain("identity");
  });
});

describe("buildUrns on the environments scope", () => {
  it("hangs every descendant off the environment it was picked for", () => {
    const policy = {
      ...newPolicy("environments"),
      instances: [ENVIRONMENT],
      selection: setRowActions(setRowActions({}, "variable", ["read"]), "deployment", ["write"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      `unkey:v1:ws_123:${ENVIRONMENT}/variables/*#read_environment_variable`,
      `unkey:v1:ws_123:${ENVIRONMENT}/deployments/*#write_deployment`,
    ]);
  });

  it("wildcards the whole path for all environments", () => {
    const policy = {
      ...newPolicy("environments"),
      selection: setRowActions({}, "variable", ["read"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/variables/*#read_environment_variable",
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
      instances: [APP],
      selection: setRowActions({}, "deployment", ["read"]),
    };
    const environment = {
      ...newPolicy("environments"),
      instances: [ENVIRONMENT],
      selection: setRowActions({}, "deployment", ["read"]),
    };
    expect(buildUrns(ws, [project, app, environment])).toEqual([
      "unkey:v1:ws_123:projects/proj_1/apps/*/environments/*/deployments/*#read_deployment",
      `unkey:v1:ws_123:${APP}/environments/*/deployments/*#read_deployment`,
      `unkey:v1:ws_123:${ENVIRONMENT}/deployments/*#read_deployment`,
    ]);
  });
});
