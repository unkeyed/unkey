import { describe, expect, test } from "vitest";
import { buildIdSchema, ratelimitActions, unkeyPermissionValidation } from "./permissions";

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
