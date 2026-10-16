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
    expect(formatTimestamp(date, "time")).toBe("3:04 PM");
    expect(formatTimestamp(date, "timeWithMillis")).toBe("15:04:05.123");
    expect(formatTimestamp(date, "monthDay")).toBe("Jan 5");
    expect(formatTimestamp(date, "monthYear")).toBe("Jan 2026");
    expect(formatTimestamp(date, "monthYearLong")).toBe("January 2026");
    expect(formatTimestamp(date, "dateTime")).toBe("Jan 5, 3:04 PM");
    expect(formatTimestamp(date, "dateTimeWithYear")).toBe("Jan 5, 2026, 3:04 PM");
    expect(formatTimestamp(date, "dateTime24h")).toBe("Jan 05, 15:04:05");
    expect(formatTimestamp(date, "full")).toBe("Jan 5, 2026, 15:04:05");
  });

  it("accepts epoch milliseconds, keeps midnight on the 24-hour clock, and can render UTC", () => {
    expect(formatTimestamp(date.getTime(), "time")).toBe("3:04 PM");
    expect(formatTimestamp(new Date(2026, 10, 30, 0, 0, 0), "full")).toBe("Nov 30, 2026, 00:00:00");
    expect(formatTimestamp(Date.UTC(2026, 0, 5, 15, 4, 5), "full", "UTC")).toBe(
      "Jan 5, 2026, 15:04:05",
    );
  });
});

describe("isSameDay", () => {
  it("compares calendar days in local time", () => {
    const noon = new Date(2026, 0, 5, 12);
    expect(isSameDay(noon, new Date(2026, 0, 5, 23, 59))).toBe(true);
    expect(isSameDay(noon, noon.getTime() + 24 * 60 * 60 * 1000)).toBe(false);
  });
});
