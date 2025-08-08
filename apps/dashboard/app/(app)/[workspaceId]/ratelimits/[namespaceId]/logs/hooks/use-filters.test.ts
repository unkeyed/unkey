import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { act, renderHook } from "@testing-library/react";
import { useQueryStates } from "nuqs";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useFilters } from "./use-filters";

const parseAsFilterValArray = parseAsFilterValueArray(["is", "contains"]);
vi.mock("nuqs", () => {
  const mockSetSearchParams = vi.fn();

  return {
    useQueryStates: vi.fn(() => [
      {
        identifiers: null,
        startTime: null,
        endTime: null,
        status: null,
        since: null,
      },
      mockSetSearchParams,
    ]),
    parseAsInteger: {
      parse: (str: string | null) => (str ? Number.parseInt(str) : null),
      serialize: (value: number | null) => value?.toString() ?? "",
    },
  };
});

vi.stubGlobal("crypto", {
  randomUUID: vi.fn(() => "test-uuid"),
});

const mockUseQueryStates = vi.mocked(useQueryStates);
const mockSetSearchParams = vi.fn();

describe("parseAsFilterValueArray", () => {
  it("should return empty array for null input", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsFilterValArray.parse(null)).toEqual([]);
  });

  it("should return empty array for empty string", () => {
    expect(parseAsFilterValArray.parse("")).toEqual([]);
  });

  it("should parse single filter correctly", () => {
    const result = parseAsFilterValArray.parse("is:200");
    expect(result).toEqual([
      {
        operator: "is",
        value: "200",
      },
    ]);
  });

  it("should parse multiple filters correctly", () => {
    const result = parseAsFilterValArray.parse("is:200,contains:error");
    expect(result).toEqual([
      { operator: "is", value: "200" },
      { operator: "contains", value: "error" },
    ]);
  });

  it("should return empty array for invalid operator", () => {
    expect(parseAsFilterValArray.parse("invalid:200")).toEqual([]);
  });

  it("should serialize empty array to empty string", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsFilterValArray.serialize([])).toBe("");
  });

  it("should serialize array of filters correctly", () => {
    const filters = [
      { operator: "is", value: "200" },
      { operator: "contains", value: "error" },
    ];
    //@ts-expect-error ts yells for no reason
    expect(parseAsFilterValArray.serialize(filters)).toBe("is:200,contains:error");
  });
});

describe("parseAsRelativeTime", () => {
  it("should return null for null input", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsRelativeTime.parse(null)).toBeNull();
  });

  it("should return null for empty string", () => {
    expect(parseAsRelativeTime.parse("")).toBeNull();
  });

  it("should parse valid single unit formats", () => {
    expect(parseAsRelativeTime.parse("1h")).toBe("1h");
    expect(parseAsRelativeTime.parse("24h")).toBe("24h");
    expect(parseAsRelativeTime.parse("7d")).toBe("7d");
    expect(parseAsRelativeTime.parse("30m")).toBe("30m");
  });

  it("should parse valid multiple unit formats", () => {
    expect(parseAsRelativeTime.parse("1h30m")).toBe("1h30m");
    expect(parseAsRelativeTime.parse("2d5h")).toBe("2d5h");
    expect(parseAsRelativeTime.parse("1d6h30m")).toBe("1d6h30m");
  });

  it("should return null for invalid formats", () => {
    expect(parseAsRelativeTime.parse("1x")).toBeNull();
    expect(parseAsRelativeTime.parse("h")).toBeNull();
    expect(parseAsRelativeTime.parse("24")).toBeNull();
    expect(parseAsRelativeTime.parse("-1h")).toBeNull();
    expect(parseAsRelativeTime.parse("1h2")).toBeNull();
    expect(parseAsRelativeTime.parse("1h 2d")).toBeNull();
  });

  it("should serialize null to empty string", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsRelativeTime.serialize(null)).toBe("");
  });

  it("should serialize valid time strings correctly", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsRelativeTime.serialize("1h")).toBe("1h");
    //@ts-expect-error ts yells for no reason
    expect(parseAsRelativeTime.serialize("2d5h30m")).toBe("2d5h30m");
  });
});

describe("useFilters hook", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseQueryStates.mockImplementation(() => [
      {
        identifiers: null,
        startTime: null,
        endTime: null,
        status: null,
        since: null,
      },
      mockSetSearchParams,
    ]);
  });

  it("should initialize with empty filters", () => {
    const { result } = renderHook(() => useFilters());
    expect(result.current.filters).toEqual([]);
  });

  it("should handle multiple filter types", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        identifiers: [{ operator: "contains", value: "api" }],
        startTime: 1609459200000,
        endTime: null,
        status: null,
        since: "24h",
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());
    expect(result.current.filters).toEqual([
      {
        id: "test-uuid",
        field: "identifiers",
        operator: "contains",
        value: "api",
      },
      {
        id: "test-uuid",
        field: "startTime",
        operator: "is",
        value: 1609459200000,
      },
      {
        id: "test-uuid",
        field: "since",
        operator: "is",
        value: "24h",
      },
    ]);
  });

  it("should remove filter correctly", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        requestIds: [{ operator: "is", value: "123" }],
        identifiers: null,
        startTime: null,
        endTime: null,
        status: null,
        since: null,
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.removeFilter("test-uuid");
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      identifiers: null,
      startTime: null,
      endTime: null,
      status: null,
      since: null,
    });
  });

  it("should handle clearing all filters", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        identifiers: [{ operator: "contains", value: "api" }],
        startTime: 1609459200000,
        endTime: null,
        status: null,
        since: "24h",
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.updateFilters([]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      identifiers: null,
      startTime: null,
      endTime: null,
      status: null,
      since: null,
    });
  });

  it("should update filters correctly", () => {
    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.updateFilters([
        {
          id: "test-uuid-1",
          field: "identifiers",
          operator: "contains",
          value: "api",
        },
      ]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      identifiers: [{ operator: "contains", value: "api" }],
      startTime: null,
      endTime: null,
      status: null,
      since: null,
    });
  });
});
