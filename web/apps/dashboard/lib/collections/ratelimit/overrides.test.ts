import { and, createLiveQueryCollection, eq } from "@tanstack/react-db";
import { describe, expect, it, vi } from "vitest";

const pages = [
  {
    result: { data: [{ overrideId: "ovr_1", identifier: "user_1", limit: 10, duration: 60_000 }] },
  },
  { result: { data: [{ overrideId: "ovr_2", identifier: "KEBAP", limit: 5, duration: 1000 }] } },
];
const listOverrides = vi.fn(async (_: { namespace: string; limit: number }) => ({
  ...pages[0],
  async *[Symbol.asyncIterator]() {
    yield* pages;
  },
}));

const { NotFound } = vi.hoisted(() => ({ NotFound: class NotFound extends Error {} }));
const getOverride = vi.fn(async ({ identifier }: { namespace: string; identifier: string }) => {
  if (identifier === "missing") {
    throw new NotFound();
  }
  if (identifier === "alice@KEBAP.com") {
    return {
      data: { overrideId: "ovr_wildcard", identifier: "*@KEBAP.com", limit: 3, duration: 1000 },
    };
  }
  return { data: { overrideId: "ovr_9", identifier, limit: 3, duration: 1000 } };
});

vi.mock("@unkey/api/models/errors", () => ({ NotFoundErrorResponse: NotFound }));
vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ ratelimit: { listOverrides, getOverride } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { ratelimitOverrides } = await import("./overrides");

describe("ratelimit overrides collection", () => {
  it("lists every page of one namespace and stamps its id on the rows", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ override: ratelimitOverrides })
        .where(({ override }) => eq(override.namespaceId, "rlns_1")),
    );
    await query.preload();

    await vi.waitFor(() => expect(query.toArray).toHaveLength(2));
    expect(query.toArray.map((o) => [o.id, o.namespaceId, o.identifier]).sort()).toEqual([
      ["ovr_1", "rlns_1", "user_1"],
      ["ovr_2", "rlns_1", "KEBAP"],
    ]);
    expect(listOverrides.mock.calls.map(([c]) => c)).toEqual([{ namespace: "rlns_1", limit: 100 }]);
  });

  it("loads a single override with getOverride instead of the whole namespace", async () => {
    listOverrides.mockClear();
    const query = createLiveQueryCollection((q) =>
      q
        .from({ override: ratelimitOverrides })
        .where(({ override }) =>
          and(eq(override.namespaceId, "rlns_2"), eq(override.identifier, "KEBAP")),
        ),
    );
    await query.preload();

    await vi.waitFor(() => expect(query.toArray).toHaveLength(1));
    expect(query.toArray[0]).toEqual({
      id: "ovr_9",
      namespaceId: "rlns_2",
      identifier: "KEBAP",
      limit: 3,
      duration: 1000,
    });
    expect(getOverride).toHaveBeenCalledWith({ namespace: "rlns_2", identifier: "KEBAP" });
    expect(listOverrides).not.toHaveBeenCalled();
  });

  it("treats a missing override as no rows", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ override: ratelimitOverrides })
        .where(({ override }) =>
          and(eq(override.namespaceId, "rlns_2"), eq(override.identifier, "missing")),
        ),
    );

    expect(await query.toArrayWhenReady()).toEqual([]);
  });

  it("keeps a wildcard match out of the collection when asked for an exact identifier", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ override: ratelimitOverrides })
        .where(({ override }) =>
          and(eq(override.namespaceId, "rlns_3"), eq(override.identifier, "alice@KEBAP.com")),
        ),
    );

    expect(await query.toArrayWhenReady()).toEqual([]);
    expect(getOverride).toHaveBeenCalledWith({
      namespace: "rlns_3",
      identifier: "alice@KEBAP.com",
    });
    expect(ratelimitOverrides.has("ovr_wildcard")).toBe(false);
  });
});
