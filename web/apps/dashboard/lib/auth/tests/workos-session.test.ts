import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  requestHeaders: new Headers(),
}));

vi.mock("next/headers", () => ({
  headers: async () => mocks.requestHeaders,
}));

vi.mock("@/lib/env", () => ({
  workosAuthEnv: vi.fn(),
}));

import { hasAuthkitMiddleware } from "../workos-session";

describe("hasAuthkitMiddleware", () => {
  beforeEach(() => {
    mocks.requestHeaders = new Headers();
  });

  it("is true when the request passed through the AuthKit middleware", async () => {
    mocks.requestHeaders = new Headers({ "x-workos-middleware": "1" });

    await expect(hasAuthkitMiddleware()).resolves.toBe(true);
  });

  it("is false for a dotted path the middleware matcher excludes", async () => {
    mocks.requestHeaders = new Headers({
      "x-url": "https://app.unkey.com/favicon.ico",
    });

    await expect(hasAuthkitMiddleware()).resolves.toBe(false);
  });
});
