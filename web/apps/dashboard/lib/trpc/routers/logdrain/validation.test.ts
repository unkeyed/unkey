import { describe, expect, it } from "vitest";
import { httpHeadersSchema } from "./validation";

describe("HTTP header validation", () => {
  it("accepts valid headers", () => {
    expect(
      httpHeadersSchema.safeParse({
        Authorization: "Bearer token",
        "X-Source": "unkey",
      }).success,
    ).toBe(true);
  });

  it.each([
    [{ "Invalid Header": "value" }, "Invalid header name"],
    [{ Authorization: "Bearer token\r\nX-Injected: value" }, "Invalid header value"],
    [{ Authorization: "one", authorization: "two" }, "Header name is duplicated"],
  ])("rejects invalid headers", (headers, message) => {
    const result = httpHeadersSchema.safeParse(headers);

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.map((issue) => issue.message)).toContain(message);
    }
  });
});
