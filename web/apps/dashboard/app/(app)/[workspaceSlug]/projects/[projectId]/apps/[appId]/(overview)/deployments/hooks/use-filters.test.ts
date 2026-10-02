import { renderHook } from "@testing-library/react";
import { useQueryStates } from "nuqs";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentListQuerySearchParams } from "../filters.schema";
import { useFilters } from "./use-filters";

vi.mock("nuqs", () => ({
  useQueryStates: vi.fn(),
  parseAsInteger: {
    parse: (str: string | null) => (str ? Number.parseInt(str) : null),
    serialize: (value: number | null) => value?.toString() ?? "",
  },
}));

vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "test-uuid") });

const mockUseQueryStates = vi.mocked(useQueryStates);
const mockSetSearchParams = vi.fn();

const stubSearchParams = (overrides: Partial<DeploymentListQuerySearchParams> = {}) => {
  mockUseQueryStates.mockImplementation(() => [
    {
      status: null,
      environment: null,
      branch: null,
      startTime: null,
      endTime: null,
      since: null,
      ...overrides,
    },
    mockSetSearchParams,
  ]);
};

const statusValues = (filters: ReturnType<typeof useFilters>["filters"]) =>
  filters.flatMap((f) => (f.field === "status" ? [f.value] : []));

describe("useFilters", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    stubSearchParams();
  });

  it("pre-selects the default status groups when the url carries no status", () => {
    const { result } = renderHook(() => useFilters());

    expect(statusValues(result.current.filters)).toEqual([
      "ready",
      "failed",
      "building",
      "queued",
      "blocked",
    ]);
    expect(result.current.isFiltered).toBe(false);
  });

  it("reports an explicit selection matching the default as unfiltered", () => {
    stubSearchParams({
      status: ["blocked", "queued", "building", "failed", "ready"].map((value) => ({
        value,
        operator: "is",
      })),
    });

    const { result } = renderHook(() => useFilters());
    expect(result.current.isFiltered).toBe(false);
  });

  it("keeps an explicit selection that widens past the default", () => {
    stubSearchParams({
      status: [{ value: "superseded", operator: "is" }],
    });

    const { result } = renderHook(() => useFilters());
    expect(statusValues(result.current.filters)).toEqual(["superseded"]);
    expect(result.current.isFiltered).toBe(true);
  });

  it("keeps the default statuses while another field is filtered", () => {
    stubSearchParams({ branch: [{ value: "main", operator: "is" }] });

    const { result } = renderHook(() => useFilters());
    expect(statusValues(result.current.filters)).toHaveLength(5);
    expect(result.current.isFiltered).toBe(true);
  });

  it("writes the default statuses alongside a newly toggled one", () => {
    const { result } = renderHook(() => useFilters());
    result.current.toggleArrayFilter("status", "superseded");

    expect(mockSetSearchParams).toHaveBeenCalledWith(
      expect.objectContaining({
        status: ["ready", "failed", "building", "queued", "blocked", "superseded"].map((value) => ({
          value,
          operator: "is",
        })),
      }),
    );
  });

  it("falls back to the default statuses when the last one is unticked", () => {
    stubSearchParams({ status: [{ value: "ready", operator: "is" }] });

    const { result } = renderHook(() => useFilters());
    result.current.toggleArrayFilter("status", "ready");

    expect(mockSetSearchParams).toHaveBeenCalledWith(expect.objectContaining({ status: null }));
  });
});
