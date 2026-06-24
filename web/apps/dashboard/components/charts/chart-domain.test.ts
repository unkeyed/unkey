import { describe, expect, it } from "vitest";
import { resolveXAxisDomain } from "./chart-domain";

const DAY_MS = 24 * 60 * 60 * 1000;
const WEEK: [number, number] = [0, 7 * DAY_MS];

describe("resolveXAxisDomain", () => {
  // Regression: the app overview "requests this week" card rendered a ~24h
  // axis because ~20h of traffic in a week-wide window sparsely covered it
  // and the axis silently contracted. Honoring the window must be the default
  // so the header and the axis agree.
  it("honors the requested window by default even when data is sparse", () => {
    const firstNonZeroTs = WEEK[1] - 20 * 60 * 60 * 1000;
    const { effectiveDomain, spanMs } = resolveXAxisDomain({
      xAxisDomain: WEEK,
      firstNonZeroTs,
      lastNonZeroTs: WEEK[1],
    });
    expect(effectiveDomain).toEqual(WEEK);
    expect(spanMs).toBe(WEEK[1]);
  });

  it("contracts to the data extent when opted in and coverage is below the threshold", () => {
    const firstNonZeroTs = WEEK[1] - 20 * 60 * 60 * 1000; // ~12% of a week
    const { effectiveDomain, spanMs } = resolveXAxisDomain({
      xAxisDomain: WEEK,
      contractOnSparseData: true,
      firstNonZeroTs,
      lastNonZeroTs: WEEK[1],
    });
    expect(effectiveDomain).toEqual([firstNonZeroTs, WEEK[1]]);
    expect(spanMs).toBe(WEEK[1] - firstNonZeroTs);
  });

  it("keeps the window when opted in but coverage clears the threshold", () => {
    const firstNonZeroTs = WEEK[1] - 6 * DAY_MS; // 6 of 7 days covered
    const { effectiveDomain } = resolveXAxisDomain({
      xAxisDomain: WEEK,
      contractOnSparseData: true,
      firstNonZeroTs,
      lastNonZeroTs: WEEK[1],
    });
    expect(effectiveDomain).toEqual(WEEK);
  });

  it("keeps the window for a degenerate single-bucket span so the lone point isn't lost", () => {
    const { effectiveDomain } = resolveXAxisDomain({
      xAxisDomain: WEEK,
      contractOnSparseData: true,
      firstNonZeroTs: 1000,
      lastNonZeroTs: 1000,
    });
    expect(effectiveDomain).toEqual(WEEK);
  });

  it("falls back to the non-zero extent when no window is requested", () => {
    const { effectiveDomain, spanMs } = resolveXAxisDomain({
      firstNonZeroTs: 100,
      lastNonZeroTs: 500,
    });
    expect(effectiveDomain).toEqual([100, 500]);
    expect(spanMs).toBe(400);
  });

  it("returns no domain when there is neither a window nor non-zero data", () => {
    const { effectiveDomain, spanMs } = resolveXAxisDomain({});
    expect(effectiveDomain).toBeUndefined();
    expect(spanMs).toBe(0);
  });
});
