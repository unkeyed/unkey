import type { InfiniteData } from "@tanstack/react-query";
import type { V2PortalListSessionsSession } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import {
  type SessionPage,
  removeSessionGroup,
  sessionGroupScopes,
  summarizeSessionGroup,
  toSessionPage,
} from "./sessions";

function session(overrides: Partial<V2PortalListSessionsSession>): V2PortalListSessionsSession {
  return {
    id: "ps_1",
    status: "active",
    createdAt: 1_000,
    expiresAt: 2_000,
    scopes: ["keys:read"],
    ...overrides,
  };
}

describe("sessionGroupScopes", () => {
  it("merges every session's scopes once, sorted", () => {
    const group = {
      externalId: "u1",
      sessions: [
        session({ id: "ps_1", scopes: ["keys:reroll", "keys:read"] }),
        session({ id: "ps_2", scopes: ["keys:read", "analytics:read"] }),
      ],
    };
    expect(sessionGroupScopes(group)).toEqual(["analytics:read", "keys:read", "keys:reroll"]);
  });
});

describe("summarizeSessionGroup", () => {
  it("uses the only session's times", () => {
    expect(summarizeSessionGroup({ externalId: "u1", sessions: [session({})] })).toEqual({
      count: 1,
      lastCreatedAt: 1_000,
      lastExpiresAt: 2_000,
    });
  });

  it("picks the latest creation and expiry independently", () => {
    const group = {
      externalId: "u1",
      sessions: [
        session({ id: "ps_new", status: "pending", createdAt: 5_000, expiresAt: 6_000 }),
        session({ id: "ps_old", status: "active", createdAt: 1_000, expiresAt: 90_000 }),
      ],
    };
    expect(summarizeSessionGroup(group)).toEqual({
      count: 2,
      lastCreatedAt: 5_000,
      lastExpiresAt: 90_000,
    });
  });
});

describe("toSessionPage", () => {
  const meta = { requestId: "req_1" };
  const groups = [{ externalId: "u1", sessions: [session({})] }];

  it("carries the cursor when more pages exist", () => {
    expect(
      toSessionPage({ meta, data: groups, pagination: { hasMore: true, cursor: "u2" } }),
    ).toEqual({
      groups,
      cursor: "u2",
    });
  });

  it("has no next cursor on the last page", () => {
    expect(
      toSessionPage({ meta, data: groups, pagination: { hasMore: false, cursor: "stale" } }),
    ).toEqual({ groups, cursor: undefined });
  });

  it("throws when more pages exist without a cursor", () => {
    expect(() => toSessionPage({ meta, data: groups, pagination: { hasMore: true } })).toThrow();
  });
});

describe("removeSessionGroup", () => {
  const u1 = { externalId: "u1", sessions: [session({ id: "ps_1" })] };
  const u2 = { externalId: "u2", sessions: [session({ id: "ps_2" })] };
  const u3 = { externalId: "u3", sessions: [session({ id: "ps_3" })] };

  it("drops only the revoked end user and keeps cursors", () => {
    const untouched: SessionPage = { groups: [u3], cursor: undefined };
    const data: InfiniteData<SessionPage> = {
      pages: [{ groups: [u1, u2], cursor: "u3" }, untouched],
      pageParams: [undefined, "u3"],
    };

    const result = removeSessionGroup(data, "u1");

    expect(result?.pages[0]).toEqual({ groups: [u2], cursor: "u3" });
    expect(result?.pages[1]).toBe(untouched);
    expect(result?.pageParams).toEqual([undefined, "u3"]);
  });

  it("leaves an empty cache alone", () => {
    expect(removeSessionGroup(undefined, "u1")).toBeUndefined();
  });
});
