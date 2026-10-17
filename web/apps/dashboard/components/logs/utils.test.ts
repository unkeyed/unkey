import { describe, expect, it } from "vitest";
import { formatTooltipTimestamp } from "./utils";

const at = new Date(2026, 0, 5, 15, 4, 5).getTime();
const DAY = 24 * 60 * 60 * 1000;

describe("formatTooltipTimestamp", () => {
  it("shows only the time for sub-day buckets within one day", () => {
    const data = [{ originalTimestamp: at - 60_000 }, { originalTimestamp: at }];
    expect(formatTooltipTimestamp(at, "perMinute", data)).toBe("3:04 PM");
  });

  it("adds the date once the data spans days or the bucket is a day or more", () => {
    const data = [{ originalTimestamp: at - 2 * DAY }, { originalTimestamp: at }];
    expect(formatTooltipTimestamp(at, "perMinute", data)).toBe("Jan 5, 3:04 PM");
    expect(formatTooltipTimestamp(at, "perDay")).toBe("Jan 5, 3:04 PM");
  });

  it("returns an empty string for unparsable input", () => {
    expect(formatTooltipTimestamp("not a date")).toBe("");
  });
});
