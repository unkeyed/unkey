import { act, renderHook } from "@testing-library/react";
import { useQueryStates } from "nuqs";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { parseAsFilterValueArray, useFilters } from "./use-filters";

vi.mock("nuqs", () => {
  const mockSetSearchParams = vi.fn();

  return {
    useQueryStates: vi.fn(() => [
      {
        status: null,
        methods: null,
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
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
    expect(parseAsFilterValueArray.parse(null)).toEqual([]);
  });

  it("should return empty array for empty string", () => {
    expect(parseAsFilterValueArray.parse("")).toEqual([]);
  });

  it("should parse single filter correctly", () => {
    const result = parseAsFilterValueArray.parse("is:200");
    expect(result).toEqual([
      {
        operator: "is",
        value: "200",
      },
    ]);
  });

  it("should parse multiple filters correctly", () => {
    const result = parseAsFilterValueArray.parse("is:200,contains:error");
    expect(result).toEqual([
      { operator: "is", value: "200" },
      { operator: "contains", value: "error" },
    ]);
  });

  it("should return empty array for invalid operator", () => {
    expect(parseAsFilterValueArray.parse("invalid:200")).toEqual([]);
  });

  it("should serialize empty array to empty string", () => {
    //@ts-expect-error ts yells for no reason
    expect(parseAsFilterValueArray.serialize([])).toBe("");
  });

  it("should serialize array of filters correctly", () => {
    const filters = [
      { operator: "is", value: "200" },
      { operator: "contains", value: "error" },
    ];
    //@ts-expect-error ts yells for no reason
    expect(parseAsFilterValueArray?.serialize(filters)).toBe("is:200,contains:error");
  });
});

describe("useFilters hook", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseQueryStates.mockImplementation(() => [
      {
        status: null,
        methods: null,
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
      },
      mockSetSearchParams,
    ]);
  });

  it("should initialize with empty filters", () => {
    const { result } = renderHook(() => useFilters());
    expect(result.current.filters).toEqual([]);
  });

  it("should initialize with existing filters", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        status: [{ operator: "is", value: "200" }],
        methods: null,
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());
    expect(result.current.filters).toEqual([
      {
        id: "test-uuid",
        field: "status",
        operator: "is",
        value: "200",
        metadata: expect.any(Object),
      },
    ]);
  });

  it("should remove filter correctly", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        status: [{ operator: "is", value: "200" }],
        methods: null,
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.removeFilter("test-uuid");
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      status: null,
      methods: null,
      paths: null,
      host: null,
      requestId: null,
      startTime: null,
      endTime: null,
    });
  });

  it("should handle multiple filters", () => {
    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.updateFilters([
        {
          id: "test-uuid-1",
          field: "status",
          operator: "is",
          value: 200,
        },
        {
          id: "test-uuid-2",
          field: "methods",
          operator: "is",
          value: "GET",
        },
      ]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      status: [{ operator: "is", value: 200 }],
      methods: [{ operator: "is", value: "GET" }],
      paths: null,
      host: null,
      requestId: null,
      startTime: null,
      endTime: null,
    });
  });

  it("should handle time range filters", () => {
    const { result } = renderHook(() => useFilters());
    const startTime = 1609459200000;

    act(() => {
      result.current.updateFilters([
        {
          id: "test-uuid",
          field: "startTime",
          operator: "is",
          value: startTime,
        },
      ]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      status: null,
      methods: null,
      paths: null,
      host: null,
      requestId: null,
      startTime,
      endTime: null,
    });
  });

  it("should handle complex filter operators", () => {
    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.updateFilters([
        {
          id: "test-uuid-1",
          field: "paths",
          operator: "contains",
          value: "/api",
        },
        {
          id: "test-uuid-2",
          field: "host",
          operator: "startsWith",
          value: "test",
        },
      ]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      status: null,
      methods: null,
      paths: [{ operator: "contains", value: "/api" }],
      host: [{ operator: "startsWith", value: "test" }],
      requestId: null,
      startTime: null,
      endTime: null,
    });
  });

  it("should handle clearing all filters", () => {
    mockUseQueryStates.mockImplementation(() => [
      {
        status: [{ operator: "is", value: "200" }],
        methods: [{ operator: "is", value: "GET" }],
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
      },
      mockSetSearchParams,
    ]);

    const { result } = renderHook(() => useFilters());

    act(() => {
      result.current.updateFilters([]);
    });

    expect(mockSetSearchParams).toHaveBeenCalledWith({
      status: null,
      methods: null,
      paths: null,
      host: null,
      requestId: null,
      startTime: null,
      endTime: null,
    });
  });
});
