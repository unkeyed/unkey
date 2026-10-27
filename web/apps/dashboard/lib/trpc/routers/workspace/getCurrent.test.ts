import { loadWorkspace } from "@/lib/db/load-workspace";
import { afterAll, beforeEach, expect, it, vi } from "vitest";
import { t } from "../../trpc";
import { getCurrentWorkspace } from "./getCurrent";

vi.hoisted(() => {
  vi.stubEnv("UNKEY_WORKSPACE_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_API_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_ROOT_KEY", undefined);
  vi.stubEnv("VAULT_URL", "http://localhost");
  vi.stubEnv("VAULT_TOKEN", "test");
  vi.stubEnv("AUTH_PROVIDER", "local");
});

const database = vi.hoisted(() => ({ findFirst: vi.fn() }));
vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: database.findFirst } } },
}));

const orgId = crypto.randomUUID();
const userId = crypto.randomUUID();
const workspaceRecord = {
  id: crypto.randomUUID(),
  orgId,
  k8sNamespace: "private-namespace",
  billing: { stripeCustomerId: "private-customer", planOverride: "Enterprise" },
  billingSubscriptions: [],
};

beforeEach(() => {
  database.findFirst.mockReset().mockResolvedValue(workspaceRecord);
});
afterAll(() => vi.unstubAllEnvs());

function caller(
  role: string | null,
  workspace: Awaited<ReturnType<typeof loadWorkspace>> = undefined,
  tenantId: string | null = orgId,
) {
  return t.router({ getCurrent: getCurrentWorkspace }).createCaller({
    req: new Request("http://localhost/api/trpc"),
    audit: { location: "unknown", userAgent: undefined },
    user: { id: userId, profile: null },
    tenant: tenantId ? { id: tenantId, role } : null,
    workspace,
  });
}

it.each(["viewer", "custom-role", null])(
  "denies workspace metadata to role %j with or without cached context",
  async (role) => {
    const workspace = await loadWorkspace(orgId);
    await expect(caller(role, workspace).getCurrent()).rejects.toMatchObject({ code: "FORBIDDEN" });
    await expect(caller(role).getCurrent()).rejects.toMatchObject({ code: "FORBIDDEN" });
  },
);

it.each(["admin", "developer", "basic_member"])(
  "retains workspace access and fallback loading for role %j",
  async (role) => {
    const workspace = await loadWorkspace(orgId);
    await expect(caller(role, workspace).getCurrent()).resolves.toMatchObject({
      id: workspaceRecord.id,
      stripeCustomerId: "private-customer",
    });
    await expect(caller(role).getCurrent()).resolves.toMatchObject({
      id: workspaceRecord.id,
      stripeCustomerId: "private-customer",
    });
  },
);

it("retains onboarding for sessions without an organization or workspace", async () => {
  database.findFirst.mockResolvedValue(undefined);
  await expect(caller(null, undefined, null).getCurrent()).rejects.toMatchObject({
    code: "NOT_FOUND",
  });
  await expect(caller("admin").getCurrent()).rejects.toMatchObject({ code: "NOT_FOUND" });
});

it("retains database errors instead of reporting a missing workspace", async () => {
  const error = new Error("database unavailable");
  database.findFirst.mockRejectedValue(error);
  await expect(caller("developer").getCurrent()).rejects.toMatchObject({
    code: "INTERNAL_SERVER_ERROR",
    cause: error,
  });
});
