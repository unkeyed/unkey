import { describe, expect, test } from "vitest";
import {
  PERMISSION_MAX_LENGTH,
  buildIdSchema,
  permissionValidation,
  portalActions,
  ratelimitActions,
  unkeyPermissionValidation,
  urnPermissionWorkspaceId,
  workosPermissionDefinitions,
} from "./permissions";
import urnGrammarCases from "./urn-grammar.fixture.json";

function firstError(input: unknown): string | undefined {
  const result = permissionValidation.safeParse(input);
  return result.success ? undefined : result.error.issues[0]?.message;
}

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

function concretePath(path: string): string {
  return path.replace(/\{([^}]+)\}/g, (_match, parameter: string) => {
    const id = idByParameter[parameter];
    if (id === undefined) {
      throw new Error(`no test id for ${parameter}`);
    }
    return id;
  });
}

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

describe("ratelimit permissions", () => {
  test("includes read analytics", () => {
    expect(ratelimitActions.safeParse("read_analytics").success).toBe(true);
    expect(unkeyPermissionValidation.safeParse("ratelimit.*.read_analytics").success).toBe(true);
  });

  test("requires the rlns namespace ID prefix", () => {
    expect(
      unkeyPermissionValidation.safeParse("ratelimit.rlns_12345678.read_analytics").success,
    ).toBe(true);
    expect(
      unkeyPermissionValidation.safeParse("ratelimit.rl_12345678.read_analytics").success,
    ).toBe(false);
  });
});

describe("portal permissions", () => {
  const testCases = [
    { input: "portal.pc_1234abcd.create_portal", valid: true },
    { input: "portal.pc_1234abcd.read_portal", valid: true },
    { input: "portal.pc_1234abcd.update_portal", valid: true },
    { input: "portal.pc_1234abcd.delete_portal", valid: true },
    { input: "portal.pc_1234abcd.create_portal_session", valid: true },
    { input: "portal.*.read_portal", valid: true },
    // action is not part of the enum
    { input: "portal.pc_1234abcd.mint_session", valid: false },
    // id does not carry the pc_ prefix
    { input: "portal.badid.read_portal", valid: false },
    // legacy tuples must have exactly three parts
    { input: "portal.create_portal", valid: false },
  ];

  for (const { input, valid } of testCases) {
    test(`${input} should be ${valid ? "valid" : "invalid"}`, () => {
      expect(unkeyPermissionValidation.safeParse(input).success).toBe(valid);
    });
  }

  test("exposes every portal action", () => {
    expect(portalActions.options).toStrictEqual([
      "create_portal",
      "read_portal",
      "update_portal",
      "delete_portal",
      "create_portal_session",
    ]);
  });
});

describe("legacy permission validation", () => {
  test("does not throw on non-string input", () => {
    expect(() => unkeyPermissionValidation.safeParse(42)).not.toThrow();
    expect(unkeyPermissionValidation.safeParse(42).success).toBe(false);
    expect(unkeyPermissionValidation.safeParse(undefined).success).toBe(false);
  });

  test("accepts the legacy wildcard", () => {
    expect(unkeyPermissionValidation.safeParse("*").success).toBe(true);
  });
});

// The fixture states the grammar alone, in step with pkg/urn. Whether an action
// belongs on a path is the catalog's business, so these cases drive the grammar
// parser rather than `permissionValidation`, which asks both questions at once.
describe("urn grammar", () => {
  for (const { input, valid, reason } of urnGrammarCases) {
    test(`${valid ? "accepts" : "rejects"} ${input} (${reason})`, () => {
      expect(urnPermissionWorkspaceId(input) !== null).toBe(valid);
    });
  }

  test("rejects non-string input", () => {
    expect(permissionValidation.safeParse(42).success).toBe(false);
  });
});

describe("permission catalog", () => {
  test("lists the WorkOS slugs the model grants", () => {
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

  test("accepts every canonical permission", () => {
    for (const permission of canonicalPermissions) {
      expect(permissionValidation.safeParse(permission).success).toBe(true);
    }
  });

  test("accepts wildcards from the leaf upwards", () => {
    const accepted = [
      `unkey:v1:${ws}:projects/*#read_project`,
      `unkey:v1:${ws}:projects/*/apps/*#read_app`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/*#read_app`,
      `unkey:v1:${ws}:projects/*/keyspaces/*/keys/*#verify_key`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/keys/*#decrypt_key`,
      `unkey:v1:${ws}:**#*`,
    ];

    for (const permission of accepted) {
      expect(permissionValidation.safeParse(permission).success).toBe(true);
    }
  });

  test("rejects paths, actions and wildcard shapes outside the catalog", () => {
    const rejected = [
      // A wildcard parent cannot carry a concrete child.
      `unkey:v1:${ws}:projects/*/apps/app_12345678#read_app`,
      `unkey:v1:${ws}:projects/*/keyspaces/ks_12345678#read_keyspace`,
      // Actions the model collapsed into write, or never had.
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#start_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/apps/app_12345678/environments/env_12345678/deployments/deploy_12345678#promote_deployment`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/keys/key_12345678#create_key`,
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/keys/key_12345678#encrypt_key`,
      // Paths that lost their project scope or gained a segment.
      `unkey:v1:${ws}:keyspaces/ks_12345678#read_keyspace`,
      `unkey:v1:${ws}:identities/id_12345678#read_identity`,
      `unkey:v1:${ws}:projects/proj_12345678/github/apps/123456#read_github_app`,
      `unkey:v1:${ws}:projects/proj_12345678/rbac#write_role`,
      // Right path, wrong action for it.
      `unkey:v1:${ws}:projects/proj_12345678/keyspaces/ks_12345678/logs#write_keyspace`,
      // "**" only carries the admin action.
      `unkey:v1:${ws}:projects/proj_12345678/**#read_keyspace`,
      `unkey:v1:${ws}:**#read_key`,
    ];

    for (const permission of rejected) {
      expect(permissionValidation.safeParse(permission).success).toBe(false);
    }
  });

  test("names the action and the path it was refused on", () => {
    expect(firstError(`unkey:v1:${ws}:projects/proj_12345678#read_keyspace`)).toBe(
      '"read_keyspace" is not an action on "projects/proj_12345678". See the permission catalog in @unkey/rbac.',
    );
  });
});

describe("urnPermissionWorkspaceId", () => {
  test("returns the workspace of a urn permission", () => {
    expect(urnPermissionWorkspaceId("unkey:v1:ws_123:keyspaces/ks_1#read_key")).toBe("ws_123");
    expect(urnPermissionWorkspaceId("unkey:v1:ws_456:**#*")).toBe("ws_456");
  });

  test("returns null for anything that is not a urn permission", () => {
    expect(urnPermissionWorkspaceId("api.api_123.read_api")).toBeNull();
    expect(urnPermissionWorkspaceId("*")).toBeNull();
    expect(urnPermissionWorkspaceId("unkey:v1:ws_123:keyspaces/ks_1")).toBeNull();
  });
});

describe("permissionValidation", () => {
  test("accepts both grammars", () => {
    expect(permissionValidation.safeParse("api.api_12345678.read_api").success).toBe(true);
    expect(permissionValidation.safeParse("*").success).toBe(true);
    expect(permissionValidation.safeParse(`unkey:v1:${ws}:projects/*#read_project`).success).toBe(
      true,
    );
  });

  test("rejects strings longer than the slug column", () => {
    const permission = `unkey:v1:ws_123:keyspaces/${"a".repeat(PERMISSION_MAX_LENGTH)}#read_key`;
    expect(firstError(permission)).toBe(
      `Permission must be at most ${PERMISSION_MAX_LENGTH} characters.`,
    );
  });

  test("reports the urn grammar rule that failed", () => {
    expect(firstError("unkey:v1:ws_123:keyspaces/ks_1")).toBe(
      'Permission must contain exactly one "#" action separator.',
    );
    expect(firstError("unkey:v1:ws_123:**/keys/key_1#read_key")).toBe(
      '"**" must be the last resource path segment.',
    );
    expect(firstError("unkey:v1:ws_123:keyspaces/ks_1#read_")).toBe(
      'Action must not start or end with "_".',
    );
    expect(firstError("unkey:v1:ws_123:keyspaces/ks_1#*")).toBe(
      'Action "*" requires the global resource path "**".',
    );
  });

  test("reports the legacy grammar rule that failed", () => {
    expect(firstError("api.api_12345678")).toBe(
      'Permission must be a "unkey:v1:<workspace_id>:<resource_path>#<action>" URN or a legacy "resource.id.action" tuple.',
    );
    expect(firstError("keyspace.ks_12345678.read_key")).toContain('Unknown resource "keyspace".');
    expect(firstError("api.nope.read_api")).toContain('Invalid id "nope" for resource "api".');
    expect(firstError("api.api_12345678.fly")).toContain(
      'Unknown action "fly" for resource "api".',
    );
  });

  test("reports one issue, not a union of two", () => {
    const result = permissionValidation.safeParse("unkey:v1:ws_123:keyspaces/ks_1#*");
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues).toHaveLength(1);
    }
  });
});
