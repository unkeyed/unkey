import { initTRPC } from "@trpc/server";
import { expect, it, vi } from "vitest";
import { rerollRootKey } from "./index";

const state = vi.hoisted(() => ({ inTransaction: false, encrypt: vi.fn(), insert: vi.fn() }));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => ({ encrypt: state.encrypt }) }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/env", () => ({ env: () => ({ UNKEY_WORKSPACE_ID: "unkey" }) }));
vi.mock("@unkey/keys", () => ({
  newKey: async () => ({ key: "secret", hash: "hash", start: "start", end: "end" }),
}));
vi.mock("../../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.create();
  const pass = t.middleware(({ next }) => next());
  return {
    workspaceProcedure: t.procedure,
    requireWorkspaceAdmin: pass,
    ratelimit: { create: {} },
    withRatelimit: () => pass,
  };
});
vi.mock("@/lib/db", () => {
  const tx = {
    select: () => ({ from: () => ({ where: () => ({ for: async () => [{ id: "key" }] }) }) }),
    query: {
      keys: {
        findFirst: async () => ({
          workspaceId: "unkey",
          start: "unkey_",
          prefix: "unkey",
          expires: null,
          encrypted: { keyId: "key" },
          keyAuth: { storeEncryptedKeys: true, defaultBytes: 16 },
        }),
      },
    },
    insert: state.insert,
  };
  return {
    and: vi.fn(),
    eq: vi.fn(),
    isNull: vi.fn(),
    schema: { keys: {} },
    db: {
      transaction: async (fn: (connection: typeof tx) => Promise<unknown>) => {
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

it("bounds encryption of the locked source and rolls back on a Vault failure", async () => {
  state.encrypt.mockImplementation(async (_request: unknown, options: { timeoutMs: number }) => {
    expect(state.inTransaction).toBe(true);
    expect(options.timeoutMs).toBe(2_000);
    throw new Error("deadline exceeded");
  });
  const caller = initTRPC
    .context<{ workspace: { id: string }; user: { id: string }; audit: { location: string } }>()
    .create()
    .router({ reroll: rerollRootKey })
    .createCaller({ workspace: { id: "ws" }, user: { id: "user" }, audit: { location: "test" } });
  await expect(caller.reroll({ keyId: "key", expiration: 0 })).rejects.toThrow();
  expect(state.encrypt).toHaveBeenCalledOnce();
  expect(state.insert).not.toHaveBeenCalled();
  expect(state.inTransaction).toBe(false);
});
