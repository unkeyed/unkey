import { createLogger } from "@redwoodjs/api/logger";
import { MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import { assert, describe, expect, it, vi } from "vitest";
import createApiKeyMiddleware from "../createApiKeyMiddleware";
import type { ApiKeyMiddlewareConfig } from "../types";

describe("createApiKeyMiddleware", () => {
  it("should create middleware", async () => {
    const config: ApiKeyMiddlewareConfig = {};
    const middleware = createApiKeyMiddleware(config);
    expect(middleware).toBeDefined();
  });
});
