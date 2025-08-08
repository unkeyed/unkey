import { trpc } from "@/lib/trpc/client";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useLogsQuery } from "./use-logs-query";

let mockFilters: any[] = [];
const mockDate = 1706024400000;

vi.mock("@/providers/query-time-provider", () => ({
  QueryTimeProvider: ({ children }: { children: React.ReactNode }) => children,
  useQueryTime: () => ({
    queryTime: new Date(mockDate),
    refreshQueryTime: vi.fn(),
  }),
}));

vi.mock("@/lib/trpc/client", () => {
  const useInfiniteQuery = vi.fn().mockReturnValue({
    data: null,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    isFetchingNextPage: false,
    isLoading: false,
  });

  const fetch = vi.fn();

  return {
    trpc: {
      useUtils: () => ({
        logs: {
          queryLogs: {
            fetch,
          },
        },
      }),
      logs: {
        queryLogs: {
          useInfiniteQuery,
        },
      },
    },
  };
});

vi.mock("../../../hooks/use-filters", () => ({
  useFilters: () => ({
    filters: mockFilters,
  }),
}));

describe("useLogsQuery filter processing", () => {
  beforeEach(() => {
    mockFilters = [];
    vi.setSystemTime(mockDate);
  });

  it("handles valid status filter", () => {
    mockFilters = [{ field: "status", operator: "is", value: "404" }];
    const { result } = renderHook(() => useLogsQuery());
    expect(result.current.isPolling).toBe(false);
  });

  it("handles multiple valid filters", () => {
    mockFilters = [
      { field: "status", operator: "is", value: "404" },
      { field: "methods", operator: "is", value: "GET" },
      { field: "paths", operator: "startsWith", value: "/api" },
    ];
    const { result } = renderHook(() => useLogsQuery());
    expect(result.current.isPolling).toBe(false);
  });

  it("handles invalid filter types", () => {
    const consoleMock = vi.spyOn(console, "error");
    mockFilters = [
      { field: "methods", operator: "is", value: 123 },
      { field: "paths", operator: "startsWith", value: true },
      { field: "host", operator: "is", value: {} },
    ];
    renderHook(() => useLogsQuery());
    expect(consoleMock).toHaveBeenCalledTimes(6);
  });

  it("handles time-based filters", () => {
    mockFilters = [
      { field: "startTime", operator: "is", value: mockDate - 3600000 },
      { field: "since", operator: "is", value: "1h" },
    ];
    const { result } = renderHook(() => useLogsQuery());
    expect(result.current.isPolling).toBe(false);
  });
});

describe("useLogsQuery realtime logs", () => {
  let useInfiniteQuery: ReturnType<typeof vi.fn>;
  let fetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.setSystemTime(mockDate);
    mockFilters = [];
    //@ts-expect-error hacky way to mock trpc
    useInfiniteQuery = vi.mocked(trpc.logs.queryLogs.useInfiniteQuery);
    //@ts-expect-error hacky way to mock trpc
    fetch = vi.mocked(trpc.useUtils().logs.queryLogs.fetch);
  });

  it("resets realtime logs when polling stops", async () => {
    const mockLogs = [
      { request_id: "1", time: Date.now(), method: "GET", path: "/api/test" },
      { request_id: "2", time: Date.now(), method: "POST", path: "/api/users" },
    ];

    useInfiniteQuery.mockReturnValue({
      data: {
        pages: [{ logs: mockLogs, nextCursor: null }],
      },
      hasNextPage: false,
      fetchNextPage: vi.fn(),
      isFetchingNextPage: false,
      isLoading: false,
    });

    fetch.mockResolvedValue({
      logs: [
        {
          request_id: "3",
          time: Date.now(),
          method: "PUT",
          path: "/api/update",
        },
      ],
    });

    const { result, rerender } = renderHook(
      ({ startPolling, pollIntervalMs }) => useLogsQuery({ startPolling, pollIntervalMs }),
      { initialProps: { startPolling: true, pollIntervalMs: 1000 } },
    );

    expect(result.current.historicalLogs).toHaveLength(2);

    // Wait for polling interval
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve));
    });

    act(() => {
      rerender({ startPolling: false, pollIntervalMs: 1000 });
    });

    expect(result.current.realtimeLogs).toHaveLength(0);
    expect(result.current.historicalLogs).toHaveLength(2);
  });
});
