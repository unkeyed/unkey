import { describe, expect, test } from "vitest";
import { buildIdSchema, unkeyPermissionValidation } from "./permissions";

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
  const testCases = [
    {
      input: "unkey:v1:ws_123:keyspaces/*/keys/*#create_key",
      valid: true,
    },
    {
      input: "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#read_deployment",
      valid: true,
    },
    {
      input: "unkey:v1:ws_123:projects/proj_123/**#read_deployment",
      valid: true,
    },
    {
      input: "unkey:v1:ws_123:**#*",
      valid: true,
    },
    {
      input: "unkey:v1:ws_123:keyspaces/ks_123#*",
      valid: false,
    },
    {
      input: "unkey:v1:ws_123:keyspaces/ks_*#read_key",
      valid: false,
    },
    {
      input: "unkey:v1:ws_123:projects/**/deployments/*#read_deployment",
      valid: false,
    },
    {
      input: "unkey:v1:ws_123:projects/*/apps/app_123#read_app",
      valid: false,
    },
    {
      input: "unkey:v1:ws_123:projects/proj_123/apps/*/environments/env_123#read_environment",
      valid: false,
    },
    {
      input:
        "unkey:v1:ws_123:projects/proj_123/apps/*/environments/*/deployments/dep_123#read_deployment",
      valid: false,
    },
  ];

  for (const { input, valid } of testCases) {
    test(`parsing ${input} should be ${valid ? "valid" : "invalid"}`, () => {
      const result = unkeyPermissionValidation.safeParse(input);
      expect(result.success).toBe(valid);
    });
  }
});
