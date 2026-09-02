import { describe, expect, it } from "vitest";
import { httpHeadersSchema, httpHeaderUpdatesSchema } from "./validation";

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

describe("HTTP header update validation", () => {
  it("accepts preserved and replaced values", () => {
    expect(
      httpHeaderUpdatesSchema.safeParse([
        { mode: "preserve", name: "Authorization" },
        { mode: "set", name: "X-Source", value: "unkey" },
      ]).success,
    ).toBe(true);
  });

  it("accepts an empty final header set", () => {
    expect(httpHeaderUpdatesSchema.safeParse([]).success).toBe(true);
  });

  it.each([
    [[{ mode: "set", name: "Authorization", value: "" }], "Invalid header value"],
    [
      [
        { mode: "preserve", name: "Authorization" },
        { mode: "set", name: "authorization", value: "replacement" },
      ],
      "Header name is duplicated",
    ],
  ])("rejects invalid updates", (headers, message) => {
    const result = httpHeaderUpdatesSchema.safeParse(headers);

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.map((issue) => issue.message)).toContain(message);
    }
  });
});
