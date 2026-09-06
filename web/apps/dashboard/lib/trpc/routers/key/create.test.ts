import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { createKey } from "./create";
import { createRootKey } from "./createRootKey";

const state = vi.hoisted(() => ({
  inTransaction: false,
  transactions: 0,
  encrypt: vi.fn(),
  insert: vi.fn(),
  audit: vi.fn(),
  identity: vi.fn(),
}));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => ({ encrypt: state.encrypt }) }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: state.audit }));
vi.mock("@/lib/env", () => ({ env: () => ({ UNKEY_WORKSPACE_ID: "unkey", UNKEY_API_ID: "api" }) }));
vi.mock("@/lib/projects/ensure-default-project-id", () => ({
  ensureDefaultProjectId: async () => "project",
}));
vi.mock("@unkey/keys", () => ({
  newKey: async () => ({ key: "secret", hash: "hash", prefix: "test", start: "start", end: "end" }),
}));
vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.create();
  return {
    workspaceProcedure: t.procedure,
    requireWorkspaceAdmin: t.middleware(({ next }) => next()),
    ratelimit: { create: {} },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});
vi.mock(
  "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema",
  async () => {
    const { z } = await import("zod");
    return {
      createKeyInputSchema: z.object({ keyAuthId: z.string(), identityId: z.string().optional() }),
    };
  },
);
vi.mock("@/lib/db", () => {
  const tx = {
    insert: (table: string) => ({ values: async (values: unknown) => state.insert(table, values) }),
    query: { identities: { findFirst: state.identity }, permissions: { findMany: async () => [] } },
  };
  return {
    schema: {
      keys: "keys",
      encryptedKeys: "encryptedKeys",
      permissions: "permissions",
      keysPermissions: "keysPermissions",
    },
    db: {
      query: {
        keyAuth: { findFirst: async () => ({ id: "ka", storeEncryptedKeys: true }) },
        apis: { findFirst: async () => ({ keyAuthId: "ka" }) },
      },
      transaction: async (fn: (connection: typeof tx) => Promise<unknown>) => {
        expect(state.inTransaction).toBe(false);
        state.transactions++;
        state.inTransaction = true;
        try {
          return await fn(tx);
        } finally {
          state.inTransaction = false;
        }
      },
    },
  };
});
const caller = initTRPC
  .context<{ workspace: { id: string }; user: { id: string }; audit: { location: string } }>()
  .create()
  .router({ create: createKey, createRoot: createRootKey })
  .createCaller({ workspace: { id: "ws" }, user: { id: "user" }, audit: { location: "test" } });

beforeEach(() => {
  state.transactions = 0;
  state.insert.mockReset();
  state.audit.mockReset();
  state.identity.mockReset().mockResolvedValue({ id: "identity" });
  state.encrypt.mockReset().mockImplementation(async () => {
    expect(state.inTransaction).toBe(false);
    return { encrypted: "ciphertext", keyId: "encryption-key" };
  });
});

it("encrypts before opening the transaction and persists the prepared ciphertext", async () => {
  expect(await caller.create({ keyAuthId: "ka" })).toMatchObject({ key: "secret" });
  expect(state.transactions).toBe(1);
  expect(state.insert).toHaveBeenCalledWith(
    "encryptedKeys",
    expect.objectContaining({ encrypted: "ciphertext", encryptionKeyId: "encryption-key" }),
  );
  expect(state.audit).toHaveBeenCalledOnce();
});

it("does not open a transaction when Vault fails", async () => {
  state.encrypt.mockRejectedValue(new Error("vault unavailable"));
  await expect(caller.create({ keyAuthId: "ka" })).rejects.toThrow();
  expect(state.transactions).toBe(0);
  expect(state.insert).not.toHaveBeenCalled();
});

it("still rejects an identity outside the workspace before writing", async () => {
  state.identity.mockResolvedValue(undefined);
  await expect(caller.create({ keyAuthId: "ka", identityId: "foreign" })).rejects.toThrow();
  expect(state.insert).not.toHaveBeenCalled();
});

it("creates root-key permissions in the caller's transaction instead of opening a second one", async () => {
  expect(await caller.createRoot({ permissions: ["api.*.create_api"] })).toMatchObject({
    key: "secret",
  });
  expect(state.transactions).toBe(1);
  expect(state.insert).toHaveBeenCalledWith("permissions", expect.any(Array));
  expect(state.insert).toHaveBeenCalledWith("keysPermissions", expect.any(Array));
  expect(state.audit).toHaveBeenCalledOnce();
});
