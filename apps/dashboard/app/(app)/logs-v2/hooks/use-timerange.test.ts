import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FilterValue } from "../filters.type";
import { getTimestampFromRelative, useTimeRange } from "./use-timerange";

describe("getTimestampFromRelative", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-01T00:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should convert hours correctly", () => {
    const result = getTimestampFromRelative("2h");
    const expected = new Date("2023-12-31T22:00:00.000Z").getTime();
    expect(result).toBe(expected);
  });

  it("should convert days correctly", () => {
    const result = getTimestampFromRelative("3d");
    const expected = new Date("2023-12-29T00:00:00.000Z").getTime();
    expect(result).toBe(expected);
  });

  it("should convert minutes correctly", () => {
    const result = getTimestampFromRelative("30m");
    const expected = new Date("2023-12-31T23:30:00.000Z").getTime();
    expect(result).toBe(expected);
  });

  it("should handle multiple units", () => {
    const result = getTimestampFromRelative("1d2h30m");
    const expected = new Date("2023-12-30T21:30:00.000Z").getTime();
    expect(result).toBe(expected);
  });

  it("should handle repeated units", () => {
    const result = getTimestampFromRelative("1h30m1h");
    const expected = new Date("2023-12-31T21:30:00.000Z").getTime();
    expect(result).toBe(expected);
  });

  it("should return current time minus zero milliseconds for empty string", () => {
    const result = getTimestampFromRelative("");
    expect(result).toBe(Date.now());
  });
});

describe("useTimeRange hook", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-01T00:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return undefined values for empty filters", () => {
    const { result } = renderHook(() => useTimeRange([]));
    expect(result.current).toEqual({
      startTime: undefined,
      endTime: undefined,
    });
  });

  it("should handle startTime and endTime filters", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "startTime",
        operator: "is",
        value: 1704067200000, // 2024-01-01T00:00:00.000Z
      },
      {
        id: "2",
        field: "endTime",
        operator: "is",
        value: 1704153600000, // 2024-01-02T00:00:00.000Z
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: 1704067200000,
      endTime: 1704153600000,
    });
  });

  it("should handle since filter", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "since",
        operator: "is",
        value: "2h",
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: new Date("2023-12-31T22:00:00.000Z").getTime(),
      endTime: Date.now(),
    });
  });

  it("should handle since filter with endTime", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "since",
        operator: "is",
        value: "2h",
      },
      {
        id: "2",
        field: "endTime",
        operator: "is",
        value: 1704153600000, // 2024-01-02T00:00:00.000Z
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: new Date("2023-12-31T22:00:00.000Z").getTime(),
      endTime: 1704153600000,
    });
  });

  it("should give precedence to since over startTime", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "since",
        operator: "is",
        value: "2h",
      },
      {
        id: "2",
        field: "startTime",
        operator: "is",
        value: 1704067200000, // This should be ignored
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: new Date("2023-12-31T22:00:00.000Z").getTime(),
      endTime: Date.now(),
    });
  });

  it("should handle partial time range (only startTime)", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "startTime",
        operator: "is",
        value: 1704067200000,
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: 1704067200000,
      endTime: undefined,
    });
  });

  it("should handle partial time range (only endTime)", () => {
    const filters: FilterValue[] = [
      {
        id: "1",
        field: "endTime",
        operator: "is",
        value: 1704153600000,
      },
    ];

    const { result } = renderHook(() => useTimeRange(filters));
    expect(result.current).toEqual({
      startTime: undefined,
      endTime: 1704153600000,
    });
  });
});
