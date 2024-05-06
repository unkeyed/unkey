import { assert, describe, expect, it } from "vitest";
import { matchesPath } from "../../util";

describe("matchesPath", () => {
  it("should return true if path matches the exact pattern", () => {
    const path = "/api/user";
    const pattern = "/api/user";
    const result = matchesPath(path, pattern);
    expect(result).toBe(true);
  });

  it("should return true if path matches the pattern with any", () => {
    const path = "/api/user";
    const pattern = "/api/(.*)";
    const result = matchesPath(path, pattern);
    expect(result).toBe(true);
  });

  it("should return true if path matches the patter with a slug", () => {
    const path = "/api/user/123";
    const pattern = "/api/user/:id";
    const result = matchesPath(path, pattern);
    expect(result).toBe(true);
  });

  it("should return false if path does not match the pattern", () => {
    const path = "/api/user";
    const pattern = "/admin";
    const result = matchesPath(path, pattern);
    expect(result).toBe(false);
  });
});
