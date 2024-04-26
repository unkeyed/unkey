import { beforeEach, describe, expect, test } from "vitest";
import { MemoryStore } from "./memory";

describe("MemoryStore", () => {
  let memoryStore: MemoryStore<{ name: string }>;

  beforeEach(() => {
    memoryStore = new MemoryStore(new Map());
  });

  test("should store value in the cache", () => {
    memoryStore.set(null as any, "name", "key", "value");
    expect(memoryStore.get(null as any, "name", "key")).toEqual(["value", false]);
  });

  test("should return undefined if key does not exist in cache", () => {
    expect(memoryStore.get(null as any, "name", "invalidKey")).toEqual([undefined, false]);
  });

  test("should remove value from cache", () => {
    memoryStore.set(null as any, "name", "key", "value");
    memoryStore.remove(null as any, "name", "key");
    expect(memoryStore.get(null as any, "name", "key")).toEqual([undefined, false]);
  });
});
