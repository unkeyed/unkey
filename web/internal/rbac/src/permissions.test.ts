import { describe, expect, test } from "vitest";
import {
  buildIdSchema,
  portalActions,
  ratelimitActions,
  unkeyPermissionValidation,
} from "./permissions";

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
