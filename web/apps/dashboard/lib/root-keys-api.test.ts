import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createRootKey,
  deleteRootKey,
  listRootKeys,
  rerollRootKey,
  updateRootKey,
} from "./root-keys-api";

const response = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });

describe("root keys API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists every cursor page through the authenticated proxy", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(
        response({
          data: [
            {
              keyId: "key_1",
              name: "First",
              start: "unkey_abc",
              end: "xyz",
              enabled: true,
              createdAt: 1,
              lastUsedAt: 0,
              expires: null,
              permissions: ["unkey:v1:ws_1:apis/*#read_api"],
            },
          ],
          pagination: { cursor: "next", hasMore: true },
        }),
      )
      .mockResolvedValueOnce(
        response({
          data: [
            {
              keyId: "key_2",
              name: "Second",
              start: "unkey_def",
              end: "uvw",
              enabled: false,
              createdAt: 2,
              lastUsedAt: 3,
              expires: 4,
              permissions: [],
            },
          ],
          pagination: { hasMore: false },
        }),
      );
    vi.stubGlobal("fetch", fetch);

    await expect(listRootKeys()).resolves.toHaveLength(2);
    expect(fetch).toHaveBeenNthCalledWith(
      1,
      "/proxy/v2/rootKeys.listKeys",
      expect.objectContaining({ body: JSON.stringify({ limit: 100 }) }),
    );
    expect(fetch).toHaveBeenNthCalledWith(
      2,
      "/proxy/v2/rootKeys.listKeys",
      expect.objectContaining({ body: JSON.stringify({ limit: 100, cursor: "next" }) }),
    );
  });

  it("writes root keys through the v2 endpoints", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(response({ data: { keyId: "key_1", key: "unkey_secret" } }))
      .mockResolvedValueOnce(response({ meta: { requestId: "req_1" } }))
      .mockResolvedValueOnce(response({ meta: { requestId: "req_2" } }))
      .mockResolvedValueOnce(response({ data: { keyId: "key_2", key: "unkey_rotated" } }));
    vi.stubGlobal("fetch", fetch);

    await expect(
      createRootKey({
        name: "Production",
        permissions: ["unkey:v1:ws_1:apis/*#read_api"],
      }),
    ).resolves.toEqual({ keyId: "key_1", key: "unkey_secret" });
    await updateRootKey({ keyId: "key_1", name: "Renamed", permissions: [] });
    await deleteRootKey({ keyId: "key_1" });
    await expect(rerollRootKey({ keyId: "key_1", expiration: null })).resolves.toEqual({
      keyId: "key_2",
      key: "unkey_rotated",
    });

    expect(fetch.mock.calls.map(([url]) => url)).toEqual([
      "/proxy/v2/rootKeys.createKey",
      "/proxy/v2/rootKeys.updateKey",
      "/proxy/v2/rootKeys.deleteKey",
      "/proxy/v2/rootKeys.rerollKey",
    ]);
    expect(fetch.mock.calls.map(([, init]) => init?.body)).toEqual([
      JSON.stringify({
        name: "Production",
        permissions: ["unkey:v1:ws_1:apis/*#read_api"],
      }),
      JSON.stringify({ keyId: "key_1", name: "Renamed", permissions: [] }),
      JSON.stringify({ keyId: "key_1" }),
      JSON.stringify({ keyId: "key_1", expiration: null }),
    ]);
  });

  it("surfaces API error details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof globalThis.fetch>().mockResolvedValue(
        new Response(JSON.stringify({ error: { detail: "You cannot grant this permission." } }), {
          status: 403,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    await expect(deleteRootKey({ keyId: "key_1" })).rejects.toThrow(
      "You cannot grant this permission.",
    );
  });

  it("rejects a repeated pagination cursor", async () => {
    const page = {
      data: [],
      pagination: { cursor: "same-cursor", hasMore: true },
    };
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(response(page))
      .mockResolvedValueOnce(response(page));
    vi.stubGlobal("fetch", fetch);

    await expect(listRootKeys()).rejects.toThrow("repeated cursor");
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
