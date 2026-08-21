import { beforeEach, describe, expect, it, vi } from "vitest";

// Chainable db mock (as in lib/stripe/linkDeploySubscription.test.ts):
// transaction(cb) runs cb with a tx whose update().set().where() resolves to
// the driver's `[{ affectedRows }]` result.
const h = vi.hoisted(() => {
  const where = vi.fn();
  const set = vi.fn().mockReturnValue({ where });
  const update = vi.fn().mockReturnValue({ set });
  const transaction = vi.fn(async (cb: (tx: unknown) => unknown) => cb({ update }));
  const insertAuditLogs = vi.fn();
  const getOrg = vi.fn();
  const logOperation = vi.fn();
  return { where, set, update, transaction, insertAuditLogs, getOrg, logOperation };
});

vi.mock("@/lib/db", () => ({
  db: { transaction: h.transaction },
  and: vi.fn(),
  eq: vi.fn(),
  sql: vi.fn(),
  schema: { workspaces: { id: {}, name: {} } },
}));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: h.insertAuditLogs }));
vi.mock("@/lib/auth/server", () => ({ auth: { getOrg: h.getOrg } }));
vi.mock("@/lib/logging/structured-logger", () => ({ logOperation: h.logOperation }));

import { mysqlServerError } from "../utils/test-helpers";
import { reconcileFailedOrgRename } from "./reconcile-org-rename";

const BASE = {
  orgId: "org_1",
  workspaceId: "ws_1",
  requestedName: "NewName",
  previousName: "OldName",
  actorId: "user_1",
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function run(overrides: Partial<typeof BASE> = {}) {
  return reconcileFailedOrgRename({ ...BASE, ...overrides });
}

beforeEach(() => {
  vi.clearAllMocks();
  // Default: the revert affects the single row it targeted.
  h.where.mockResolvedValue([{ affectedRows: 1 }]);
});

describe("reconcileFailedOrgRename", () => {
  it("leaves the DB at the requested name and does not touch it when the org is unreadable, because reverting could create drift if the rename had in fact applied", async () => {
    // Auth clients reject with a plain {success, code, message} object whose
    // message may quote the org name.
    h.getOrg.mockRejectedValue({
      success: false,
      code: "ORG_READ_FAILED",
      message: "Could not read organization 'Acme Corp'",
    });

    const outcome = await run();

    expect(outcome).toBe("left-at-requested-name");
    expect(h.transaction).not.toHaveBeenCalled();
    expect(h.insertAuditLogs).not.toHaveBeenCalled();
  });

  it("logs only the auth error's symbolic code when the org read fails, keeping the quoted org name out of the logs", async () => {
    h.getOrg.mockRejectedValue({
      success: false,
      code: "ORG_READ_FAILED",
      message: "Could not read organization 'Acme Corp'",
    });

    await run();

    expect(h.logOperation).toHaveBeenCalledTimes(1);
    const [level, , attributes] = h.logOperation.mock.calls[0];
    expect(level).toBe("error");
    expect(attributes).toMatchObject({
      workspace_id: "ws_1",
      error_detail: "error code: ORG_READ_FAILED",
    });
    expect(JSON.stringify(attributes)).not.toContain("Acme Corp");
  });

  it("reports rename-in-effect when the org already holds the requested name, so the caller reports success for a rename the provider applied but whose response was lost", async () => {
    h.getOrg.mockResolvedValue({ id: "org_1", name: "NewName" });

    const outcome = await run();

    expect(outcome).toBe("rename-in-effect");
    expect(h.transaction).not.toHaveBeenCalled();
    expect(h.insertAuditLogs).not.toHaveBeenCalled();
  });

  it("treats a provider-normalized variant of the requested name as rename-in-effect instead of falsely reverting a rename the provider applied", async () => {
    // NFD (e + combining acute) plus a trailing space vs the NFC form we
    // sent; escapes keep the two genuinely different in source.
    h.getOrg.mockResolvedValue({ id: "org_1", name: "Cafe\u0301 " });

    const outcome = await run({ requestedName: "Caf\u00e9" });

    expect(outcome).toBe("rename-in-effect");
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("reports left-at-requested-name for a same-name repair without writing a nonsensical 'reverted X to X' audit entry", async () => {
    // Org holds some other name, so the rename-in-effect check does not fire;
    // the same-name guard is what must return here.
    h.getOrg.mockResolvedValue({ id: "org_1", name: "OldName" });

    const outcome = await run({ requestedName: "SameName", previousName: "SameName" });

    expect(outcome).toBe("left-at-requested-name");
    expect(h.transaction).not.toHaveBeenCalled();
    expect(h.insertAuditLogs).not.toHaveBeenCalled();
  });

  it("checks the org's current name before the same-name guard, so a same-name repair whose org already holds the name still reports success", async () => {
    // Ordering is load-bearing: a refactor that ran the same-name guard first
    // would return left-at-requested-name here and mis-report a success.
    h.getOrg.mockResolvedValue({ id: "org_1", name: "SameName" });

    const outcome = await run({ requestedName: "SameName", previousName: "SameName" });

    expect(outcome).toBe("rename-in-effect");
  });

  it("reverts the DB to the previous name and audits it when the org never took the rename", async () => {
    h.getOrg.mockResolvedValue({ id: "org_1", name: "OldName" });

    const outcome = await run();

    expect(outcome).toBe("reverted");
    expect(h.insertAuditLogs).toHaveBeenCalledTimes(1);
    const [, auditEntry] = h.insertAuditLogs.mock.calls[0];
    expect(auditEntry.description).toBe(
      "Reverted name from NewName back to OldName after the organization rename failed",
    );
    expect(auditEntry.resources[0].name).toBe("OldName");
    expect(h.logOperation).not.toHaveBeenCalled();
  });

  it("reports superseded when the revert affects no rows, so a concurrent rename that moved the row past the requested name is neither clobbered nor misreported as a completed revert", async () => {
    h.getOrg.mockResolvedValue({ id: "org_1", name: "OldName" });
    h.where.mockResolvedValue([{ affectedRows: 0 }]);

    const outcome = await run();

    expect(outcome).toBe("superseded");
    expect(h.transaction).toHaveBeenCalledTimes(1);
    expect(h.insertAuditLogs).not.toHaveBeenCalled();
  });

  it("reports left-at-requested-name and logs a symbolic code when the revert transaction fails, keeping the DB and org knowingly divergent rather than claiming nothing changed", async () => {
    h.getOrg.mockResolvedValue({ id: "org_1", name: "OldName" });
    // Once-only: vi.clearAllMocks() clears calls, not implementations, so a
    // plain mockRejectedValue would leak into later tests.
    h.transaction.mockRejectedValueOnce(
      mysqlServerError(
        "Deadlock found when trying to get lock; 'OldName'",
        "ER_LOCK_DEADLOCK",
        1213,
        "40001",
      ),
    );

    const outcome = await run();

    expect(outcome).toBe("left-at-requested-name");
    expect(h.logOperation).toHaveBeenCalledTimes(1);
    const [level, , attributes] = h.logOperation.mock.calls[0];
    expect(level).toBe("error");
    expect(attributes).toMatchObject({
      workspace_id: "ws_1",
      error_detail: "database error: ER_LOCK_DEADLOCK",
    });
    expect(JSON.stringify(attributes)).not.toContain("OldName");
  });
});
