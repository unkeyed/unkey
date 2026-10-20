import { describe, expect, it } from "vitest";
import { formatTimestampForChart, formatTimestampLabel } from "./format-timestamp";

const at = new Date(2026, 0, 5, 15, 4, 5).getTime();

describe("formatTimestampForChart", () => {
  it("adds detail as the bucket grows: time, date and time, date, month", () => {
    expect(formatTimestampForChart(at, "perMinute")).toBe("3:04 PM");
    expect(formatTimestampForChart(at, "per2Hours")).toBe("Jan 5, 3:04 PM");
    expect(formatTimestampForChart(at, "perWeek")).toBe("Jan 5");
    expect(formatTimestampForChart(at, "perMonth")).toBe("Jan 2026");
  });
});

describe("formatTimestampLabel", () => {
  it("uppercases the label and accepts numeric strings", () => {
    expect(formatTimestampLabel(at)).toBe("JAN 5, 3:04 PM");
    expect(formatTimestampLabel(String(at))).toBe("JAN 5, 3:04 PM");
  });
});
