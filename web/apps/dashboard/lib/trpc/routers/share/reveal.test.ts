import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { revealSharedSecret } from "./reveal";

type Row = { workspaceId: string; encrypted: string; expiresAt: number };
const state = vi.hoisted(() => ({
  row: undefined as Row | undefined,
  inTransaction: false,
  decrypt: vi.fn(),
  audit: vi.fn(),
}));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => ({ decrypt: state.decrypt }) }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: state.audit }));
vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return {
    publicProcedure: initTRPC.context<{ audit: { location: string } }>().create().procedure,
  };
});
vi.mock("@/lib/db", () => {
  const select = () => ({
    from: () => ({
      where: () =>
        Object.assign(Promise.resolve(state.row ? [{ ...state.row }] : []), {
          for: async () => (state.row ? [{ ...state.row }] : []),
        }),
    }),
  });
  const tx = {
    select,
    delete: () => ({
      where: async () => {
        state.row = undefined;
      },
    }),
  };
  let previous: Promise<unknown> = Promise.resolve();
  return {
    eq: vi.fn(),
    schema: { sharedSecrets: {} },
    db: {
      select,
      transaction: (fn: (connection: typeof tx) => Promise<unknown>) => {
        const result = previous
          .catch(() => {})
          .then(async () => {
            const before = state.row;
            state.inTransaction = true;
            try {
              return await fn(tx);
            } catch (err) {
              state.row = before;
              throw err;
            } finally {
              state.inTransaction = false;
            }
          });
        previous = result;
        return result;
      },
    },
  };
});
const caller = initTRPC
  .context<{ audit: { location: string } }>()
  .create()
  .router({ reveal: revealSharedSecret })
  .createCaller({ audit: { location: "test" } });

beforeEach(() => {
  state.row = { workspaceId: "ws", encrypted: "ciphertext", expiresAt: Date.now() + 60_000 };
  state.audit.mockReset();
  state.decrypt.mockReset().mockImplementation(async () => {
    expect(state.inTransaction).toBe(false);
    return { plaintext: "secret" };
  });
});

it("decrypts without a transaction and returns the secret only once", async () => {
  const results = await Promise.all([
    caller.reveal({ id: "share" }),
    caller.reveal({ id: "share" }),
  ]);
  expect(results.filter((r) => r.ok)).toHaveLength(1);
  expect(results).toContainEqual({ ok: true, secret: "secret" });
  expect(state.audit).toHaveBeenCalledTimes(1);
});

it("leaves the share retryable when Vault fails", async () => {
  state.decrypt.mockRejectedValue(new Error("vault unavailable"));
  await expect(caller.reveal({ id: "share" })).rejects.toThrow("vault unavailable");
  expect(state.row).toBeDefined();
  expect(state.audit).not.toHaveBeenCalled();
});

it.each(["expired", "replaced", "deleted"])(
  "does not return a share %s during decryption",
  async (change) => {
    state.decrypt.mockImplementation(async () => {
      if (change === "deleted") {
        state.row = undefined;
      } else if (state.row) {
        state.row = {
          ...state.row,
          ...(change === "expired" ? { expiresAt: 0 } : { encrypted: "other" }),
        };
      }
      return { plaintext: "secret" };
    });
    expect(await caller.reveal({ id: "share" })).toEqual({ ok: false });
    expect(state.audit).not.toHaveBeenCalled();
  },
);

it("does not consume the share or return plaintext if the audit write fails", async () => {
  state.audit.mockRejectedValue(new Error("audit unavailable"));
  await expect(caller.reveal({ id: "share" })).rejects.toThrow("audit unavailable");
  expect(state.row).toBeDefined();
});
