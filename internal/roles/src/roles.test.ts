import { describe, test, expect } from "bun:test"
import { apiIdSchema } from "./roles"



describe("apiIdSchema", () => {
  const testCases = [
    { input: "123456789012", valid: false },
    { input: "api_1234asfas12", valid: true },
  ]

  for (const { input, valid } of testCases) {
    test(`parsing ${input} should be ${valid ? "valid" : "invalid"}`, () => {
      const result = apiIdSchema.safeParse(input)
      expect(result.success).toBe(valid)
    })
  }
})
