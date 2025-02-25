import { describe, expect, it } from "vitest";
import { HOUR_IN_MS, WEEK_IN_MS } from "../../utils/constants";
import { getTimeseriesGranularity } from "../../utils/granularity";
import { transformFilters } from "./utils";

describe("getTimeseriesGranularity", () => {
  const NOW = 1706024400000; // 2024-01-23T12:00:00.000Z

  // Original tests to ensure backward compatibility
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

  // New tests for additional granularities
  it("should return per5Minutes granularity for timerange > 10 minutes", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS / 4; // 15 minutes
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per5Minutes",
      startTime,
      endTime,
    });
  });

  it("should return per15Minutes granularity for timerange > 30 minutes", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS / 1.5; // 40 minutes
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per15Minutes",
      startTime,
      endTime,
    });
  });

  it("should return per30Minutes granularity for timerange > 45 minutes", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 0.8; // 48 minutes
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per30Minutes",
      startTime,
      endTime,
    });
  });

  it("should return per2Hours granularity for timerange > 3 hours", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 4;
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per2Hours",
      startTime,
      endTime,
    });
  });

  it("should return per4Hours granularity for timerange > 6 hours", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 7;
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per4Hours",
      startTime,
      endTime,
    });
  });

  it("should return per6Hours granularity for timerange > 8 hours", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 9;
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per6Hours",
      startTime,
      endTime,
    });
  });

  it("should return per8Hours granularity for timerange > 12 hours", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 13;
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per8Hours",
      startTime,
      endTime,
    });
  });

  it("should return per12Hours granularity for timerange > 16 hours", () => {
    const endTime = NOW;
    const startTime = endTime - HOUR_IN_MS * 18;
    const result = getTimeseriesGranularity(startTime, endTime);
    expect(result).toEqual({
      granularity: "per12Hours",
      startTime,
      endTime,
    });
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

  // Additional tests for transformFilters with new granularities
  it('should return per6Hours granularity for "7d" since parameter', () => {
    const payload = {
      ...basePayload,
      since: "7d",
    };
    const result = transformFilters(payload);
    expect(result.granularity).toBe("per6Hours");
    expect(result.params.endTime - result.params.startTime).toBeCloseTo(
      7 * 24 * 60 * 60 * 1000,
      -2,
    );
  });

  it('should return per12Hours granularity for "14d" since parameter', () => {
    const payload = {
      ...basePayload,
      since: "14d",
    };
    const result = transformFilters(payload);
    expect(result.granularity).toBe("per12Hours");
    expect(result.params.endTime - result.params.startTime).toBeCloseTo(
      14 * 24 * 60 * 60 * 1000,
      -2,
    );
  });

  it('should return perDay granularity for "30d" since parameter', () => {
    const payload = {
      ...basePayload,
      since: "30d",
    };
    const result = transformFilters(payload);
    expect(result.granularity).toBe("perDay");
    expect(result.params.endTime - result.params.startTime).toBeCloseTo(
      30 * 24 * 60 * 60 * 1000,
      -2,
    );
  });
});
