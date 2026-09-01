import { describe, expect, it } from "vitest";
import { headerFieldsSchema, headerUpdateFieldsSchema, toHeaderRecord } from "./header-fields";

describe("HTTP header fields", () => {
  it("accepts an empty row as no headers", () => {
    const headers = [{ name: "", value: "" }];

    expect(headerFieldsSchema.safeParse(headers).success).toBe(true);
    expect(toHeaderRecord(headers)).toEqual({});
  });

  it("converts complete rows to a header record", () => {
    const headers = [
      { name: " Authorization ", value: "Bearer token" },
      { name: "X-Source", value: "unkey" },
    ];

    expect(headerFieldsSchema.safeParse(headers).success).toBe(true);
    expect(toHeaderRecord(headers)).toEqual({
      Authorization: "Bearer token",
      "X-Source": "unkey",
    });
  });

  it.each([
    [[{ name: "Authorization", value: "" }], "Enter a header value"],
    [[{ name: "", value: "Bearer token" }], "Enter a header name"],
    [[{ name: "Invalid Header", value: "value" }], "Enter a valid header name"],
    [
      [{ name: "Authorization", value: "Bearer token\r\nX-Injected: value" }],
      "Enter a valid header value",
    ],
    [
      [
        { name: "Authorization", value: "one" },
        { name: "authorization", value: "two" },
      ],
      "Header name is duplicated",
    ],
  ])("rejects invalid rows", (headers, message) => {
    const result = headerFieldsSchema.safeParse(headers);

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.map((issue) => issue.message)).toContain(message);
    }
  });
});

describe("write-only HTTP header fields", () => {
  it("accepts stored values without plaintext and new values with plaintext", () => {
    expect(
      headerUpdateFieldsSchema.safeParse([
        { mode: "preserve", name: "Authorization" },
        { mode: "set", name: "X-Source", value: "unkey" },
      ]).success,
    ).toBe(true);
  });

  it("accepts an empty final header set", () => {
    expect(headerUpdateFieldsSchema.safeParse([]).success).toBe(true);
  });

  it("rejects case-insensitive duplicate names", () => {
    const result = headerUpdateFieldsSchema.safeParse([
      { mode: "preserve", name: "Authorization" },
      { mode: "set", name: "authorization", value: "replacement" },
    ]);

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.map((issue) => issue.message)).toContain(
        "Header name is duplicated",
      );
    }
  });
});
