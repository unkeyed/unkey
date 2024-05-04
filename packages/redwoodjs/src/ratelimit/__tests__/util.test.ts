import type { MiddlewareRequest } from "@redwoodjs/vite/dist/middleware";
import { assert, describe, expect, it } from "vitest";
import { defaultRatelimitIdentifier, matchesPath } from "../util";

describe("defaultRatelimitIdentifier", () => {
  it("should return correct identifier", () => {
    // Your test logic here
    const req = {} as MiddlewareRequest;
    assert.equal(defaultRatelimitIdentifier(req), "default");
  });
});

describe("matchesPath", () => {
  it("should return true if path matches the pattern", () => {
    const path = "/api/user";
    const pattern = "/api/*";
    const result = matchesPath(path, pattern);
    expect(result).toBe(true);
  });

  it("should return false if path does not match the pattern", () => {
    const path = "/api/user";
    const pattern = "/admin/*";
    const result = matchesPath(path, pattern);
    expect(result).toBe(false);
  });
});
