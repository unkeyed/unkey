import { describe, expect, it } from "vitest";
import { transformFilters } from "./utils";

describe("transformFilters", () => {
  const basePayload = {
    startTime: 1706024400000,
    endTime: 1706028000000,
    since: "",
    limit: 50,
    path: null,
    host: null,
    method: null,
    status: null,
    requestId: null,
    cursor: null,
  };

  it("should transform empty filters", () => {
    const result = transformFilters(basePayload);

    expect(result).toEqual({
      startTime: basePayload.startTime,
      endTime: basePayload.endTime,
      limit: 50,
      hosts: [],
      methods: [],
      paths: [],
      statusCodes: [],
      requestIds: [],
      cursorTime: null,
      cursorRequestId: null,
    });
  });

  it("should transform all filters with values", () => {
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
      requestId: {
        filters: [{ operator: "is" as const, value: "req123" }],
      },
      cursor: {
        time: 1706024400000,
        requestId: "cursor123",
      },
    };

    const result = transformFilters(payload);

    expect(result).toEqual({
      startTime: payload.startTime,
      endTime: payload.endTime,
      limit: 50,
      hosts: ["example.com"],
      methods: ["GET"],
      paths: [{ operator: "startsWith", value: "/api" }],
      statusCodes: [200],
      requestIds: ["req123"],
      cursorTime: 1706024400000,
      cursorRequestId: "cursor123",
    });
  });

  it("should handle relative time with since parameter", () => {
    const payload = {
      ...basePayload,
      since: "24h",
    };

    const result = transformFilters(payload);

    expect(result.endTime).toBeGreaterThan(result.startTime);
    expect(result.endTime - result.startTime).toBeCloseTo(24 * 60 * 60 * 1000, -2);
  });

  it("should handle cursor values", () => {
    const payload = {
      ...basePayload,
      cursor: {
        time: 1706024400000,
        requestId: "req123",
      },
    };

    const result = transformFilters(payload);

    expect(result.cursorTime).toBe(1706024400000);
    expect(result.cursorRequestId).toBe("req123");
  });
});
