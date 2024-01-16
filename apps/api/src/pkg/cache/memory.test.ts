import { beforeEach, describe, expect, test } from "vitest";
import { MemoryCache } from "./memory";

describe("MemoryCache", () => {
  let memoryCache: MemoryCache<{ name: string }>;

  beforeEach(() => {
    memoryCache = new MemoryCache({
      fresh: 1_000_000,
      stale: 1_000_000,
    });
  });

  test("should store value in the cache", () => {
    memoryCache.set(null as any, "name", "key", "value");
    expect(memoryCache.get(null as any, "name", "key")).toEqual(["value", false]);
  });

  test("should return undefined if key does not exist in cache", () => {
    expect(memoryCache.get(null as any, "name", "invalidKey")).toEqual([undefined, false]);
  });

  test("should remove value from cache", () => {
    memoryCache.set(null as any, "name", "key", "value");
    memoryCache.remove(null as any, "name", "key");
    expect(memoryCache.get(null as any, "name", "key")).toEqual([undefined, false]);
  });
});
