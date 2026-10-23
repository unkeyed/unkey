import { describe, expect, it } from "vitest";
import { formatTimestamp, isSameDay, relativeTime } from "./time";

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

describe("formatTimestamp", () => {
  const date = new Date(2026, 0, 5, 15, 4, 5, 123);

  it("renders each style in the reader's timezone", () => {
    expect(formatTimestamp(date, "h:mm a")).toBe("3:04 PM");
    expect(formatTimestamp(date, "HH:mm:ss.SSS")).toBe("15:04:05.123");
    expect(formatTimestamp(date, "MMM d")).toBe("Jan 5");
    expect(formatTimestamp(date, "M/d/yy")).toBe("1/5/26");
    expect(formatTimestamp(date, "MMM yyyy")).toBe("Jan 2026");
    expect(formatTimestamp(date, "MMMM yyyy")).toBe("January 2026");
    expect(formatTimestamp(date, "MMM d, h:mm a")).toBe("Jan 5, 3:04 PM");
    expect(formatTimestamp(date, "MMM d, yyyy, h:mm a")).toBe("Jan 5, 2026, 3:04 PM");
    expect(formatTimestamp(date, "MMM dd, HH:mm:ss")).toBe("Jan 05, 15:04:05");
    expect(formatTimestamp(date, "MMM d, yyyy, HH:mm:ss")).toBe("Jan 5, 2026, 15:04:05");
  });

  it("accepts epoch milliseconds, keeps midnight on the 24-hour clock, and can render UTC", () => {
    expect(formatTimestamp(date.getTime(), "h:mm a")).toBe("3:04 PM");
    expect(formatTimestamp(new Date(2026, 10, 30, 0, 0, 0), "MMM d, yyyy, HH:mm:ss")).toBe(
      "Nov 30, 2026, 00:00:00",
    );
    expect(formatTimestamp(Date.UTC(2026, 0, 5, 15, 4, 5), "MMM d, yyyy, HH:mm:ss", "UTC")).toBe(
      "Jan 5, 2026, 15:04:05",
    );
  });

  it("names TIMESTAMP_STYLES when a style is missing", () => {
    // @ts-expect-error "HH:mm" is not a style yet
    expect(() => formatTimestamp(date, "HH:mm")).toThrow(/TIMESTAMP_STYLES/);
  });
});

describe("isSameDay", () => {
  it("compares calendar days in local time", () => {
    const noon = new Date(2026, 0, 5, 12);
    expect(isSameDay(noon, new Date(2026, 0, 5, 23, 59))).toBe(true);
    expect(isSameDay(noon, noon.getTime() + 24 * 60 * 60 * 1000)).toBe(false);
  });
});
