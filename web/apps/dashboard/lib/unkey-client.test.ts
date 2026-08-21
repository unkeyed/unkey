import type { BaseError } from "@unkey/api/models/components";
import {
  ConnectionError,
  ForbiddenErrorResponse,
  PreconditionFailedErrorResponse,
} from "@unkey/api/models/errors";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getErrorMessage, getErrorToast, getUnkeyClient, noRetry } from "./unkey-client";

function makeErrorResponse<T>(
  Ctor: new (
    data: { error: BaseError; meta: { requestId: string } },
    httpMeta: { body: string; request: Request; response: Response },
  ) => T,
  error: BaseError,
): T {
  return new Ctor(
    { error, meta: { requestId: "req_123" } },
    {
      body: "",
      request: new Request("https://api.unkey.com/v2/apps.deleteApp"),
      response: new Response(null, { status: error.status }),
    },
  );
}

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

  it("sends a deployment create once when the API returns 500", async () => {
    const requests = stubServerError();

    await expect(
      getUnkeyClient().deployments.createDeployment(
        {
          project: "proj_kebap",
          app: "app_kebap",
          environment: "production",
          image: { dockerImage: "kebap:latest" },
        },
        noRetry,
      ),
    ).rejects.toThrow();

    expect(requests).toHaveLength(1);
  });

  it("retries a 500 when a backoff strategy is configured", async () => {
    const requests = stubServerError();

    await expect(
      getUnkeyClient().deployments.createDeployment(
        {
          project: "proj_kebap",
          app: "app_kebap",
          environment: "production",
          image: { dockerImage: "kebap:latest" },
        },
        {
          retries: {
            strategy: "backoff",
            backoff: { initialInterval: 1, maxInterval: 2, exponent: 1, maxElapsedTime: 20 },
            retryConnectionErrors: true,
          },
        },
      ),
    ).rejects.toThrow();

    expect(requests.length).toBeGreaterThan(1);
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

describe("getErrorToast", () => {
  it("recognizes delete protection", () => {
    const error = makeErrorResponse(PreconditionFailedErrorResponse, {
      detail: "This app has delete protection enabled. Disable it before attempting to delete.",
      status: 412,
      title: "Precondition Failed",
      type: "https://unkey.com/docs/errors/unkey/application/protected_resource",
    });

    expect(getErrorToast(error, "Failed to Delete App")).toEqual({
      message: "Delete Protection Enabled",
      description:
        "This app has delete protection enabled. Disable it before attempting to delete.",
    });
  });

  it("titles by error class and keeps the API detail", () => {
    const error = makeErrorResponse(ForbiddenErrorResponse, {
      detail: "Missing one of these permissions: app.*.delete_app",
      status: 403,
      title: "Insufficient Permissions",
      type: "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions",
    });

    expect(getErrorToast(error, "Failed to Delete App")).toEqual({
      message: "Permission Denied",
      description: "Missing one of these permissions: app.*.delete_app",
    });
  });

  it("reports network failures as connection problems", () => {
    const error = new ConnectionError("fetch failed", { cause: new TypeError("fetch failed") });

    expect(getErrorToast(error, "Failed to Delete App")).toEqual({
      message: "Connection Problem",
      description: "Check your internet connection and try again.",
    });
  });

  it("falls back to the operation title for unrecognized errors", () => {
    expect(getErrorToast(new Error("Network request failed"), "Failed to Delete App")).toEqual({
      message: "Failed to Delete App",
      description: "An unexpected error occurred. Please try again later.",
    });
  });
});

// Counts the requests the SDK actually put on the wire while every attempt
// fails, so a test can tell one attempt from a retry storm.
function stubServerError(): Request[] {
  const requests: Request[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      requests.push(input instanceof Request ? input : new Request(input, init));
      return new Response(
        JSON.stringify({
          meta: { requestId: "req_123" },
          error: {
            detail: "Something went wrong.",
            status: 500,
            title: "Internal Server Error",
            type: "https://unkey.com/docs/errors/unkey/application/internal_server_error",
          },
        }),
        { status: 500, headers: { "Content-Type": "application/json" } },
      );
    }),
  );
  return requests;
}
