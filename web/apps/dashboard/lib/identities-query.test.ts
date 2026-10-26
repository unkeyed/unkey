import { queryKeys } from "@/lib/query-keys";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, createElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCreateIdentityMutation } from "./identities-query";

const identityApi = vi.hoisted(() => ({
  createIdentity: vi.fn(),
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ id: "ws_1" }),
}));

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ identities: identityApi }),
}));

describe("identity query cache", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
  });

  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("scopes list and detail keys to the workspace", () => {
    expect(queryKeys.identities.list("ws_1", "alice")).toEqual([
      "identities",
      "ws_1",
      "list",
      "alice",
    ]);
    expect(queryKeys.identities.detail("ws_2", "id_1")).toEqual([
      "identities",
      "ws_2",
      "detail",
      "id_1",
    ]);
  });

  it("refetches a list canceled during identity creation", async () => {
    let requestCount = 0;
    identityApi.createIdentity.mockResolvedValueOnce({ data: { identityId: "id_new" } });

    function wrapper({ children }: PropsWithChildren) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }

    const { result } = renderHook(
      () => {
        const list = useQuery({
          queryKey: queryKeys.identities.list("ws_1", ""),
          queryFn: ({ signal }) => {
            requestCount += 1;
            if (requestCount > 1) {
              return Promise.resolve("fresh");
            }
            return new Promise<string>((_resolve, reject) => {
              if (!signal) {
                reject(new Error("Expected React Query to provide an abort signal"));
                return;
              }
              signal.addEventListener(
                "abort",
                () => reject(new DOMException("Request aborted", "AbortError")),
                { once: true },
              );
            });
          },
        });
        const mutation = useCreateIdentityMutation();
        return { list, mutation };
      },
      { wrapper },
    );

    await waitFor(() => expect(result.current.list.isFetching).toBe(true));
    await act(async () => {
      await result.current.mutation.mutateAsync({ externalId: "new_identity" });
    });
    await waitFor(() => expect(result.current.list.data).toBe("fresh"));
    expect(requestCount).toBe(2);
  });

  it("refetches an inactive list after identity creation", async () => {
    const queryKey = queryKeys.identities.list("ws_1", "");
    let response = "stale";
    const queryFn = vi.fn(() => Promise.resolve(response));
    await queryClient.fetchQuery({ queryKey, queryFn });
    response = "fresh";
    identityApi.createIdentity.mockResolvedValueOnce({ data: { identityId: "id_new" } });

    function wrapper({ children }: PropsWithChildren) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }

    const { result } = renderHook(() => useCreateIdentityMutation(), { wrapper });
    await act(async () => {
      await result.current.mutateAsync({ externalId: "new_identity" });
    });

    expect(queryClient.getQueryData(queryKey)).toBe("fresh");
    expect(queryFn).toHaveBeenCalledTimes(2);
  });
});
