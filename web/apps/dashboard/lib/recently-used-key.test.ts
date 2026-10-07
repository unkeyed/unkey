import { describe, expect, test } from "vitest";
import {
  RECENTLY_USED_WINDOW_MS,
  formatTimeSinceLastUse,
  isRecentlyUsed,
} from "./recently-used-key";

const NOW = 1_700_000_000_000;

describe("isRecentlyUsed", () => {
  test("treats a never used key as not recently used", () => {
    expect(isRecentlyUsed(0, NOW)).toBe(false);
  });

  test("flags usage inside the window", () => {
    expect(isRecentlyUsed(NOW - RECENTLY_USED_WINDOW_MS + 1, NOW)).toBe(true);
  });

  test("ignores usage at or beyond the window", () => {
    expect(isRecentlyUsed(NOW - RECENTLY_USED_WINDOW_MS, NOW)).toBe(false);
  });
});

describe("formatTimeSinceLastUse", () => {
  test("collapses sub-minute usage", () => {
    expect(formatTimeSinceLastUse(NOW - 5_000, NOW)).toBe("less than a minute ago");
  });

  test("collapses clock skew from the future", () => {
    expect(formatTimeSinceLastUse(NOW + 5_000, NOW)).toBe("less than a minute ago");
  });

  test("formats longer gaps", () => {
    expect(formatTimeSinceLastUse(NOW - 3 * 60 * 60 * 1000, NOW)).toBe("3 hours ago");
  });
});
