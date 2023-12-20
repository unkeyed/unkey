import { buildIdSchema } from "./roles";
import { describe, expect, test } from "bun:test";

describe("apiIdSchema", () => {
  const testCases = [
    { input: "123456789012", valid: false },
    { input: "a1234asfas12", valid: true },
    { input: "*", valid: true },
  ];

  for (const { input, valid } of testCases) {
    test(`parsing ${input} should be ${valid ? "valid" : "invalid"}`, () => {
      const result = buildIdSchema("api").safeParse(input);
      expect(result.success).toBe(valid);
    });
  }
});
