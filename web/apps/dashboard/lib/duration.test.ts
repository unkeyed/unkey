import { describe, expect, test } from "vitest";
import { getTimestampFromRelative, parseDuration } from "./duration";

describe("parseDuration", () => {
  test.each([
    { aliases: ["ms", "millisecond", "milliseconds"], milliseconds: 1 },
    { aliases: ["s", "sec", "second", "seconds"], milliseconds: 1000 },
    { aliases: ["m", "min", "minute", "minutes"], milliseconds: 60_000 },
    { aliases: ["h", "hr", "hour", "hours"], milliseconds: 3_600_000 },
    { aliases: ["d", "day", "days"], milliseconds: 86_400_000 },
    { aliases: ["w", "week", "weeks"], milliseconds: 604_800_000 },
  ])("supports $aliases", ({ aliases, milliseconds }) => {
    for (const unit of aliases) {
      expect(parseDuration(`1 ${unit}`)).toBe(milliseconds);
    }
  });

  test.each([
    [" 2 HOURS ", 7_200_000],
    ["1w2d3h30m", 790_200_000],
    ["1 hour 2 minutes 3 seconds 4ms", 3_723_004],
    ["0m", 0],
  ])("parses %s", (input, expected) => {
    expect(parseDuration(input)).toBe(expected);
  });

  test.each(["", "1", "-1h", "1.5h", "1h!", "1h2", "1h2unknown", "1constructor"])(
    "rejects the whole invalid duration %s",
    (input) => {
      expect(parseDuration(input)).toBe(0);
    },
  );
});

describe("getTimestampFromRelative", () => {
  test("subtracts a combined duration from the supplied time", () => {
    expect(getTimestampFromRelative("1w2d3h30m", Date.parse("2026-09-28T12:34:56.789Z"))).toBe(
      Date.parse("2026-09-19T09:04:56.789Z"),
    );
  });

  test("accepts a zero-length relative window", () => {
    expect(getTimestampFromRelative("0m", 123)).toBe(123);
  });

  test.each(["", "1h!", "1s", "1ms", "1H", "1 hour", " 1w"])(
    "preserves URL validation for %s",
    (input) => {
      expect(() => getTimestampFromRelative(input)).toThrow("Invalid relative time format");
    },
  );
});
