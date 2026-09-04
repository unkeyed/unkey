import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const state = vi.hoisted(() => ({
  useInfiniteQuery: vi.fn(),
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    alerts: {
      list: {
        useInfiniteQuery: state.useInfiniteQuery,
      },
    },
  },
}));

import { useAnomalyAlerts } from "./use-anomaly-alerts";

describe("useAnomalyAlerts", () => {
  beforeEach(() => {
    state.useInfiniteQuery.mockReset();
    state.useInfiniteQuery.mockReturnValue({
      data: {
        pages: [
          { alerts: [{ id: "alert_1" }], nextCursor: "cursor_1" },
          { alerts: [{ id: "alert_2" }], nextCursor: undefined },
        ],
      },
      hasNextPage: false,
    });
  });

  it("loads resolved alerts across cursor pages", () => {
    const { result } = renderHook(() =>
      useAnomalyAlerts({
        appId: "app_1",
        environmentId: "env_1",
        startMs: 1_000,
        endMs: 2_000,
        enabled: true,
      }),
    );

    expect(result.current.alerts.map((alert) => alert.id)).toEqual(["alert_1", "alert_2"]);
    expect(state.useInfiniteQuery).toHaveBeenCalledWith(
      {
        appId: "app_1",
        environmentId: "env_1",
        startMs: 1_000,
        endMs: 2_000,
        includeResolved: true,
        limit: 25,
      },
      expect.objectContaining({ enabled: true, staleTime: 30_000 }),
    );

    const options = state.useInfiniteQuery.mock.calls[0]?.[1];
    expect(options?.getNextPageParam({ nextCursor: "cursor_2" })).toBe("cursor_2");
  });
});
