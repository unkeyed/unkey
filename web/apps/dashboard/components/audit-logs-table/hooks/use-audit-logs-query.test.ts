import { trpc } from "@/lib/trpc/client";
import { renderHook } from "@testing-library/react";
import { useCallback, useState } from "react";
import { describe, expect, it, vi } from "vitest";

// Controllable, in-memory stand-in for the URL query string, so `page` can be
// driven without nuqs' throttle timers. nuqs is otherwise kept real for the
// parser used by the shared pagination primitives.
const urlStore: Record<string, unknown> = {};

vi.mock("nuqs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("nuqs")>();
  const useQueryState = (key: string, parser?: { defaultValue?: unknown }) => {
    const [, force] = useState(0);
    const read = () =>
      Object.prototype.hasOwnProperty.call(urlStore, key)
        ? urlStore[key]
        : (parser?.defaultValue ?? null);
    // biome-ignore lint/correctness/useExhaustiveDependencies: test-only fake; `read` closes over the stable `key`/`parser`.
    const setValue = useCallback(
      (next: unknown) => {
        urlStore[key] = typeof next === "function" ? next(read()) : next;
        force((n) => n + 1);
        return Promise.resolve(true);
      },
      [key],
    );
    return [read(), setValue];
  };
  return { ...actual, useQueryState };
});

vi.mock("@/lib/trpc/client", () => {
  const useQuery = vi.fn();
  return {
    trpc: {
      useUtils: () => ({ audit: { logs: { prefetch: vi.fn() } } }),
      audit: { logs: { useQuery } },
    },
  };
});

vi.mock("@/app/(app)/[workspaceSlug]/audit/hooks/use-filters", () => ({
  useFilters: () => ({ filters: [] }),
}));

import { useAuditLogsQuery } from "./use-audit-logs-query";

const mockedUseQuery = vi.mocked(trpc.audit.logs.useQuery);

const rows = (count: number) => Array.from({ length: count }, (_, i) => ({ id: `log-${i}` }));

describe("useAuditLogsQuery pagination", () => {
  it("keeps pagination usable when the count query fails (total null)", () => {
    mockedUseQuery.mockReturnValue({
      data: { auditLogs: rows(50), total: null },
      isLoading: false,
      isFetching: false,
    } as never);

    const { result } = renderHook(() => useAuditLogsQuery());

    expect(result.current.totalCount).toBe(50);
    expect(result.current.totalPages).toBe(2);
  });

  it("uses the exact count when the server returns one", () => {
    mockedUseQuery.mockReturnValue({
      data: { auditLogs: rows(50), total: 120 },
      isLoading: false,
      isFetching: false,
    } as never);

    const { result } = renderHook(() => useAuditLogsQuery());

    expect(result.current.totalCount).toBe(120);
    expect(result.current.totalPages).toBe(3);
  });
});
