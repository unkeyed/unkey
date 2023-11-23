import { Unkey, type UnkeyError } from "../src";
import { resetFetchMock, testOneFetchCall } from "./mock-fetch.test";
import { beforeEach, describe, expect, mock, test } from "bun:test";

describe("Unkey client", () => {
  test("default config", () => {
    const rootKey = "root key";
    const client = new Unkey({ rootKey });

    expect(client.baseUrl).toBe("https://api.unkey.dev");
    expect(client.retry.attempts).toBe(5);
    // @ts-expect-error
    expect(client.retry.backoff).toStrictEqual(expect.any(Function));
  });

  test("respects config args", () => {
    const baseUrl = "https://example.com/some/path";
    const rootKey = "root key";
    const retry = {
      attempts: 3,
      backoff(n: number) {
        return n * 100;
      },
    };
    const client = new Unkey({ baseUrl, rootKey, retry });

    expect(client.baseUrl).toBe(baseUrl);
    expect(client.retry.attempts).toBe(retry.attempts);
    expect(client.retry.backoff).toBe(retry.backoff);
  });

  test("throws for invalid rootKey", () => {
    // @ts-expect-error
    expect(() => new Unkey()).toThrow();
    // @ts-expect-error
    expect(() => new Unkey({ rootKey: null })).toThrow(
      "Unkey root key must be set, maybe you passed in `undefined` or an empty string?",
    );
    expect(() => new Unkey({ rootKey: "" })).toThrow(
      "Unkey root key must be set, maybe you passed in `undefined` or an empty string?",
    );
  });

  test("retry.attempts works", async () => {
    const client = new Unkey({
      rootKey: "root key",
      retry: {
        attempts: 3,
        backoff: () => 0,
      },
    });

    const error: UnkeyError = {
      code: "FETCH_ERROR",
      docs: "docs",
      message: "failure",
      requestId: "request id",
    };

    const mockFetch = mock(async () => ({
      ok: false,
      json: async () => error,
    }));
    // @ts-ignore
    globalThis.fetch = mockFetch;
    const response = await client.apis.get({ apiId: "apiId" });
    resetFetchMock();

    expect(response).toStrictEqual({ error });
    expect(mockFetch).toHaveBeenCalledTimes(4);
  });

  test("calls succeed if fetch fails some retries", async () => {
    const client = new Unkey({
      rootKey: "root key",
      retry: {
        attempts: 3,
        backoff: () => 0,
      },
    });

    const result = {
      id: "id",
      name: "name",
      workspaceId: "workspace id",
    };
    const mockFetch = mock(
      async () =>
        ({
          ok: true,
          json: async () => result,
        }) as {
          ok: boolean;
          json?: () => Promise<unknown>;
        },
    )
      .mockResolvedValueOnce({ ok: false })
      .mockResolvedValueOnce({ ok: false });
    // @ts-ignore
    globalThis.fetch = mockFetch;
    const response = await client.apis.get({ apiId: "apiId" });
    resetFetchMock();

    expect(response).toStrictEqual({ result });
    expect(mockFetch).toHaveBeenCalledTimes(3);
  });

  test("returns an error if fetch throws", async () => {
    const client = new Unkey({
      rootKey: "root key",
      retry: {
        attempts: 3,
        backoff: () => 0,
      },
    });

    const mockFetch = mock(() => Promise.reject({ message: "failure" }));
    // @ts-ignore
    globalThis.fetch = mockFetch;
    const response = await client.apis.get({ apiId: "apiId" });
    resetFetchMock();

    expect(response).toStrictEqual({
      error: {
        code: "FETCH_ERROR",
        message: "failure",
        docs: "https://developer.mozilla.org/en-US/docs/Web/API/fetch",
        requestId: "N/A",
      },
    });
    expect(mockFetch).toHaveBeenCalledTimes(4);
  });

  test("retry.backoff is called as expected on each failure", async () => {
    const backoff = mock((_n: number) => 0);
    const client = new Unkey({
      rootKey: "root key",
      retry: {
        attempts: 3,
        backoff,
      },
    });

    const mockFetch = mock(() => Promise.reject({ message: "failure" }));
    // @ts-ignore
    globalThis.fetch = mockFetch;
    const response = await client.apis.get({ apiId: "apiId" });
    resetFetchMock();

    expect(response).toStrictEqual({
      error: {
        code: "FETCH_ERROR",
        message: "failure",
        docs: "https://developer.mozilla.org/en-US/docs/Web/API/fetch",
        requestId: "N/A",
      },
    });
    expect(backoff.mock.calls).toEqual([[0], [1], [2], [3]]);
    expect(backoff).toHaveBeenCalledTimes(4);
    expect(mockFetch).toHaveBeenCalledTimes(4);
  });
});

describe("snapshot api calls", () => {
  beforeEach(() => resetFetchMock());
  const client = new Unkey({ rootKey: "root key" });

  describe("client.keys", () => {
    test("create", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/keys",
        method: "POST",
        jsonBody: {
          name: "key name",
          apiId: "key api id",
          prefix: "key prefix",
          byteLength: 100,
          ownerId: "owner id",
          meta: {
            metaKeyA: "a",
            metaKeyB: "b",
          },
          expires: 0,
          ratelimit: {
            type: "fast" as "fast",
            limit: 100,
            refillRate: 100,
            refillInterval: 100,
          },
          remaining: 100,
        },
        execute: (req) => client.keys.create(req),
      }));

    test("update", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/keys/keyId",
        method: "PUT",
        jsonBody: {
          keyId: "keyId",
          name: "key name",
          ownerId: "owner id",
          meta: {
            metaKeyA: "a",
            metaKeyB: "b",
          },
          expires: 0,
          ratelimit: {
            type: "fast" as "fast",
            limit: 100,
            refillRate: 100,
            refillInterval: 100,
          },
          remaining: 100,
        },
        execute: (req) => client.keys.update(req),
      }));

    test("verify", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/keys/verify",
        method: "POST",
        jsonBody: {
          key: "key",
          apiId: "api id",
        },
        execute: (req) => client.keys.verify(req),
      }));

    test("revoke", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/keys/keyId",
        method: "DELETE",
        execute: () => client.keys.revoke({ keyId: "keyId" }),
      }));
  });

  describe("client.apis", () => {
    test("update", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/apis.createApi",
        method: "POST",
        jsonBody: {
          name: "api name",
        },
        execute: (req) => client.apis.create(req),
      }));

    test("remove", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/apis.removeApi",
        method: "POST",
        jsonBody: {
          apiId: "api id",
        },
        execute: (req) => client.apis.remove(req),
      }));

    test("get", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/apis/apiId",
        method: "GET",
        execute: () => client.apis.get({ apiId: "apiId" }),
      }));

    test("listKeys", async () =>
      testOneFetchCall({
        url: "https://api.unkey.dev/v1/apis/apiId/keys?limit=10&offset=5&ownerId=owner+id",
        method: "GET",
        execute: () =>
          client.apis.listKeys({
            apiId: "apiId",
            limit: 10,
            offset: 5,
            ownerId: "owner id",
          }),
      }));

    describe("client._internal", () => {
      test("createRootKey", async () =>
        testOneFetchCall({
          url: "https://api.unkey.dev/v1/internal/rootkeys",
          method: "POST",
          jsonBody: {
            name: "root key name",
            expires: 0,
            forWorkspaceId: "workspace id",
          },
          execute: (req) => client._internal.createRootKey(req),
        }));

      test("deleteRootKey", async () =>
        testOneFetchCall({
          url: "https://api.unkey.dev/v1/internal.removeRootKey",
          method: "POST",
          jsonBody: {
            keyId: "key id",
          },
          execute: (req) => client._internal.deleteRootKey(req),
        }));
    });
  });
});
