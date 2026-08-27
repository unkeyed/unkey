import { describe, expect, test } from "vitest";
import {
  buildIdSchema,
  unkeyPermissionValidation,
  workosPermissionDefinitions,
} from "./permissions";

const ws = "ws_12345678";

const idByParameter: Record<string, string> = {
  app_id: "app_12345678",
  deployment_id: "deploy_12345678",
  domain_id: "domain_12345678",
  environment_id: "env_12345678",
  github_app_id: "123456",
  identity_id: "id_12345678",
  key_id: "key_12345678",
  keyspace_id: "ks_12345678",
  namespace_id: "rlns_12345678",
  override_id: "ro_12345678",
  permission_id: "perm_12345678",
  policy_id: "pol_12345678",
  project_id: "proj_12345678",
  role_id: "role_12345678",
  variable_id: "var_12345678",
};

const concretePath = (path: string): string =>
  path.replace(/\{([^}]+)\}/g, (_match: string, parameter: string) => {
    const id = idByParameter[parameter];
    if (id === undefined) {
      throw new Error(`no test id for ${parameter}`);
    }
    return id;
  });

const canonicalPermissions = workosPermissionDefinitions.map(
  (definition) => `unkey:v1:${ws}:${concretePath(definition.path)}#${definition.action}`,
);

describe("apiIdSchema", () => {
  const testCases = [
    { input: "123456789012", valid: false },
    { input: "a1234asfas12", valid: false },
    { input: "api_123456789ABCDEFGHJKLMNPQRS", valid: true },
    { input: "api_0OIl0OIl", valid: true },
    { input: "*", valid: true },
  ];

  for (const { input, valid } of testCases) {
    test(`parsing ${input} should be ${valid ? "valid" : "invalid"}`, () => {
      const result = buildIdSchema("api").safeParse(input);
      expect(result.success).toBe(valid);
    });
  }
});

describe("unkeyPermissionValidation", () => {
  test("contains the exact WorkOS permission slugs", () => {
    expect(workosPermissionDefinitions).toHaveLength(53);
    expect(workosPermissionDefinitions.map((definition) => definition.slug)).toEqual([
      "admin:*",
      "apps:delete",
      "apps:read",
      "apps:write",
      "deployment_logs:read",
      "deployments:delete",
      "deployments:read",
      "deployments:write",
      "domains:delete",
      "domains:read",
      "domains:write",
      "environment_variables:delete",
      "environment_variables:read",
      "environment_variables:write",
      "environments:delete",
      "environments:read",
      "environments:write",
      "gateway_logs:read",
      "gateway_policies:delete",
      "gateway_policies:read",
      "gateway_policies:write",
      "github_apps:delete",
      "github_apps:read",
      "github_apps:write",
      "identities:delete",
      "identities:read",
      "identities:write",
      "keys:decrypt",
      "keys:delete",
      "keys:read",
      "keys:verify",
      "keys:write",
      "keyspace_logs:read",
      "keyspaces:delete",
      "keyspaces:read",
      "keyspaces:write",
      "permissions:delete",
      "permissions:read",
      "permissions:write",
      "projects:delete",
      "projects:read",
      "projects:write",
      "ratelimit_logs:read",
      "ratelimit_namespaces:delete",
      "ratelimit_namespaces:limit",
      "ratelimit_namespaces:read",
      "ratelimit_namespaces:write",
      "ratelimit_overrides:delete",
      "ratelimit_overrides:read",
      "ratelimit_overrides:write",
      "roles:delete",
      "roles:read",
      "roles:write",
    ]);
  });

  test("accepts every canonical permission in the WorkOS catalog", () => {
    for (const permission of canonicalPermissions) {
      expect(unkeyPermissionValidation.safeParse(permission).success).toBe(true);
    }
  });

  test("accepts single-segment wildcards in catalog resource ids", () => {
    expect(
      unkeyPermissionValidation.safeParse(
        `unkey:v1:${ws}:projects/*/apps/*/environments/*/deployments/*#write_deployment`,
      ).success,
    ).toBe(true);
  });

  test("accepts the canonical admin grant", () => {
    expect(unkeyPermissionValidation.safeParse(`unkey:v1:${ws}:**#*`).success).toBe(true);
  });

  test("rejects unsupported resources and action pairs", () => {
    const invalid = [
      "*",
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/gateway#read_gateway_logs`,
      `unkey:v1:${ws}:projects/proj_12345678/rbac#write_role`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/logs#write_keyspace`,
      `unkey:v1:${ws}:projects/proj_12345678/ratelimits/namespaces/rlns_12345678/logs#delete_ratelimit_namespace`,
      `unkey:v1:${ws}:projects/*/apps/app_12345678#read_app`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/*/environments/env_12345678#read_environment`,
      `unkey:v1:${ws}:projects/*/keyspaces/ks_12345678#read_keyspace`,
      `unkey:v1:${ws}:projects/*/ratelimits/namespaces/rlns_12345678#read_ratelimit_namespace`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#promote_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#rollback_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#release`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#start_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#stop_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678#promote_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/keys/key_12345678#encrypt_key`,
      `unkey:v1:${ws}:projects/proj_12345678/github/apps/123456#read_github_app`,
      `unkey:v1:${ws}:identities/id_12345678#read_identity`,
      `unkey:v1:${ws}:vault/keys/key_12345678#read_key`,
      `unkey:v1:${ws}:ratelimit.rlns_12345678.read_analytics`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/keys/key_12345678#create_key`,
      `unkey:v1:${ws}:projects/proj_*/keyspaces/ks_12345678#read_keyspace`,
      `unkey:v1:${ws}:projects/proj_12345678/**#read_keyspace`,
      `unkey:v1:${ws}:projects/proj_12345678#read`,
    ];

    for (const permission of invalid) {
      expect(unkeyPermissionValidation.safeParse(permission).success).toBe(false);
    }
  });
});
