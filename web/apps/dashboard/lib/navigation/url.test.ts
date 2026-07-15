import { describe, expect, it } from "vitest";
import { buildUrl, toQueryString, withQuery } from "./url";

describe("toQueryString", () => {
  it("writes values verbatim and joins with &", () => {
    expect(toQueryString({ product_id: 12374, quantity: 42 })).toBe("product_id=12374&quantity=42");
  });

  it("keeps readable special characters unencoded", () => {
    expect(toQueryString({ deploymentId: "contains:dep_123" })).toBe(
      "deploymentId=contains:dep_123",
    );
  });

  it("drops null and undefined values but keeps 0 and false", () => {
    expect(toQueryString({ a: null, b: undefined, c: 0, d: false })).toBe("c=0&d=false");
  });

  it("returns an empty string when every value is dropped", () => {
    expect(toQueryString({ a: null, b: undefined })).toBe("");
  });
});

describe("withQuery", () => {
  it("appends a query string to the base", () => {
    expect(withQuery("/p", { deploymentId: "contains:dep_123" })).toBe(
      "/p?deploymentId=contains:dep_123",
    );
  });

  it("leaves the base untouched when the query is empty", () => {
    expect(withQuery("/p", {})).toBe("/p");
    expect(withQuery("/p", { a: undefined })).toBe("/p");
  });
});

describe("buildUrl", () => {
  it("joins base and segments verbatim, preserving slashes inside a segment", () => {
    expect(buildUrl({ base: "https://github.com", segments: ["acme/api", "commit", "x"] })).toBe(
      "https://github.com/acme/api/commit/x",
    );
  });

  it("builds a relative path from segments only", () => {
    expect(buildUrl({ segments: ["acme", "projects", "p_1"] })).toBe("acme/projects/p_1");
  });

  it("returns the base alone when there are no segments", () => {
    expect(buildUrl({ base: "https://github.com" })).toBe("https://github.com");
  });

  it("appends the query after base and segments", () => {
    expect(
      buildUrl({ base: "", segments: ["acme", "p_1", "requests"], query: { since: "24h" } }),
    ).toBe("acme/p_1/requests?since=24h");
  });
});
