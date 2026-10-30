import type { User } from "@/lib/auth/types";
import type { LatestAuditLogByTargetRow } from "@unkey/clickhouse/src/audit-logs";
import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import { toRollbackEvent } from "./rollback-event";

const appId = newId("app");
const toId = newId("test");
const fromId = newId("test");
const userId = newId("test");

function rowOf(overrides: Partial<LatestAuditLogByTargetRow> = {}): LatestAuditLogByTargetRow {
  return {
    time: 1_700_000_000_000,
    actorType: "user",
    actorId: userId,
    actorName: "",
    targets: [
      ["deployment", toId],
      ["app", appId],
      ["deployment", fromId],
    ],
    ...overrides,
  };
}

const user: User = {
  id: userId,
  email: "ada@example.com",
  firstName: "Ada",
  lastName: "Lovelace",
  avatarUrl: "https://example.com/ada.png",
  fullName: "Ada Lovelace",
};

describe("toRollbackEvent", () => {
  it("names the user, their avatar and the replaced deployment", () => {
    expect(toRollbackEvent(rowOf(), user, toId)).toEqual({
      at: 1_700_000_000_000,
      actor: { name: "Ada Lovelace", avatarUrl: "https://example.com/ada.png" },
      fromDeploymentId: fromId,
    });
  });

  it("falls back to the email when the user has no name", () => {
    const nameless = { ...user, firstName: null, lastName: null };
    expect(toRollbackEvent(rowOf(), nameless, toId)?.actor).toEqual({
      name: "ada@example.com",
      avatarUrl: "https://example.com/ada.png",
    });
  });

  it("uses the audit actor name without an avatar for a root key", () => {
    const row = rowOf({ actorType: "key", actorName: "ci-key" });
    expect(toRollbackEvent(row, null, toId)?.actor).toEqual({
      name: "ci-key",
      avatarUrl: null,
    });
  });

  it("uses the audit actor name when the user lookup failed", () => {
    const row = rowOf({ actorName: "ada" });
    expect(toRollbackEvent(row, null, toId)?.actor).toEqual({ name: "ada", avatarUrl: null });
  });

  it("has no actor when nothing names it", () => {
    expect(toRollbackEvent(rowOf(), null, toId)?.actor).toBeNull();
  });

  it("has no replaced deployment when the event lists only one", () => {
    const row = rowOf({
      targets: [
        ["deployment", toId],
        ["app", appId],
      ],
    });
    expect(toRollbackEvent(row, user, toId)?.fromDeploymentId).toBeNull();
  });

  it("ignores an event for a deployment that is no longer live", () => {
    expect(toRollbackEvent(rowOf(), user, newId("test"))).toBeNull();
  });
});
