import { Unkey } from "@unkey/api";
import type {
  NotificationPagination,
  V2NotificationsListDestinationsResponseBody,
  V2NotificationsListMembersResponseBody,
  V2NotificationsListRulesResponseBody,
} from "@unkey/api/models/components";
import { UnkeyError } from "@unkey/api/models/errors";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  creationRejected,
  listNotificationDestinations,
  listNotificationMembers,
  listNotificationRules,
} from "./client";

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => new Unkey({ serverURL: "http://dashboard.test/proxy/" }),
}));

function response(
  body:
    | V2NotificationsListRulesResponseBody
    | V2NotificationsListDestinationsResponseBody
    | V2NotificationsListMembersResponseBody,
) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

const cases = [
  {
    endpoint: "listRules",
    list: listNotificationRules,
    page: (pagination: NotificationPagination) =>
      response({
        meta: { requestId: crypto.randomUUID() },
        data: [
          {
            id: crypto.randomUUID(),
            name: "Failures",
            notificationType: "deployment.failed",
            enabled: true,
            targets: [{ type: "audience", audience: "workspace_admins" }],
          },
        ],
        pagination,
      }),
  },
  {
    endpoint: "listDestinations",
    list: listNotificationDestinations,
    page: (pagination: NotificationPagination) =>
      response({
        meta: { requestId: crypto.randomUUID() },
        data: [
          {
            id: crypto.randomUUID(),
            name: "Webhook",
            channel: "webhook",
            urlHint: "https://example.com",
            headerNames: ["X-Token"],
            enabled: true,
            createdAt: 1,
          },
        ],
        pagination,
      }),
  },
  {
    endpoint: "listMembers",
    list: listNotificationMembers,
    page: (pagination: NotificationPagination) =>
      response({
        meta: { requestId: crypto.randomUUID() },
        data: [{ userId: crypto.randomUUID(), email: "admin@example.com", admin: true }],
        pagination,
      }),
  },
];

beforeEach(() => vi.stubEnv("UNKEY_ROOT_KEY", undefined));
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe.each(cases)("notifications.$endpoint", ({ endpoint, list, page }) => {
  it("reads every cursor page through the proxy without a root key", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(page({ hasMore: true, nextCursor: "next" }))
      .mockResolvedValueOnce(page({ hasMore: false }));
    vi.stubGlobal("fetch", fetch);
    expect(await list()).toHaveLength(2);
    const bodies = await Promise.all(
      fetch.mock.calls.map(async ([request]) => {
        if (!(request instanceof Request)) {
          throw new Error("Expected SDK Request");
        }
        expect(request.url).toBe(`http://dashboard.test/proxy/v2/notifications.${endpoint}`);
        expect(request.headers.has("authorization")).toBe(false);
        return request.json();
      }),
    );
    expect(bodies).toEqual([{ limit: 100 }, { limit: 100, cursor: "next" }]);
  });

  it("rejects incomplete pagination instead of silently truncating recipients or rules", async () => {
    const fetch = vi.fn<typeof globalThis.fetch>().mockResolvedValueOnce(page({ hasMore: true }));
    vi.stubGlobal("fetch", fetch);
    await expect(list()).rejects.toThrow("without a cursor");
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("stops repeated cursors instead of looping", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockImplementation(async () => page({ hasMore: true, nextCursor: "same" }));
    vi.stubGlobal("fetch", fetch);
    await expect(list()).rejects.toThrow("repeated cursor");
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});

describe("ambiguous creation outcomes", () => {
  it.each([408, 409, 500, 503])("keeps the original request after HTTP %i", (status) => {
    const error = new UnkeyError("Failed", {
      request: new Request("https://example.com"),
      response: new Response(null, { status }),
      body: "",
    });
    expect(creationRejected(error)).toBe(false);
  });
  it("keeps the original request after a network failure", () => {
    expect(creationRejected(new Error("Connection lost"))).toBe(false);
  });
  it.each([400, 401, 403, 404, 422, 429])(
    "allows correction of a rejected HTTP %i request",
    (status) => {
      const error = new UnkeyError("Rejected", {
        request: new Request("https://example.com"),
        response: new Response(null, { status }),
        body: "",
      });
      expect(creationRejected(error)).toBe(true);
    },
  );
});
