import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getSinceTime } from "./utils";

const NOW = new Date(2026, 0, 31, 12, 0, 0);
const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

describe("getSinceTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("steps from seconds to weeks by elapsed time", () => {
    const ago = (ms: number) => getSinceTime(NOW.getTime() - ms);
    expect(ago(30_000)).toBe("just now");
    expect(ago(5 * MINUTE)).toBe("5m ago");
    expect(ago(3 * HOUR)).toBe("3h ago");
    expect(ago(2 * DAY)).toBe("2d ago");
    expect(ago(10 * DAY)).toBe("1w ago");
    expect(ago(29 * DAY)).toBe("4w ago");
  });

  it("does not count a month until the anniversary's time of day has passed", () => {
    expect(getSinceTime(new Date(2025, 11, 31, 18).getTime())).toBe("4w ago");
    expect(getSinceTime(new Date(2025, 0, 31, 18).getTime())).toBe("11 month(s) ago");
  });

  it("counts months and years on the calendar", () => {
    expect(getSinceTime(new Date(2025, 11, 31, 12).getTime())).toBe("1 month(s) ago");
    expect(getSinceTime(new Date(2025, 10, 15, 12).getTime())).toBe("2 month(s) ago");
    expect(getSinceTime(new Date(2025, 0, 31, 12).getTime())).toBe("1 year(s) ago");
  });
});
