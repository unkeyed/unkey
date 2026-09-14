import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, createElement } from "react";
import { afterEach, expect, it, vi } from "vitest";
import { useCreateLogdrainMutation, useLogdrains } from "./logdrains-query";

const api = vi.hoisted(() => ({ listLogdrains: vi.fn(), createLogdrain: vi.fn() }));
vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ id: "ws_1" }),
}));
vi.mock("@/lib/unkey-client", () => ({ getUnkeyClient: () => ({ logdrains: api }) }));

afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});

it("loads every page before exposing the workspace drain count", async () => {
  api.listLogdrains
    .mockResolvedValueOnce({
      data: [{ id: "ld_a" }],
      pagination: { hasMore: true, cursor: "ld_a" },
    })
    .mockResolvedValueOnce({ data: [{ id: "ld_b" }], pagination: { hasMore: false } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: PropsWithChildren) =>
    createElement(QueryClientProvider, { client }, children);
  const { result } = renderHook(() => useLogdrains(), { wrapper });
  await waitFor(() => expect(result.current.data).toEqual([{ id: "ld_a" }, { id: "ld_b" }]));
  expect(api.listLogdrains).toHaveBeenNthCalledWith(
    2,
    { limit: 100, cursor: "ld_a" },
    { signal: expect.any(AbortSignal) },
  );
  client.clear();
});

it("refreshes a list that was still loading when a drain was created", async () => {
  api.listLogdrains
    .mockImplementationOnce(
      (_input, { signal }: { signal: AbortSignal }) =>
        new Promise((_resolve, reject) => {
          signal.addEventListener(
            "abort",
            () => reject(new DOMException("Aborted", "AbortError")),
            { once: true },
          );
        }),
    )
    .mockResolvedValueOnce({ data: [{ id: "ld_created" }], pagination: { hasMore: false } });
  api.createLogdrain.mockResolvedValueOnce({ data: { id: "ld_created" } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: PropsWithChildren) =>
    createElement(QueryClientProvider, { client }, children);
  const { result } = renderHook(
    () => ({
      list: useLogdrains(),
      create: useCreateLogdrainMutation({ onSuccess: vi.fn(), onError: vi.fn() }),
    }),
    { wrapper },
  );
  await waitFor(() => expect(result.current.list.isFetching).toBe(true));
  result.current.create.mutate({
    name: "Logs",
    stream: "audit_logs",
    destination: { http: { url: "https://logs.example.com" } },
  });
  await waitFor(() => expect(result.current.list.data).toEqual([{ id: "ld_created" }]));
  client.clear();
});

it("does not expose a partial count when an API page has no continuation cursor", async () => {
  api.listLogdrains.mockResolvedValueOnce({
    data: [{ id: "ld_a" }],
    pagination: { hasMore: true },
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: PropsWithChildren) =>
    createElement(QueryClientProvider, { client }, children);
  const { result } = renderHook(() => useLogdrains(), { wrapper });
  await waitFor(() => expect(result.current.isError).toBe(true));
  expect(result.current.data).toBeUndefined();
  client.clear();
});
