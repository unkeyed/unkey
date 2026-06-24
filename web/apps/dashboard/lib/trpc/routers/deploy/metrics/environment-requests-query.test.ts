import { describe, expect, test } from "vitest";
import { selectEnvironmentRequestsQuery } from "./environment-requests-query";

const minuteMs = 60_000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;
const now = Date.UTC(2026, 0, 8, 12);

describe("selectEnvironmentRequestsQuery", () => {
  test("1m interval, range hour, for apps younger than one hour", () => {
    const appCreatedAtMs = now - 59 * minuteMs;

    expect(selectEnvironmentRequestsQuery(appCreatedAtMs, now)).toEqual({
      range: "hour",
      interval: "1m",
      bucketMs: minuteMs,
      // Full past hour, anchored at now rather than truncated to creation.
      startTimeMs: now - hourMs,
      endTimeMs: now,
    });
  });

  test("spans the full tier window for a brand-new app, not just its lifetime", () => {
    // Regression: an app created minutes ago still gets the full past-hour
    // window. Buckets before it existed fill as zeros, so the chart shows a
    // full-width axis instead of collapsing to two or three points.
    expect(selectEnvironmentRequestsQuery(now - 5 * minuteMs, now)).toMatchObject({
      range: "hour",
      interval: "1m",
      startTimeMs: now - hourMs,
      endTimeMs: now,
    });
  });

  test("floors both grid bounds to bucketMs", () => {
    // offsetNow sits 37s into a minute; both the window start (now - 1h) and
    // the end (now) floor down to the minute boundary.
    const offsetNow = now + 37_000;
    const appCreatedAtMs = now - 40 * minuteMs;

    expect(selectEnvironmentRequestsQuery(appCreatedAtMs, offsetNow)).toMatchObject({
      bucketMs: minuteMs,
      startTimeMs: now - hourMs,
      endTimeMs: now,
    });
  });

  test("5m interval, range day, at one hour", () => {
    expect(selectEnvironmentRequestsQuery(now - hourMs, now)).toMatchObject({
      range: "day",
      interval: "5m",
      bucketMs: 5 * minuteMs,
    });
  });

  test("5m interval, range day, at six hours", () => {
    expect(selectEnvironmentRequestsQuery(now - 6 * hourMs, now)).toMatchObject({
      range: "day",
      interval: "5m",
      bucketMs: 5 * minuteMs,
    });
  });

  test("15m interval, range day, at twelve hours", () => {
    expect(selectEnvironmentRequestsQuery(now - 12 * hourMs, now)).toMatchObject({
      range: "day",
      interval: "15m",
      bucketMs: 15 * minuteMs,
    });
  });

  test("15m interval, range day, at one day", () => {
    expect(selectEnvironmentRequestsQuery(now - dayMs, now)).toMatchObject({
      range: "day",
      interval: "15m",
      bucketMs: 15 * minuteMs,
    });
  });

  test("15m interval, range week, just after one day", () => {
    expect(selectEnvironmentRequestsQuery(now - dayMs - 1, now)).toMatchObject({
      range: "week",
      interval: "15m",
      bucketMs: 15 * minuteMs,
    });
  });

  test("15m interval, range week, at two days", () => {
    expect(selectEnvironmentRequestsQuery(now - 2 * dayMs, now)).toMatchObject({
      range: "week",
      interval: "15m",
      bucketMs: 15 * minuteMs,
    });
  });

  test("1h interval, range week, just after two days", () => {
    expect(selectEnvironmentRequestsQuery(now - 2 * dayMs - 1, now)).toMatchObject({
      range: "week",
      interval: "1h",
      bucketMs: hourMs,
    });
  });

  test("caps old app windows to the last seven days", () => {
    expect(selectEnvironmentRequestsQuery(now - 30 * dayMs, now)).toMatchObject({
      interval: "1h",
      startTimeMs: now - 7 * dayMs,
      endTimeMs: now,
    });
  });
});
