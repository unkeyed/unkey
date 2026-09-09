import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { Portal } from "@unkey/api/models/components";
import type { ReactNode } from "react";
import { createElement } from "react";
import { describe, expect, it, vi } from "vitest";

const getPortalByKeyspace = vi.fn();

vi.mock("./client", () => ({
  getPortalByKeyspace: (...args: unknown[]) => getPortalByKeyspace(...args),
}));

vi.mock("@/lib/unkey-client", () => ({
  getErrorMessage: (error: unknown) => (error instanceof Error ? error.message : "unknown"),
  getErrorToast: () => ({ message: "", description: "" }),
  getUnkeyClient: () => ({ portal: {} }),
}));

vi.mock("@unkey/ui", () => ({
  toast: { error: vi.fn() },
}));

const { portalQueryKey, usePortal } = await import("./use-portal");

const portal: Portal = {
  id: "portal_123",
  slug: "acme",
  displayName: "Acme",
  enabled: true,
  keyspaceId: "ks_123",
  createdAt: 0,
};

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

describe("usePortal", () => {
  it("keeps rendering the cached portal when a background refetch fails", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(portalQueryKey("ks_123"), { found: true, portal });
    getPortalByKeyspace.mockRejectedValue(new Error("upstream exploded"));

    const { result } = renderHook(() => usePortal("ks_123"), { wrapper: wrapperFor(client) });

    // The refetch fires immediately because the seeded row is already stale.
    await waitFor(() => {
      expect(client.getQueryState(portalQueryKey("ks_123"))?.error).toBeTruthy();
    });

    // A single failed focus/reconnect refetch must not unmount the
    // configuration view and discard in-progress edits.
    expect(result.current).toEqual({ status: "enabled", portal });
  });

  it("surfaces the error when there is no cached row to fall back on", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    getPortalByKeyspace.mockRejectedValue(new Error("upstream exploded"));

    const { result } = renderHook(() => usePortal("ks_456"), { wrapper: wrapperFor(client) });

    await waitFor(() => {
      expect(result.current).toEqual({ status: "error", message: "upstream exploded" });
    });
  });
});
