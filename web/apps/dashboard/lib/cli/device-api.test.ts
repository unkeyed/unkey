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
  parseDecision,
  parseDeviceLogin,
  revokeCliSession,
} from "./device-api";

const userCode = "BCDF-GHJK";
const handle = "BCDFGHJKLMNPQRSTVWXZ234567";
const redirectUri = "https://gateway.example.com:8443/oauth/callback";
const redirectTo = `${redirectUri}?code=${newId("test")}&state=${newId("test")}`;
const sessionId = newId("test");
const permissions = [`unkey:v1:${newId("workspace")}:**#read`];
const meta = { requestId: newId("request") };
const loginResponse: V2CliGetDeviceLoginResponseBody = {
  meta,
  data: {
    userCode,
    delivery: "device",
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
const redirectResponse: V2CliDecideDeviceLoginResponseBody = { meta, data: { redirectTo } };
const deviceLogin = parseDeviceLogin(loginResponse.data);
const clientId = newId("test");
const appResponse: V2CliGetDeviceLoginResponseBody["data"] = {
  ...loginResponse.data,
  userCode: handle,
  delivery: "redirect",
  deviceName: "Unkey MCP",
  client: { id: clientId, name: "Unkey MCP", redirectUri },
};
const appLogin = parseDeviceLogin(appResponse);

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
      call: () => approveDeviceLogin(deviceLogin, permissions),
      body: { userCode, permissions, decision: "approve" },
      response: emptyResponse,
      expected: { delivery: "device" },
    },
    {
      route: "decideDeviceLogin",
      call: () => denyDeviceLogin(deviceLogin),
      body: { userCode, decision: "deny" },
      response: emptyResponse,
      expected: { delivery: "device" },
    },
    {
      route: "decideDeviceLogin",
      call: () => approveDeviceLogin(appLogin, permissions),
      body: { userCode: handle, permissions, decision: "approve" },
      response: redirectResponse,
      expected: { delivery: "redirect", redirectTo },
    },
    {
      route: "decideDeviceLogin",
      call: () => denyDeviceLogin(appLogin),
      body: { userCode: handle, decision: "deny" },
      response: redirectResponse,
      expected: { delivery: "redirect", redirectTo },
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
    { operation: "approve", call: () => approveDeviceLogin(deviceLogin, permissions) },
    { operation: "deny", call: () => denyDeviceLogin(deviceLogin) },
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

describe("parseDeviceLogin", () => {
  it("keeps a CLI login as device delivery without an app", () => {
    expect(parseDeviceLogin(loginResponse.data)).toEqual({
      ...loginResponse.data,
      delivery: "device",
    });
  });

  it("adds the redirect host to an OAuth app login", () => {
    expect(appLogin).toEqual({
      ...appResponse,
      delivery: "redirect",
      client: {
        id: clientId,
        name: "Unkey MCP",
        redirectUri,
        redirectHost: "gateway.example.com:8443",
      },
    });
  });

  it("rejects a redirect login without its app", () => {
    expect(() =>
      parseDeviceLogin({ ...loginResponse.data, userCode: handle, delivery: "redirect" }),
    ).toThrow();
  });

  it.each(["javascript:alert(1)", "not a url", "unkey://callback"])(
    "rejects the redirect URI %s",
    (uri) => {
      expect(() =>
        parseDeviceLogin({
          ...loginResponse.data,
          userCode: handle,
          delivery: "redirect",
          client: { id: newId("test"), name: "Unkey MCP", redirectUri: uri },
        }),
      ).toThrow();
    },
  );
});

describe("parseDecision", () => {
  it("ignores redirectTo for a CLI login", () => {
    expect(parseDecision(deviceLogin, { redirectTo })).toEqual({ delivery: "device" });
  });

  it("follows redirectTo on the app's redirect origin", () => {
    expect(parseDecision(appLogin, { redirectTo })).toEqual({ delivery: "redirect", redirectTo });
  });

  it.each([
    { name: "missing", response: {} },
    { name: "another host", response: { redirectTo: "https://evil.example.com/oauth/callback" } },
    {
      name: "another port",
      response: { redirectTo: "https://gateway.example.com/oauth/callback" },
    },
    { name: "a downgraded scheme", response: { redirectTo: "http://gateway.example.com:8443/cb" } },
    { name: "a script", response: { redirectTo: "javascript:alert(1)" } },
  ])("rejects a redirect to $name", ({ response }) => {
    expect(() => parseDecision(appLogin, response)).toThrow();
  });
});
