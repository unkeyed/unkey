import { describe, expect, it } from "vitest";
import { relativeTime } from "./time";

const NOW = Date.UTC(2026, 9, 5, 12, 0, 0);

describe("relativeTime", () => {
  it("reads 'just now' for anything under a minute ago by default", () => {
    expect(relativeTime(NOW - 59_000, NOW)).toBe("just now");
  });

  it("keeps seconds for future times so an expiry is never 'just now'", () => {
    expect(relativeTime(NOW + 30_000, NOW)).toBe("in 30 seconds");
  });

  it("counts minutes once a minute has passed", () => {
    expect(relativeTime(NOW - 60_000, NOW)).toBe("1 minute ago");
  });

  it("counts seconds with second precision", () => {
    expect(relativeTime(NOW - 21_000, NOW, "long", "second")).toBe("21 seconds ago");
    expect(relativeTime(NOW - 500, NOW, "long", "second")).toBe("just now");
  });

  it("applies the style to larger units", () => {
    expect(relativeTime(NOW - 3 * 60 * 60_000, NOW, "short")).toBe("3 hr. ago");
  });
});
