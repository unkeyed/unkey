import { getErrorMessage } from "@/lib/unkey-client";
import type {
  V2CliDecideDeviceLoginResponseBody,
  V2CliGetDeviceLoginResponseBody,
  V2CliListSessionsResponseBody,
} from "@unkey/api/models/components";
import { ResponseValidationError } from "@unkey/api/models/errors";
import { newId } from "@unkey/id";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  approveDeviceLogin,
  denyDeviceLogin,
  getDeviceLogin,
  listCliSessions,
  revokeCliSession,
} from "./device-api";

const userCode = "BCDF-GHJK";
const sessionId = newId("test");
const permissions = [`unkey:v1:${newId("workspace")}:**#read`];
const meta = { requestId: newId("request") };
const loginResponse: V2CliGetDeviceLoginResponseBody = {
  meta,
  data: {
    userCode,
    deviceName: "Test laptop",
    requesterIp: "192.0.2.1",
    status: "pending",
    grantablePermissions: permissions,
    createdAt: 1_700_000_000_000,
    expiresAt: 1_700_000_600_000,
  },
};
const sessionsResponse: V2CliListSessionsResponseBody = {
  meta,
  data: {
    sessions: [
      {
        id: sessionId,
        userId: newId("test"),
        permissions,
        createdAt: 1_700_000_000_000,
        lastUsedAt: 1_700_000_100_000,
        expiresAt: 1_700_604_800_000,
      },
    ],
  },
};
const emptyResponse: V2CliDecideDeviceLoginResponseBody = { meta, data: {} };

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("dashboard CLI API", () => {
  it.each([
    {
      route: "getDeviceLogin",
      call: () => getDeviceLogin(userCode),
      body: { userCode },
      response: loginResponse,
      expected: loginResponse.data,
    },
    {
      route: "decideDeviceLogin",
      call: () => approveDeviceLogin({ userCode, permissions }),
      body: { userCode, permissions, decision: "approve" },
      response: emptyResponse,
      expected: undefined,
    },
    {
      route: "decideDeviceLogin",
      call: () => denyDeviceLogin(userCode),
      body: { userCode, decision: "deny" },
      response: emptyResponse,
      expected: undefined,
    },
    {
      route: "listSessions",
      call: listCliSessions,
      body: undefined,
      response: sessionsResponse,
      expected: sessionsResponse.data.sessions,
    },
    {
      route: "revokeSession",
      call: () => revokeCliSession(sessionId),
      body: { sessionId },
      response: emptyResponse,
      expected: undefined,
    },
  ])("sends $route through the proxy without credentials", async (test) => {
    const requests: Request[] = [];
    vi.stubGlobal("fetch", async (input: RequestInfo | URL, init?: RequestInit) => {
      requests.push(input instanceof Request ? input : new Request(input, init));
      return Response.json(test.response);
    });

    await expect(test.call()).resolves.toEqual(test.expected);

    expect(requests).toHaveLength(1);
    const request = requests[0];
    if (!request) {
      throw new Error("Expected a CLI API request");
    }
    expect(request.url).toBe(`http://localhost:3000/proxy/v2/cli.${test.route}`);
    expect(request.method).toBe("POST");
    expect(request.headers.has("Authorization")).toBe(false);
    if (test.body === undefined) {
      expect(request.body).toBeNull();
      expect(request.headers.has("Content-Type")).toBe(false);
      return;
    }
    await expect(request.json()).resolves.toEqual(test.body);
  });

  it.each([
    { operation: "approve", call: () => approveDeviceLogin({ userCode, permissions }) },
    { operation: "deny", call: () => denyDeviceLogin(userCode) },
    { operation: "revoke", call: () => revokeCliSession(sessionId) },
  ])(
    "does not retry $operation after a server error and keeps the API detail",
    async ({ call }) => {
      const errorBody = {
        meta,
        error: {
          detail: "CLI session storage is unavailable.",
          status: 500,
          title: "Internal Server Error",
          type: "https://unkey.com/docs/errors/unkey/application/internal_server_error",
        },
      };
      const fetch = vi.fn(async () => Response.json(errorBody, { status: 500 }));
      vi.stubGlobal("fetch", fetch);

      const error = await call().catch((error: unknown) => error);

      expect(getErrorMessage(error, "Operation failed.")).toBe(errorBody.error.detail);
      expect(fetch).toHaveBeenCalledTimes(1);
    },
  );

  it("rejects malformed success data instead of treating a session as authorized", async () => {
    vi.stubGlobal("fetch", async () =>
      Response.json({
        ...loginResponse,
        data: { ...loginResponse.data, expiresAt: "not a timestamp" },
      }),
    );

    await expect(getDeviceLogin(userCode)).rejects.toBeInstanceOf(ResponseValidationError);
  });

  it("uses the operation fallback for proxy-local errors", async () => {
    vi.stubGlobal("fetch", async () =>
      Response.json({ error: "Authentication required." }, { status: 401 }),
    );

    const error = await listCliSessions().catch((error: unknown) => error);

    expect(error).toBeInstanceOf(Error);
    expect(getErrorMessage(error, "Could not load CLI sessions.")).toBe(
      "Could not load CLI sessions.",
    );
  });
});
