import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { type SavedFiltersGroup, useBookmarkedFilters } from "./use-bookmarked-filters";
describe("useBookmarkedFilters", () => {
  const mockSavedFiltersGroup: SavedFiltersGroup[] = [
    {
      id: "1",
      createdAt: Date.now(),
      filters: {
        methods: [],
        paths: [],
        status: [],
        host: [],
        requestId: [],
        startTime: 0,
        endTime: 0,
        since: "2d",
      },
      bookmarked: false,
    },
  ];
  beforeEach(() => {
    localStorage.clear();
  });

  it("should toggle bookmark correctly", () => {
    const { result } = renderHook(() => useBookmarkedFilters());

    localStorage.setItem("savedFilters", JSON.stringify(mockSavedFiltersGroup));

    act(() => {
      result.current.toggleBookmark("1");
    });

    const updatedFilters = JSON.parse(localStorage.getItem("savedFilters") || "[]");
    expect(updatedFilters[0].bookmarked).toBe(true);

    act(() => {
      result.current.toggleBookmark("1");
    });

    const toggledBackFilters = JSON.parse(localStorage.getItem("savedFilters") || "[]");
    expect(toggledBackFilters[0].bookmarked).toBe(false);
  });
});
