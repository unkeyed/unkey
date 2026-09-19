import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  logOperation: vi.fn(),
}));

vi.mock("@/lib/logging", () => ({
  logOperation: mocks.logOperation,
}));

import { logAuthkitMiddlewareBypass } from "./telemetry";

describe("logAuthkitMiddlewareBypass", () => {
  it("records the bypassed request with its host, matched route, and client", () => {
    mocks.logOperation.mockClear();

    logAuthkitMiddlewareBypass(
      new Headers({
        host: "app.unkey.com",
        accept: "*/*",
        "user-agent": "NetworkingExtension/8625.1.29.10.29",
        "x-matched-path": "/[workspaceSlug]",
      }),
    );

    expect(mocks.logOperation).toHaveBeenCalledWith(
      "warn",
      "Request bypassed the AuthKit middleware",
      {
        auth_event: "middleware_bypass",
        next_route: "/[workspaceSlug]",
        request_host: "app.unkey.com",
        request_user_agent: "NetworkingExtension/8625.1.29.10.29",
        request_accept: "*/*",
      },
    );
  });

  it("leaves attributes the request does not carry unset", () => {
    mocks.logOperation.mockClear();

    logAuthkitMiddlewareBypass(new Headers());

    expect(mocks.logOperation).toHaveBeenCalledWith(
      "warn",
      "Request bypassed the AuthKit middleware",
      {
        auth_event: "middleware_bypass",
        next_route: undefined,
        request_host: undefined,
        request_user_agent: undefined,
        request_accept: undefined,
      },
    );
  });
});
