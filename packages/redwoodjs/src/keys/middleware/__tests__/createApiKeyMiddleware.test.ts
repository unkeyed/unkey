import { createLogger } from "@redwoodjs/api/logger";
import { MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import { describe, expect, it, vi } from "vitest";
import createApiKeyMiddleware from "../createApiKeyMiddleware";
import type { ApiKeyMiddlewareConfig, VerifyResponse } from "../types";

const VALID_KEY = "valid-key";
const INVALID_KEY = "invalid-key";
const DEFAULT_API_ID = "defaultApiId";

/**
 * Mock Unkey key verification
 *
 * If the key is valid and the apiId is the defaultApiId, return a valid response
 *
 * Uses a known valid key to return a valid response
 *
 * Important: This behavior is for *this test mock only*. The actual verifyKey will *not* behave this way.
 */
vi.mock("@unkey/api", () => {
  return {
    Unkey: class {
      public get keys() {
        return {
          verify: ({
            key,
            apiId,
          }: {
            key: string;
            apiId?: string;
          }): VerifyResponse => {
            if (apiId !== DEFAULT_API_ID) {
              return {
                error: {
                  code: "INTERNAL_SERVER_ERROR",
                  docs: "",
                  message: `Internal server error ${key} ${apiId}`,
                  requestId: "123",
                },
              };
            }

            if (key === VALID_KEY && apiId === DEFAULT_API_ID) {
              return {
                error: undefined,
                result: {
                  valid: true,
                  code: "VALID",
                },
              };
            }

            if (key !== VALID_KEY && apiId === DEFAULT_API_ID) {
              return {
                error: undefined,
                result: {
                  valid: false,
                  code: "FORBIDDEN",
                },
              };
            }

            return {
              error: {
                code: "INTERNAL_SERVER_ERROR",
                docs: "",
                message: `Internal server error ${key} ${apiId}`,
                requestId: "123",
              },
            };
          },
        };
      }
    },
  };
});

describe("createApiKeyMiddleware", () => {
  it("should create middleware", async () => {
    const config: ApiKeyMiddlewareConfig = {};
    const middleware = createApiKeyMiddleware(config);
    expect(middleware).toBeDefined();
  });

  it("key is valid is a 200", async () => {
    const config: ApiKeyMiddlewareConfig = { apiId: "defaultApiId" };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { authorization: `Bearer ${VALID_KEY}` },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(200);
  });

  it("should 401 if key is invalid", async () => {
    const config: ApiKeyMiddlewareConfig = { apiId: "defaultApiId" };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { authorization: `Bearer ${INVALID_KEY}` },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(401);
  });

  it("should 401 if no headers", async () => {
    const config: ApiKeyMiddlewareConfig = { apiId: "defaultApiId" };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: {},
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(401);
  });

  it("should 401 if no authorization header", async () => {
    const config: ApiKeyMiddlewareConfig = { apiId: "defaultApiId" };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { agent: "test" },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(401);
  });

  it("should 500 if unsupported appId", async () => {
    const config: ApiKeyMiddlewareConfig = { apiId: "badid" };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { authorization: `Bearer ${VALID_KEY}` },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(500);
  });

  it("should be valid when using a custom key function", async () => {
    const config: ApiKeyMiddlewareConfig = {
      apiId: "defaultApiId",
      getKey: (req) => {
        return req.headers.get("x-api-key") ?? "";
      },
    };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { "x-api-key": VALID_KEY },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(200);
  });

  it("should be invalid when using a custom key function with invalid token", async () => {
    const config: ApiKeyMiddlewareConfig = {
      apiId: "defaultApiId",
      getKey: (req) => {
        return req.headers.get("x-api-key") ?? "";
      },
    };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { "x-api-key": INVALID_KEY },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.status).toBe(401);
  });

  it("should be a custom invalid response when using a custom key function with invalid token", async () => {
    const config: ApiKeyMiddlewareConfig = {
      apiId: "defaultApiId",
      getKey: (req) => {
        return req.headers.get("x-api-key") ?? "";
      },
      onInvalidKey: (_req, _result) => {
        return new MiddlewareResponse("Custom forbidden", { status: 403 });
      },
    };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { "x-api-key": INVALID_KEY },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.body).toBe("Custom forbidden");
    expect(result?.status).toBe(403);
  });

  it("should have a custom error response if unsupported appId", async () => {
    const config: ApiKeyMiddlewareConfig = {
      apiId: "badid",
      onError: (_req, _err) => new MiddlewareResponse("Custom unavailable", { status: 503 }),
    };
    const middleware = createApiKeyMiddleware(config);
    const request = new Request("http://localhost:8910/api/protected", {
      headers: { authorization: `Bearer ${INVALID_KEY}` },
    });
    const req = new MiddlewareRequest(request);
    const res = new MiddlewareResponse();
    const result = (await middleware(req, res)) as MiddlewareResponse;
    expect(result?.body).toBe("Custom unavailable");
    expect(result?.status).toBe(503);
  });
});
