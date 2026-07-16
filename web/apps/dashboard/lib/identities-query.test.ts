import {
  type InfiniteData,
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { Identity } from "@unkey/api/models/components";
import { type PropsWithChildren, createElement } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  type IdentityPage,
  identityQueryKeys,
  removeIdentityFromPages,
  replaceIdentityInPages,
  useCreateIdentityMutation,
} from "./identities-query";

const identityApi = vi.hoisted(() => ({
  createIdentity: vi.fn(),
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ id: "ws_1" }),
}));

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ identities: identityApi }),
}));

function identity(id: string): Identity {
  return { id, externalId: `external-${id}` };
}

function infiniteData(...pages: IdentityPage[]): InfiniteData<IdentityPage> {
  return { pages, pageParams: pages.map((_, index) => index) };
}

describe("identity query cache", () => {
  it("scopes list and detail keys to the workspace", () => {
    expect(identityQueryKeys.list("ws_1", "alice")).toEqual([
      "identities",
      "ws_1",
      "list",
      "alice",
    ]);
    expect(identityQueryKeys.detail("ws_2", "id_1")).toEqual([
      "identities",
      "ws_2",
      "detail",
      "id_1",
    ]);
  });

  it("refetches a list canceled during identity creation", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    let requestCount = 0;
    identityApi.createIdentity.mockResolvedValueOnce({ data: { identityId: "id_new" } });

    function wrapper({ children }: PropsWithChildren) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }

    const { result } = renderHook(
      () => {
        const list = useQuery({
          queryKey: identityQueryKeys.list("ws_1", ""),
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
    await result.current.mutation.mutateAsync({ externalId: "new_identity" });
    await waitFor(() => expect(result.current.list.data).toBe("fresh"));
    expect(requestCount).toBe(2);
  });

  it("replaces an identity across cached pages without changing cursors", () => {
    const data = infiniteData(
      { identities: [identity("id_1")], cursor: "next" },
      { identities: [identity("id_2")] },
    );
    const updated = { ...identity("id_2"), externalId: "updated" };

    expect(
      replaceIdentityInPages(data, updated, { identity: updated.id, meta: updated.meta }),
    ).toEqual(
      infiniteData({ identities: [identity("id_1")], cursor: "next" }, { identities: [updated] }),
    );
  });

  it("preserves fields omitted from a partial update response", () => {
    const existing = {
      ...identity("id_1"),
      meta: { plan: "pro" },
      ratelimits: [{ id: "rl_1", name: "requests", limit: 10, duration: 60, autoApply: true }],
    };
    const response = { ...existing, meta: {} };

    expect(
      replaceIdentityInPages(infiniteData({ identities: [existing] }), response, {
        identity: existing.id,
        ratelimits: response.ratelimits,
      }).pages[0]?.identities[0]?.meta,
    ).toEqual({ plan: "pro" });
  });

  it("removes an identity from every cached page", () => {
    const data = infiniteData(
      { identities: [identity("id_1"), identity("id_2")], cursor: "next" },
      { identities: [identity("id_2"), identity("id_3")] },
    );

    expect(removeIdentityFromPages(data, "id_2")).toEqual(
      infiniteData(
        { identities: [identity("id_1")], cursor: "next" },
        { identities: [identity("id_3")] },
      ),
    );
  });
});
