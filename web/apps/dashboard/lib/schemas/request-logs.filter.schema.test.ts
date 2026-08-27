import { describe, expect, it } from "vitest";
import { requestLogsFilterOutputSchema } from "./request-logs.filter.schema";

describe("requestLogsFilterOutputSchema", () => {
  it.each([
    ["region", "is", "us-east-1", true],
    ["host", "is", "api.example.com", true],
    ["requestId", "is", "req_123", true],
    ["paths", "startsWith", "/api", true],
    ["paths", "endsWith", "/users", false],
  ])("validates %s %s filters", (field, operator, value, valid) => {
    expect(
      requestLogsFilterOutputSchema.safeParse({
        filters: [{ field, filters: [{ operator, value }] }],
      }).success,
    ).toBe(valid);
  });
});
