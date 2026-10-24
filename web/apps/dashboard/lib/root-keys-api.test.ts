import type { V2RootKeysListKeysResponseBody } from "@unkey/api/models/components";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { type V2RootKey, listRootKeys } from "./root-keys-api";

const listKeys = vi.hoisted(() => vi.fn());

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ rootKeys: { listKeys } }),
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
): { result: V2RootKeysListKeysResponseBody } => ({
  result: { meta: { requestId: "req" }, data, pagination },
});

describe("listRootKeys", () => {
  beforeEach(() => {
    listKeys.mockReset();
  });

  it("follows cursors until hasMore is false", async () => {
    const signal = new AbortController().signal;
    listKeys
      .mockResolvedValueOnce(page([rootKey("key_1")], { cursor: "next", hasMore: true }))
      .mockResolvedValueOnce(page([rootKey("key_2")], { cursor: "ignored", hasMore: false }));

    const rootKeys = await listRootKeys(signal);

    expect(rootKeys.map((key) => key.keyId)).toEqual(["key_1", "key_2"]);
    expect(listKeys.mock.calls).toEqual([
      [{ limit: 100, cursor: undefined }, { signal }],
      [{ limit: 100, cursor: "next" }, { signal }],
    ]);
  });

  it("rejects another page without a cursor", async () => {
    listKeys.mockResolvedValueOnce(page([], { hasMore: true }));

    await expect(listRootKeys()).rejects.toThrow("without a cursor");
    expect(listKeys).toHaveBeenCalledTimes(1);
  });

  it("rejects a repeated cursor", async () => {
    listKeys.mockResolvedValue(page([], { cursor: "same-cursor", hasMore: true }));

    await expect(listRootKeys()).rejects.toThrow("repeated cursor");
    expect(listKeys).toHaveBeenCalledTimes(2);
  });
});
