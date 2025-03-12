import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { type SavedFiltersGroup, useBookmarkedFilters } from "./use-bookmarked-filters";

// Mock modules
vi.mock("nuqs", () => {
  const mockSetSearchParams = vi.fn();

  return {
    useQueryStates: vi.fn(() => [
      {
        requestIds: null,
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

let mockFilters: any[] = [];
const mockUpdateFilters = vi.fn((newArray) => {
  mockFilters.push(newArray);
});
vi.mock("./use-filters", () => ({
  useFilters: vi.fn(() => ({
    filters: mockFilters,
    updateFilters: mockUpdateFilters,
    removeFilter: vi.fn(),
  })),
}));

vi.stubGlobal("crypto", {
  randomUUID: vi.fn(() => "test-uuid"),
});

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, "localStorage", {
  value: localStorageMock,
});

describe("useBookmarkedFilters", () => {
  beforeEach(() => {
    localStorageMock.clear();
    mockFilters = [];
  });

  it("should return savedFilters from localStorage", () => {
    const savedFilters = [
      {
        id: "group-1",
        createdAt: 1632000000000,
        filters: {
          since: "2d",
          startTime: 0,
          endTime: 0,
        },
        bookmarked: false,
      },
    ];

    localStorageMock.setItem("savedFilters", JSON.stringify(savedFilters));

    const { result } = renderHook(() => useBookmarkedFilters());

    expect(result.current.savedFilters).toEqual(savedFilters);
    expect(localStorageMock.getItem).toHaveBeenCalledWith("savedFilters");
  });

  it("should toggle bookmark status correctly", () => {
    const savedFilters = [
      {
        id: "group-1",
        createdAt: 1632000000000,
        filters: {
          status: [{ value: 200, operator: "is" }],
          endTime: 0,
          startTime: 0,
          since: null,
        },
        bookmarked: false,
      },
    ];

    localStorageMock.setItem("savedFilters", JSON.stringify(savedFilters));

    const { result, rerender } = renderHook(() => useBookmarkedFilters());

    act(() => {
      result.current.toggleBookmark(savedFilters[0].id);
    });

    rerender();

    let updatedFilters = JSON.parse(localStorageMock.getItem("savedFilters")!);

    expect(updatedFilters[0].bookmarked).toBe(true);
    expect(result.current.savedFilters[0].bookmarked).toBe(true);
    act(() => {
      result.current.toggleBookmark(savedFilters[0].id);
    });

    rerender();

    updatedFilters = JSON.parse(localStorageMock.getItem("savedFilters")!);

    expect(updatedFilters[0].bookmarked).toBe(false);
    expect(result.current.savedFilters[0].bookmarked).toBe(false);
  });

  it("should apply filter group correctly", () => {
    const savedGroup: SavedFiltersGroup = {
      id: "group-1",
      createdAt: 1632000000000,
      filters: {
        since: "24h",
        methods: null,
        paths: null,
        status: null,
        host: null,
        requestId: null,
      },
    };

    const { result, rerender } = renderHook(() => useBookmarkedFilters());
    const { applyFilterGroup } = result.current;
    expect(mockFilters.length).toBe(0);
    act(() => {
      applyFilterGroup(savedGroup);
    });

    rerender();

    expect(mockFilters.length).toBe(1);
    expect(mockFilters[0][0]).toEqual({
      id: "test-uuid",
      field: "since",
      operator: "is",
      value: "24h",
    });
  });
});
