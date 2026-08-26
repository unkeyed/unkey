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
  WORKOS_PERMISSION_SLUGS,
} from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import { ALL_INSTANCES, newPolicy, setRowActions, supportedRowActions } from "./policy";
import {
  buildUrn,
  buildUrns,
  grantPaths,
  instancePath,
  isValidResourcePath,
  rowGrants,
  urnActions,
} from "./urn";

const ws = "ws_123";

const rowOf = (catalogue: typeof workspaceCatalogue, id: string): PermissionRow => {
  const row = catalogueRows(catalogue).find((entry) => entry.id === id);
  if (!row) {
    throw new Error(`no row ${id} in ${catalogue.scope}`);
  }
  return row;
};

const everything = (rows: readonly PermissionRow[]) =>
  rows.reduce<PermissionSelection>(
    (acc, row) => setRowActions(acc, row.id, supportedRowActions(row)),
    {},
  );

const workspaceGrantMappings = (): string[] =>
  catalogueRows(workspaceCatalogue)
    .flatMap((row) =>
      supportedRowActions(row).flatMap((action) =>
        urnActions(row, action).map(
          (grant) =>
            `${grant.slug} ${instancePath(grant.path ?? row.path, ALL_INSTANCES)}#${grant.name}`,
        ),
      ),
    )
    .sort();

const sampleInstanceByScope: Record<(typeof RESOURCE_SCOPES)[number], string> = {
  workspace: "ignored",
  projects: "proj_1",
  apps: "projects/proj_1/apps/app_1",
  environments: "projects/proj_1/apps/app_1/environments/env_1",
  keyspaces: "projects/proj_1/keyspaces/ks_1",
  "ratelimit-namespaces": "projects/proj_1/ratelimits/namespaces/rlns_1",
  identities: "ignored",
  rbac: "ignored",
};

const projectCatalogUrns = (projectId: string): string[] => [
  `unkey:v1:${ws}:projects/${projectId}#read_project`,
  `unkey:v1:${ws}:projects/${projectId}#write_project`,
  `unkey:v1:${ws}:projects/${projectId}#delete_project`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*#read_app`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*#write_app`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*#delete_app`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*#read_environment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*#write_environment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*#delete_environment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/variables/*#read_environment_variable`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/variables/*#write_environment_variable`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/variables/*#delete_environment_variable`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/domains/*#read_domain`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/domains/*#write_domain`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/domains/*#delete_domain`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*#read_deployment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*#write_deployment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*#delete_deployment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*#start_deployment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*#stop_deployment`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/deployments/*/logs#read_deployment_logs`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/gateway/logs#read_gateway_logs`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/gateway/policies/*#read_gateway_policy`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/gateway/policies/*#write_gateway_policy`,
  `unkey:v1:${ws}:projects/${projectId}/apps/*/environments/*/gateway/policies/*#delete_gateway_policy`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*#read_keyspace`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*#write_keyspace`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*#delete_keyspace`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/logs#read_keyspace_logs`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/keys/*#read_key`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/keys/*#write_key`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/keys/*#delete_key`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/keys/*#decrypt_key`,
  `unkey:v1:${ws}:projects/${projectId}/keyspaces/*/keys/*#verify_key`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*#read_ratelimit_namespace`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*#write_ratelimit_namespace`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*#delete_ratelimit_namespace`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*#limit_ratelimit_namespace`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*/logs#read_ratelimit_logs`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*/overrides/*#read_ratelimit_override`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*/overrides/*#write_ratelimit_override`,
  `unkey:v1:${ws}:projects/${projectId}/ratelimits/namespaces/*/overrides/*#delete_ratelimit_override`,
  `unkey:v1:${ws}:projects/${projectId}/identities/*#read_identity`,
  `unkey:v1:${ws}:projects/${projectId}/identities/*#write_identity`,
  `unkey:v1:${ws}:projects/${projectId}/identities/*#delete_identity`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/roles/*#read_role`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/roles/*#write_role`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/roles/*#delete_role`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/permissions/*#read_permission`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/permissions/*#write_permission`,
  `unkey:v1:${ws}:projects/${projectId}/rbac/permissions/*#delete_permission`,
];

describe("urnActions", () => {
  it("maps every non-admin WorkOS slug to its canonical resource action", () => {
    expect(WORKOS_PERMISSION_SLUGS).toHaveLength(55);
    expect(workspaceGrantMappings()).toEqual([
      "apps:delete projects/*/apps/*#delete_app",
      "apps:read projects/*/apps/*#read_app",
      "apps:write projects/*/apps/*#write_app",
      "deployment_logs:read projects/*/apps/*/environments/*/deployments/*/logs#read_deployment_logs",
      "deployments:delete projects/*/apps/*/environments/*/deployments/*#delete_deployment",
      "deployments:read projects/*/apps/*/environments/*/deployments/*#read_deployment",
      "deployments:start projects/*/apps/*/environments/*/deployments/*#start_deployment",
      "deployments:stop projects/*/apps/*/environments/*/deployments/*#stop_deployment",
      "deployments:write projects/*/apps/*/environments/*/deployments/*#write_deployment",
      "domains:delete projects/*/apps/*/environments/*/domains/*#delete_domain",
      "domains:read projects/*/apps/*/environments/*/domains/*#read_domain",
      "domains:write projects/*/apps/*/environments/*/domains/*#write_domain",
      "environment_variables:delete projects/*/apps/*/environments/*/variables/*#delete_environment_variable",
      "environment_variables:read projects/*/apps/*/environments/*/variables/*#read_environment_variable",
      "environment_variables:write projects/*/apps/*/environments/*/variables/*#write_environment_variable",
      "environments:delete projects/*/apps/*/environments/*#delete_environment",
      "environments:read projects/*/apps/*/environments/*#read_environment",
      "environments:write projects/*/apps/*/environments/*#write_environment",
      "gateway_logs:read projects/*/apps/*/environments/*/gateway/logs#read_gateway_logs",
      "gateway_policies:delete projects/*/apps/*/environments/*/gateway/policies/*#delete_gateway_policy",
      "gateway_policies:read projects/*/apps/*/environments/*/gateway/policies/*#read_gateway_policy",
      "gateway_policies:write projects/*/apps/*/environments/*/gateway/policies/*#write_gateway_policy",
      "github_apps:delete github/apps/*#delete_github_app",
      "github_apps:read github/apps/*#read_github_app",
      "github_apps:write github/apps/*#write_github_app",
      "identities:delete projects/*/identities/*#delete_identity",
      "identities:read projects/*/identities/*#read_identity",
      "identities:write projects/*/identities/*#write_identity",
      "keys:decrypt projects/*/keyspaces/*/keys/*#decrypt_key",
      "keys:delete projects/*/keyspaces/*/keys/*#delete_key",
      "keys:read projects/*/keyspaces/*/keys/*#read_key",
      "keys:verify projects/*/keyspaces/*/keys/*#verify_key",
      "keys:write projects/*/keyspaces/*/keys/*#write_key",
      "keyspace_logs:read projects/*/keyspaces/*/logs#read_keyspace_logs",
      "keyspaces:delete projects/*/keyspaces/*#delete_keyspace",
      "keyspaces:read projects/*/keyspaces/*#read_keyspace",
      "keyspaces:write projects/*/keyspaces/*#write_keyspace",
      "permissions:delete projects/*/rbac/permissions/*#delete_permission",
      "permissions:read projects/*/rbac/permissions/*#read_permission",
      "permissions:write projects/*/rbac/permissions/*#write_permission",
      "projects:delete projects/*#delete_project",
      "projects:read projects/*#read_project",
      "projects:write projects/*#write_project",
      "ratelimit_logs:read projects/*/ratelimits/namespaces/*/logs#read_ratelimit_logs",
      "ratelimit_namespaces:delete projects/*/ratelimits/namespaces/*#delete_ratelimit_namespace",
      "ratelimit_namespaces:limit projects/*/ratelimits/namespaces/*#limit_ratelimit_namespace",
      "ratelimit_namespaces:read projects/*/ratelimits/namespaces/*#read_ratelimit_namespace",
      "ratelimit_namespaces:write projects/*/ratelimits/namespaces/*#write_ratelimit_namespace",
      "ratelimit_overrides:delete projects/*/ratelimits/namespaces/*/overrides/*#delete_ratelimit_override",
      "ratelimit_overrides:read projects/*/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
      "ratelimit_overrides:write projects/*/ratelimits/namespaces/*/overrides/*#write_ratelimit_override",
      "roles:delete projects/*/rbac/roles/*#delete_role",
      "roles:read projects/*/rbac/roles/*#read_role",
      "roles:write projects/*/rbac/roles/*#write_role",
    ]);
  });

  it("uses resource-qualified action names and drops unsupported pairs", () => {
    const key = rowOf(keyspacesCatalogue, "key");
    const logs = rowOf(keyspacesCatalogue, "keyspace_log");

    expect(urnActions(key, "read_key")).toEqual([{ name: "read_key", slug: "keys:read" }]);
    expect(urnActions(key, "decrypt_key")).toEqual([{ name: "decrypt_key", slug: "keys:decrypt" }]);
    expect(urnActions(logs, "write_keyspace")).toEqual([]);
  });

  it("offers only the actions each row supports", () => {
    expect(supportedRowActions(rowOf(projectsCatalogue, "deployment"))).toEqual([
      "read_deployment",
      "write_deployment",
      "delete_deployment",
      "start_deployment",
      "stop_deployment",
    ]);
    expect(supportedRowActions(rowOf(projectsCatalogue, "key"))).toEqual([
      "read_key",
      "write_key",
      "delete_key",
      "decrypt_key",
      "verify_key",
    ]);
    expect(supportedRowActions(rowOf(projectsCatalogue, "ratelimit_namespace"))).toEqual([
      "read_ratelimit_namespace",
      "write_ratelimit_namespace",
      "delete_ratelimit_namespace",
      "limit_ratelimit_namespace",
    ]);
    expect(supportedRowActions(rowOf(projectsCatalogue, "gateway_log"))).toEqual([
      "read_gateway_logs",
    ]);
  });
});

describe("instancePath", () => {
  it("substitutes a concrete instance id", () => {
    expect(instancePath("{instance}/keys/*", "projects/proj_1/keyspaces/ks_1")).toBe(
      "projects/proj_1/keyspaces/ks_1/keys/*",
    );
  });

  it("substitutes a single-segment wildcard for all instances", () => {
    expect(instancePath("projects/*/ratelimits/namespaces/{instance}", ALL_INSTANCES)).toBe(
      "projects/*/ratelimits/namespaces/*",
    );
  });

  it("uses allPath when all instances need more than one path segment", () => {
    expect(instancePath("{instance}/keys/*", ALL_INSTANCES, "projects/*/keyspaces/*/keys/*")).toBe(
      "projects/*/keyspaces/*/keys/*",
    );
  });
});

describe("buildUrn", () => {
  it("composes the versioned urn with the action suffix", () => {
    expect(buildUrn(ws, "projects/*/rbac/roles/*", "write_role")).toBe(
      "unkey:v1:ws_123:projects/*/rbac/roles/*#write_role",
    );
  });
});

describe("isValidResourcePath", () => {
  it("accepts single-segment wildcards anywhere", () => {
    expect(isValidResourcePath("projects/*/keyspaces/*/keys/*")).toBe(true);
  });

  it("accepts a descendant wildcard only in trailing position", () => {
    expect(isValidResourcePath("projects/proj_123/**")).toBe(true);
    expect(isValidResourcePath("projects/proj_123/**/deployments/*")).toBe(false);
  });

  it("rejects empty and partial-segment wildcards", () => {
    expect(isValidResourcePath("")).toBe(false);
    expect(isValidResourcePath("projects/proj_123/")).toBe(false);
    expect(isValidResourcePath("projects/proj_*")).toBe(false);
  });

  it("holds for every catalogue path, for one instance and for all", () => {
    for (const scope of RESOURCE_SCOPES) {
      for (const row of catalogueRows(CATALOGUES[scope])) {
        for (const action of ACTIONS) {
          for (const grant of urnActions(row, action)) {
            expect(
              isValidResourcePath(
                instancePath(grant.path ?? row.path, sampleInstanceByScope[scope], row.allPath),
              ),
            ).toBe(true);
            expect(
              isValidResourcePath(instancePath(grant.path ?? row.path, ALL_INSTANCES, row.allPath)),
            ).toBe(true);
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

  it("deduplicates urns shared by two policies", () => {
    const policy = {
      ...newPolicy("identities"),
      selection: setRowActions({}, "identity", ["read_identity"]),
    };
    expect(buildUrns(ws, [policy, policy])).toEqual([
      "unkey:v1:ws_123:projects/*/identities/*#read_identity",
    ]);
  });

  it("builds the complete workspace catalog with wildcards", () => {
    const policy = {
      ...newPolicy(),
      selection: everything(catalogueRows(workspaceCatalogue)),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      ...projectCatalogUrns("*"),
      "unkey:v1:ws_123:github/apps/*#read_github_app",
      "unkey:v1:ws_123:github/apps/*#write_github_app",
      "unkey:v1:ws_123:github/apps/*#delete_github_app",
    ]);
  });

  it("builds the complete project-scoped catalog for a picked project", () => {
    const policy = {
      ...newPolicy("projects"),
      instances: ["proj_1"],
      selection: everything(catalogueRows(projectsCatalogue)),
    };
    expect(buildUrns(ws, [policy])).toEqual(projectCatalogUrns("proj_1"));
  });

  it("repeats app grants for each app picked", () => {
    const policy = {
      ...newPolicy("apps"),
      instances: ["projects/proj_1/apps/app_1", "projects/proj_2/apps/app_2"],
      selection: setRowActions({}, "deployment", ["start_deployment"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/proj_1/apps/app_1/environments/*/deployments/*#start_deployment",
      "unkey:v1:ws_123:projects/proj_2/apps/app_2/environments/*/deployments/*#start_deployment",
    ]);
  });

  it("wildcards only the selected environment segment when all environments are picked", () => {
    const policy = {
      ...newPolicy("environments"),
      selection: setRowActions({}, "gateway_policy", ["write_gateway_policy"]),
    };
    expect(buildUrns(ws, [policy])).toEqual([
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#write_gateway_policy",
    ]);
  });

  it("builds keyspace and ratelimit scoped catalogs under projects", () => {
    expect(
      buildUrns(ws, [
        {
          ...newPolicy("keyspaces"),
          instances: ["projects/proj_1/keyspaces/ks_1"],
          selection: everything(catalogueRows(keyspacesCatalogue)),
        },
      ]),
    ).toEqual([
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1#read_keyspace",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1#write_keyspace",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1#delete_keyspace",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/logs#read_keyspace_logs",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/*#read_key",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/*#write_key",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/*#delete_key",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/*#decrypt_key",
      "unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/*#verify_key",
    ]);

    expect(
      buildUrns(ws, [
        {
          ...newPolicy("ratelimit-namespaces"),
          instances: ["projects/proj_1/ratelimits/namespaces/rlns_1"],
          selection: everything(catalogueRows(ratelimitNamespacesCatalogue)),
        },
      ]),
    ).toEqual([
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1#read_ratelimit_namespace",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1#write_ratelimit_namespace",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1#delete_ratelimit_namespace",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1#limit_ratelimit_namespace",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1/logs#read_ratelimit_logs",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1/overrides/*#read_ratelimit_override",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1/overrides/*#write_ratelimit_override",
      "unkey:v1:ws_123:projects/proj_1/ratelimits/namespaces/rlns_1/overrides/*#delete_ratelimit_override",
    ]);
  });
});

describe("deploy hierarchy paths", () => {
  it("nests each scope one level deeper than the last", () => {
    expect(rowOf(projectsCatalogue, "project").path).toBe("projects/{instance}");
    expect(rowOf(appsCatalogue, "app").path).toBe("{instance}");
    expect(rowOf(appsCatalogue, "app").allPath).toBe("projects/*/apps/*");
    expect(rowOf(environmentsCatalogue, "environment").path).toBe("{instance}");
    expect(rowOf(environmentsCatalogue, "environment").allPath).toBe(
      "projects/*/apps/*/environments/*",
    );
  });

  it("wildcards every level below the picked one", () => {
    expect(rowOf(projectsCatalogue, "deployment").path).toBe(
      "projects/{instance}/apps/*/environments/*/deployments/*",
    );
    expect(rowOf(appsCatalogue, "deployment").path).toBe("{instance}/environments/*/deployments/*");
    expect(rowOf(environmentsCatalogue, "deployment").path).toBe("{instance}/deployments/*");
  });

  it("treats logs as first-class resources", () => {
    expect(rowOf(projectsCatalogue, "deployment_log").path).toBe(
      "projects/{instance}/apps/*/environments/*/deployments/*/logs",
    );
    expect(rowOf(projectsCatalogue, "gateway_log").path).toBe(
      "projects/{instance}/apps/*/environments/*/gateway/logs",
    );
    expect(urnActions(rowOf(projectsCatalogue, "deployment_log"), "write_deployment")).toEqual([]);
  });
});

describe("rowGrants", () => {
  it("lists every path and action a row emits for the instance picked", () => {
    expect(rowGrants(rowOf(keyspacesCatalogue, "key"), ["projects/proj_1/keyspaces/ks_1"])).toEqual(
      [
        "projects/proj_1/keyspaces/ks_1/keys/*#read_key",
        "projects/proj_1/keyspaces/ks_1/keys/*#write_key",
        "projects/proj_1/keyspaces/ks_1/keys/*#delete_key",
        "projects/proj_1/keyspaces/ks_1/keys/*#decrypt_key",
        "projects/proj_1/keyspaces/ks_1/keys/*#verify_key",
      ],
    );
  });

  it("wildcards the instance segment for all instances", () => {
    expect(rowGrants(rowOf(projectsCatalogue, "app"), [ALL_INSTANCES])).toEqual([
      "projects/*/apps/*#read_app",
      "projects/*/apps/*#write_app",
      "projects/*/apps/*#delete_app",
    ]);
  });
});

describe("grantPaths", () => {
  it("keeps each distinct path once, without its actions", () => {
    expect(
      grantPaths(rowGrants(rowOf(keyspacesCatalogue, "key"), ["projects/proj_1/keyspaces/ks_1"])),
    ).toEqual(["projects/proj_1/keyspaces/ks_1/keys/*"]);
  });
});
