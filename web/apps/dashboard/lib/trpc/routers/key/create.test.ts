import { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: {
    keyAuthId: string;
    bytes: number;
    identityId: string;
    enabled: boolean;
  };
}) => Promise<unknown>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  identityProjectId: "proj_identity",
  keyInsertCalls: 0,
}));

vi.mock("@/lib/db", () => {
  const tx = {
    query: {
      identities: {
        findFirst: async () => ({ id: "id_1", projectId: state.identityProjectId }),
      },
    },
    insert: () => ({
      values: async () => {
        state.keyInsertCalls++;
      },
    }),
  };

  return {
    db: {
      query: {
        keyAuth: {
          findFirst: async () => ({
            id: "ks_1",
            projectId: "proj_key",
            storeEncryptedKeys: false,
          }),
        },
      },
      transaction: async <T>(fn: (transaction: typeof tx) => Promise<T>) => fn(tx),
    },
    schema: { keys: {} },
  };
});

vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: vi.fn(() => ({})) }));
vi.mock("@unkey/keys", () => ({
  newKey: vi.fn(async () => ({
    key: "key",
    hash: "hash",
    prefix: "",
    start: "key_start",
    end: "last",
  })),
}));

vi.mock("../../trpc", () => ({
  ratelimit: { create: {} },
  withRatelimit: () => ({}),
  workspaceProcedure: {
    use: () => ({
      input: () => ({
        mutation: (fn: Resolver) => {
          state.resolver = fn;
          return {};
        },
      }),
    }),
  },
}));

import "./create";

const ctx = {
  workspace: { id: "ws_1" },
  user: { id: "user_1" },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function createKey() {
  if (!state.resolver) {
    throw new Error("createKey resolver was not registered");
  }

  return state.resolver({
    ctx,
    input: { keyAuthId: "ks_1", bytes: 16, identityId: "id_1", enabled: true },
  });
}

describe("createKey", () => {
  beforeEach(() => {
    state.identityProjectId = "proj_identity";
    state.keyInsertCalls = 0;
  });

  it("rejects an identity from another project before writing the key", async () => {
    await expect(createKey()).rejects.toMatchObject({
      code: "INTERNAL_SERVER_ERROR",
    } satisfies Partial<TRPCError>);
    expect(state.keyInsertCalls).toBe(0);
  });
});
