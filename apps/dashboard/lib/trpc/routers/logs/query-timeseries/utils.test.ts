import { describe, expect, it } from "vitest";

import { HOUR_IN_MS, WEEK_IN_MS } from "./constants";
import { getTimeseriesGranularity, transformFilters } from "./utils";

describe("getTimeseriesGranularity", () => {
  const NOW = 1706024400000; // 2024-01-23T12:00:00.000Z

  it("should return perMinute granularity for missing start and end times", () => {
    const result = getTimeseriesGranularity(null, null);
    expect(result.granularity).toBe("perMinute");
    expect(result.endTime - result.startTime).toBe(HOUR_IN_MS);
  });

  it("should return perMinute granularity for timerange <= 1 hour", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS / 2;
    const result = getTimeseriesGranularity(startTime, endTime);

    expect(result).toEqual({
      granularity: "perMinute",
      startTime,
      endTime,
    });
  });

  it("should return perHour granularity for timerange > 1 hour and <= 1 week", () => {
    const endTime = NOW;
    const startTime = endTime - WEEK_IN_MS / 2;
    const result = getTimeseriesGranularity(startTime, endTime);

    expect(result).toEqual({
      granularity: "perHour",
      startTime,
      endTime,
    });
  });

  it("should return perDay granularity for timerange > 1 week", () => {
    const endTime = NOW;
    const startTime = endTime - WEEK_IN_MS * 2;
    const result = getTimeseriesGranularity(startTime, endTime);

    expect(result).toEqual({
      granularity: "perDay",
      startTime,
      endTime,
    });
  });

  it("should use current time as endTime when only startTime is provided", () => {
    const startTime = NOW - HOUR_IN_MS;
    const result = getTimeseriesGranularity(startTime, null);

    expect(result.endTime).toBeGreaterThan(startTime);
    expect(result.startTime).toBe(startTime);
  });
});

describe("transformFilters", () => {
  const basePayload = {
    startTime: 1706024400000,
    endTime: 1706028000000,
    since: "",
    path: null,
    host: null,
    method: null,
    status: null,
  };

  it("should transform empty filters correctly", () => {
    const result = transformFilters(basePayload);

    expect(result).toEqual({
      params: {
        startTime: basePayload.startTime,
        endTime: basePayload.endTime,
        hosts: [],
        methods: [],
        paths: [],
        statusCodes: [],
      },
      granularity: "perMinute",
    });
  });

  it("should transform filters with values correctly", () => {
    const payload = {
      ...basePayload,
      host: {
        filters: [{ operator: "is" as const, value: "example.com" }],
      },
      method: {
        filters: [{ operator: "is" as const, value: "GET" }],
      },
      path: {
        filters: [{ operator: "startsWith" as const, value: "/api" }],
      },
      status: {
        filters: [{ operator: "is" as const, value: 200 }],
      },
    };

    const result = transformFilters(payload);

    expect(result).toEqual({
      params: {
        startTime: payload.startTime,
        endTime: payload.endTime,
        hosts: ["example.com"],
        methods: ["GET"],
        paths: [{ operator: "startsWith", value: "/api" }],
        statusCodes: [200],
      },
      granularity: "perMinute",
    });
  });

  it('should handle relative time with "since" parameter', () => {
    const payload = {
      ...basePayload,
      since: "24h",
    };

    const result = transformFilters(payload);

    expect(result.params.endTime).toBeGreaterThan(result.params.startTime);
    expect(result.params.endTime - result.params.startTime).toBeCloseTo(24 * 60 * 60 * 1000, -2);
    expect(result.granularity).toBe("perHour");
  });
});
