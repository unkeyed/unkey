import { ForbiddenErrorResponse } from "@unkey/api/models/errors";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getErrorMessage, getUnkeyClient } from "./unkey-client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("getUnkeyClient", () => {
  it("sends SDK requests through the dashboard proxy", async () => {
    const requests: Request[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        requests.push(input instanceof Request ? input : new Request(input, init));
        return new Response(
          JSON.stringify({
            meta: { requestId: "req_123" },
            data: {},
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );

    await getUnkeyClient().keys.updateKey({ keyId: "key_1234abcd" });

    expect(requests).toHaveLength(1);
    const request = requests[0];
    if (!request) {
      throw new Error("Expected the SDK to make a request");
    }
    expect(request.url).toBe("http://localhost:3000/proxy/v2/keys.updateKey");
    expect(request.method).toBe("POST");
    expect(request.headers.has("Authorization")).toBe(false);
    await expect(request.clone().json()).resolves.toEqual({ keyId: "key_1234abcd" });
  });
});

describe("getErrorMessage", () => {
  it("returns the SDK error detail", () => {
    const error = new ForbiddenErrorResponse(
      {
        error: {
          detail: "Missing one of these permissions: api.*.update_key",
          status: 403,
          title: "Insufficient Permissions",
          type: "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
        },
        meta: {
          requestId: "req_123",
        },
      },
      {
        body: "",
        request: new Request("https://api.unkey.com/v2/keys.updateKey"),
        response: new Response(null, { status: 403 }),
      },
    );

    expect(getErrorMessage(error)).toBe("Missing one of these permissions: api.*.update_key");
  });

  it("returns the fallback for non-SDK errors", () => {
    expect(getErrorMessage(new Error("Network request failed"), "Try again")).toBe("Try again");
  });
});
