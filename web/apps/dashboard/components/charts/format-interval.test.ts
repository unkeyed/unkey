import { describe, expect, it } from "vitest";
import { formatBucketInterval } from "./format-interval";

// Midday, well clear of any day boundary, so the multi-day assertion holds
// regardless of the runner's timezone.
const T0 = Date.UTC(2026, 5, 17, 12, 0, 0);
const PLUS_1MIN = T0 + 60_000;
const PLUS_2DAYS = T0 + 2 * 24 * 60 * 60 * 1000;

// Assertions check the separators this function controls (the en-dash range
// joiner and the literal ", " date prefix) rather than locale-rendered
// date/time strings, which vary by the runner's locale and timezone.
const commaCount = (s: string) => (s.match(/,/g) ?? []).length;

describe("formatBucketInterval", () => {
  it("renders only the start when there is no next bucket", () => {
    expect(formatBucketInterval(T0, undefined)).not.toContain("–");
  });

  it("renders a start-end range when a next bucket is given", () => {
    expect(formatBucketInterval(T0, PLUS_1MIN)).toContain("–");
  });

  it("adds a date prefix only when withDate is set", () => {
    const plain = formatBucketInterval(T0, PLUS_1MIN);
    const dated = formatBucketInterval(T0, PLUS_1MIN, true);
    expect(commaCount(dated)).toBeGreaterThan(commaCount(plain));
  });

  it("carries both endpoints' dates across a multi-day range so a bare time isn't ambiguous", () => {
    // Start date + cross-day end date → two ", " prefixes.
    expect(commaCount(formatBucketInterval(T0, PLUS_2DAYS, true))).toBe(2);
  });
});
