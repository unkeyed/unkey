import { trpc } from "@/lib/trpc/client";
import { renderHook } from "@testing-library/react";
import { type Mock, beforeEach, describe, expect, it, vi } from "vitest";
import {
  buildRequestDetailsQueryParams,
  useFetchRequestDetails,
} from "./use-fetch-request-details";

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    logs: {
      queryLogs: {
        useQuery: vi.fn(),
      },
    },
  },
}));

describe("buildRequestDetailsQueryParams", () => {
  const time = 1_757_872_504_000;

  it("brackets the window around the row's own time", () => {
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.startTime).toBeLessThan(time);
    expect(params.endTime).toBeGreaterThan(time);
    expect(params.requestId).toEqual({ filters: [{ operator: "is", value: "req_1" }] });
    expect(params.limit).toBe(1);
  });

  it("covers a row stamped after the page's query clock", () => {
    // The regression: live-tailed rows arrive with a time past the frozen
    // query clock, and an endTime taken from that clock excluded them forever.
    const staleQueryClock = time - 5 * 60_000;
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.endTime).toBeGreaterThan(staleQueryClock);
    expect(params.endTime).toBeGreaterThanOrEqual(time);
  });

  it("reaches back far enough for a request that started before its verification", () => {
    // api_requests rows are stamped at request start, verification rows
    // mid-handler; the API caps a request at a one minute timeout.
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.startTime).toBeLessThanOrEqual(time - 60_000);
  });

  it("never asks for a negative start time", () => {
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time: 0 });

    expect(params.startTime).toBe(0);
  });

  it("drops the request filter when there is no selected row", () => {
    expect(buildRequestDetailsQueryParams({}).requestId).toBeNull();
  });
});

describe("useFetchRequestDetails", () => {
  const time = 1_757_872_504_000;
  const maxAttempts = 6;
  const retryIntervalMs = 2_000;

  type QueryState = {
    data?: { logs: { request_id: string }[] };
    dataUpdatedAt: number;
    errorUpdatedAt: number;
    isSuccess: boolean;
    isError: boolean;
    isLoading: boolean;
    error: { message: string } | null;
  };

  const settledEmpty = (at: number): QueryState => ({
    data: { logs: [] },
    dataUpdatedAt: at,
    errorUpdatedAt: 0,
    isSuccess: true,
    isError: false,
    isLoading: false,
    error: null,
  });

  const settledFailed = (at: number): QueryState => ({
    data: undefined,
    dataUpdatedAt: 0,
    errorUpdatedAt: at,
    isSuccess: false,
    isError: true,
    isLoading: false,
    error: { message: "clickhouse unavailable" },
  });

  const settledWithLog = (at: number): QueryState => ({
    data: { logs: [{ request_id: "req_1" }] },
    dataUpdatedAt: at,
    errorUpdatedAt: 0,
    isSuccess: true,
    isError: false,
    isLoading: false,
    error: null,
  });

  const useQueryMock = trpc.logs.queryLogs.useQuery as unknown as Mock;

  let queryState: QueryState;
  let lastOptions: { refetchInterval: number | false } | undefined;

  const renderRequestDetails = () =>
    renderHook(() => useFetchRequestDetails({ requestId: "req_1", time }));

  beforeEach(() => {
    lastOptions = undefined;
    useQueryMock.mockReset();
    useQueryMock.mockImplementation(
      (_params: unknown, options: { refetchInterval: number | false }) => {
        lastOptions = options;
        return queryState;
      },
    );
  });

  it("keeps polling while the log has not been ingested yet", () => {
    queryState = settledEmpty(1);
    const { result } = renderRequestDetails();

    expect(lastOptions?.refetchInterval).toBe(retryIntervalMs);
    expect(result.current.isLoading).toBe(true);
  });

  it("stops polling once failed refetches exhaust the attempt budget", () => {
    // A failed refetch leaves data and dataUpdatedAt alone, so counting only
    // successful settles pinned the budget mid-way and polled forever.
    queryState = settledEmpty(1);
    const { result, rerender } = renderRequestDetails();

    expect(lastOptions?.refetchInterval).toBe(retryIntervalMs);
    expect(result.current.isLoading).toBe(true);

    for (let attempt = 2; attempt <= maxAttempts; attempt++) {
      queryState = settledFailed(attempt);
      rerender();
    }

    expect(lastOptions?.refetchInterval).toBe(false);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toEqual({ message: "clickhouse unavailable" });
  });

  it("settles as soon as the log turns up after a failed refetch", () => {
    queryState = settledFailed(1);
    const { result, rerender } = renderRequestDetails();

    queryState = settledWithLog(2);
    rerender();

    expect(lastOptions?.refetchInterval).toBe(false);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.log).toEqual({ request_id: "req_1" });
  });
});
