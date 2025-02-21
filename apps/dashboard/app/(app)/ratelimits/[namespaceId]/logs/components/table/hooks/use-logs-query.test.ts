import { trpc } from "@/lib/trpc/client";
import { act, renderHook } from "@testing-library/react";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useRatelimitLogsQuery } from "./use-logs-query";

let mockFilters: any[] = [];
const mockDate = 1706024400000;

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
        ratelimit: {
          logs: {
            query: {
              fetch,
            },
          },
        },
      }),
      ratelimit: {
        logs: {
          query: {
            useInfiniteQuery,
          },
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

describe("useRatelimitLogsQuery filter processing", () => {
  beforeEach(() => {
    mockFilters = [];
    vi.setSystemTime(mockDate);
  });

  it("handles valid status filter", () => {
    mockFilters = [{ field: "status", operator: "is", value: "rejected" }];
    const { result } = renderHook(() => useRatelimitLogsQuery({ namespaceId: "test-namespace" }));
    expect(result.current.isPolling).toBe(false);
  });

  it("handles multiple valid filters", () => {
    mockFilters = [
      { field: "status", operator: "is", value: "succeeded" },
      { field: "identifiers", operator: "is", value: "test-id" },
      { field: "requestIds", operator: "is", value: "req-123" },
    ];
    const { result } = renderHook(() => useRatelimitLogsQuery({ namespaceId: "test-namespace" }));
    expect(result.current.isPolling).toBe(false);
  });

  it("handles invalid filter types", () => {
    const consoleMock = vi.spyOn(console, "error");
    mockFilters = [
      { field: "identifiers", operator: "is", value: 123 },
      { field: "requestIds", operator: "is", value: true },
      { field: "status", operator: "is", value: {} },
    ];
    renderHook(() => useRatelimitLogsQuery({ namespaceId: "test-namspace" }));
    expect(consoleMock).toHaveBeenCalledTimes(3);
  });

  it("handles time-based filters", () => {
    mockFilters = [
      { field: "startTime", operator: "is", value: mockDate - 3600000 },
      { field: "since", operator: "is", value: "1h" },
    ];
    const { result } = renderHook(() => useRatelimitLogsQuery({ namespaceId: "test-namespace" }));
    expect(result.current.isPolling).toBe(false);
  });
});

describe("useRatelimitLogsQuery realtime logs", () => {
  let useInfiniteQuery: ReturnType<typeof vi.fn>;
  let fetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.setSystemTime(mockDate);
    mockFilters = [];
    //@ts-expect-error hacky way to mock trpc
    useInfiniteQuery = vi.mocked(trpc.ratelimit.logs.query.useInfiniteQuery);
    //@ts-expect-error hacky way to mock trpc
    fetch = vi.mocked(trpc.useUtils().ratelimit.logs.query.fetch);
  });

  it("resets realtime logs when polling stops", async () => {
    const mockLogs: Partial<RatelimitLog>[] = [
      {
        request_id: "1",
        time: Date.now(),
        status: 1,
        identifier: "test-1",
      },
      {
        request_id: "2",
        time: Date.now(),
        status: 0,
        identifier: "test-2",
      },
    ];

    useInfiniteQuery.mockReturnValue({
      data: {
        pages: [{ ratelimitLogs: mockLogs, nextCursor: null }],
      },
      hasNextPage: false,
      fetchNextPage: vi.fn(),
      isFetchingNextPage: false,
      isLoading: false,
    });

    fetch.mockResolvedValue({
      ratelimitLogs: [
        {
          request_id: "3",
          time: Date.now(),
          status: "succeeded",
          identifier: "test-3",
        },
      ],
    });

    const { result, rerender } = renderHook(
      ({ startPolling, pollIntervalMs }) =>
        useRatelimitLogsQuery({
          startPolling,
          pollIntervalMs,
          namespaceId: "test-namespace",
        }),
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
