import { describe, expect, test } from "vitest";
import {
  PERMISSION_MAX_LENGTH,
  buildIdSchema,
  permissionValidation,
  portalActions,
  ratelimitActions,
  unkeyPermissionValidation,
  unkeyUrnPermissionValidation,
  urnPermissionWorkspaceId,
} from "./permissions";
import urnGrammarCases from "./urn-grammar.fixture.json";

function firstError(input: unknown): string | undefined {
  const result = permissionValidation.safeParse(input);
  return result.success ? undefined : result.error.issues[0]?.message;
}

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

describe("urn grammar", () => {
  for (const { input, valid, reason } of urnGrammarCases) {
    test(`${valid ? "accepts" : "rejects"} ${input} (${reason})`, () => {
      expect(unkeyUrnPermissionValidation.safeParse(input).success).toBe(valid);
      expect(permissionValidation.safeParse(input).success).toBe(valid);
    });
  }

  test("rejects non-string input", () => {
    expect(unkeyUrnPermissionValidation.safeParse(42).success).toBe(false);
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
    expect(permissionValidation.safeParse("unkey:v1:ws_123:**#read_key").success).toBe(true);
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
