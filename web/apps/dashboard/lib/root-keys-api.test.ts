import { Unkey } from "@unkey/api";
import type { V2RootKeysListKeysResponseBody } from "@unkey/api/models/components";
import { afterEach, describe, expect, it, vi } from "vitest";
import { type V2RootKey, listRootKeys } from "./root-keys-api";

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => new Unkey({ serverURL: "http://dashboard.test/proxy/" }),
}));

const rootKey = (keyId: string): V2RootKey => ({
  keyId,
  name: null,
  start: "unkey_abc",
  end: "xyz",
  enabled: true,
  createdAt: 1,
  lastUsedAt: 0,
  expires: null,
  permissions: [],
});

const page = (
  data: V2RootKey[],
  pagination: V2RootKeysListKeysResponseBody["pagination"],
): Response => {
  const body: V2RootKeysListKeysResponseBody = { meta: { requestId: "req" }, data, pagination };
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
};

const stubFetch = () => {
  const fetch = vi.fn<typeof globalThis.fetch>();
  vi.stubGlobal("fetch", fetch);
  return fetch;
};

const requestBodies = (fetch: ReturnType<typeof stubFetch>) =>
  Promise.all(
    fetch.mock.calls.map(async ([input]) => {
      if (!(input instanceof Request)) {
        throw new Error("Expected the SDK to call fetch with a Request");
      }
      expect(input.url).toBe("http://dashboard.test/proxy/v2/rootKeys.listKeys");
      return input.json();
    }),
  );

describe("listRootKeys", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("follows cursors until hasMore is false", async () => {
    const fetch = stubFetch()
      .mockResolvedValueOnce(page([rootKey("key_1")], { cursor: "next", hasMore: true }))
      .mockResolvedValueOnce(page([rootKey("key_2")], { cursor: "ignored", hasMore: false }));

    const rootKeys = await listRootKeys();

    expect(rootKeys.map((key) => key.keyId)).toEqual(["key_1", "key_2"]);
    await expect(requestBodies(fetch)).resolves.toEqual([
      { limit: 100 },
      { limit: 100, cursor: "next" },
    ]);
  });

  it("rejects another page without a cursor", async () => {
    const fetch = stubFetch().mockResolvedValueOnce(page([], { hasMore: true }));

    await expect(listRootKeys()).rejects.toThrow("without a cursor");
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("rejects a repeated cursor", async () => {
    const fetch = stubFetch().mockImplementation(async () =>
      page([], { cursor: "same-cursor", hasMore: true }),
    );

    await expect(listRootKeys()).rejects.toThrow("repeated cursor");
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
