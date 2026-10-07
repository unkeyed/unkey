import { describe, expect, it } from "vitest";
import { isLeaveSentinel } from "./use-prevent-leave";

describe("isLeaveSentinel", () => {
  it("recognises the sentinel entry, including after Next.js merges its own state", () => {
    expect(isLeaveSentinel({ unkeyPreventLeaveSentinel: true })).toBe(true);
    expect(isLeaveSentinel({ unkeyPreventLeaveSentinel: true, __NA: true })).toBe(true);
  });

  it("rejects ordinary history entries", () => {
    expect(isLeaveSentinel(null)).toBe(false);
    expect(isLeaveSentinel(undefined)).toBe(false);
    expect(isLeaveSentinel({ __NA: true })).toBe(false);
    expect(isLeaveSentinel("unkeyPreventLeaveSentinel")).toBe(false);
  });
});
