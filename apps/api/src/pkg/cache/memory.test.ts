import { MemoryCache } from "./memory";
import { describe, beforeEach, test, expect } from "bun:test";

describe("MemoryCache", () => {
  let memoryCache: MemoryCache<string, string>;

  beforeEach(() => {
    memoryCache = new MemoryCache({
      fresh: 1_000_000,
      stale: 1_000_000,
    });
  });

  test("should store value in the cache", () => {
    memoryCache.set(null as any, "namespace", "key", "value");
    expect(memoryCache.get(null as any, "namespace", "key")).toEqual(["value", false]);
  });

  test("should return undefined if key does not exist in cache", () => {
    expect(memoryCache.get(null as any, "namespace", "invalidKey")).toEqual([undefined, false]);
  });

  test("should remove value from cache", () => {
    memoryCache.set(null as any, "namespace", "key", "value");
    memoryCache.remove(null as any, "namespace", "key");
    expect(memoryCache.get(null as any, "namesapce", "key")).toEqual([undefined, false]);
  });
});
