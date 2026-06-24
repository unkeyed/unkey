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
      startTimeMs: appCreatedAtMs,
      endTimeMs: now,
    });
  });

  test("floors both grid bounds to bucketMs", () => {
    // now sits 37s into a minute, app created 22s into an earlier minute.
    const offsetNow = now + 37_000;
    const appCreatedAtMs = now - 40 * minuteMs + 22_000;

    expect(selectEnvironmentRequestsQuery(appCreatedAtMs, offsetNow)).toMatchObject({
      bucketMs: minuteMs,
      startTimeMs: now - 40 * minuteMs,
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

  test("15m interval, range day, at six hours", () => {
    expect(selectEnvironmentRequestsQuery(now - 6 * hourMs, now)).toMatchObject({
      range: "day",
      interval: "15m",
      bucketMs: 15 * minuteMs,
    });
  });

  test("1h interval, range week, at twelve hours", () => {
    expect(selectEnvironmentRequestsQuery(now - 12 * hourMs, now)).toMatchObject({
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
