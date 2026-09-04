import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: {
    keyId: string;
    expiration: number;
  };
}) => Promise<unknown>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  insertedKey: null as Record<string, unknown> | null,
}));

vi.mock("@/gen/proto/vault/v1/service_pb", () => ({ VaultService: {} }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn(async () => undefined) }));
vi.mock("@/lib/env", () => ({
  env: () => ({ UNKEY_WORKSPACE_ID: "ws_internal" }),
}));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => ({}) }));
vi.mock("@unkey/id", () => ({ newId: (prefix: string) => `${prefix}_new` }));
vi.mock("@unkey/keys", () => ({
  newKey: vi.fn(async () => ({
    key: "unkey_new_secret",
    hash: "new_hash",
    prefix: "unkey",
    start: "unkey_new",
    end: "cret",
  })),
}));

vi.mock("@/lib/db", () => {
  const schema = {
    keys: {
      id: "keys.id",
      workspaceId: "keys.workspace_id",
      forWorkspaceId: "keys.for_workspace_id",
      deletedAtM: "keys.deleted_at_m",
    },
    encryptedKeys: {},
    ratelimits: {},
    keysRoles: {},
    keysPermissions: {},
  };

  const tx = {
    select: () => ({
      from: () => ({
        where: () => ({
          for: async () => [{ id: "key_legacy" }],
        }),
      }),
    }),
    query: {
      keys: {
        findFirst: async () => ({
          id: "key_legacy",
          keyAuthId: "ks_root",
          prefix: "unkey",
          start: "unkey_legacy",
          workspaceId: "ws_internal",
          forWorkspaceId: "ws_customer",
          name: "Legacy root key",
          identityId: "identity_legacy_user",
          meta: null,
          expires: null,
          refillDay: null,
          refillAmount: null,
          lastRefillAt: null,
          enabled: true,
          remaining: null,
          environment: null,
          keyAuth: {
            storeEncryptedKeys: false,
            defaultBytes: 16,
            api: null,
          },
          encrypted: null,
          ratelimits: [],
          roles: [],
          permissions: [],
        }),
      },
    },
    insert: (table: unknown) => ({
      values: async (values: Record<string, unknown>) => {
        if (table === schema.keys) {
          state.insertedKey = values;
        }
      },
    }),
    update: () => ({
      set: () => ({
        where: async () => undefined,
      }),
    }),
  };

  return {
    and: (...filters: unknown[]) => filters,
    db: {
      transaction: async <T>(fn: (transaction: typeof tx) => Promise<T>) => fn(tx),
    },
    eq: (left: unknown, right: unknown) => [left, right],
    isNull: (value: unknown) => [value, null],
    schema,
  };
});

vi.mock("../../../trpc", () => {
  type Procedure = {
    use: () => Procedure;
    input: () => Procedure;
    mutation: (resolver: Resolver) => object;
  };

  const procedure: Procedure = {
    use: () => procedure,
    input: () => procedure,
    mutation: (resolver) => {
      state.resolver = resolver;
      return {};
    },
  };

  return {
    ratelimit: { create: {} },
    requireWorkspaceAdmin: {},
    withRatelimit: () => ({}),
    workspaceProcedure: procedure,
  };
});

import "./index";

function rerollRootKey() {
  if (!state.resolver) {
    throw new Error("root-key reroll resolver was not registered");
  }

  return state.resolver({
    ctx: {
      workspace: { id: "ws_customer" },
      user: { id: "user_admin" },
      audit: { location: "127.0.0.1", userAgent: "vitest" },
    },
    input: {
      keyId: "key_legacy",
      expiration: 0,
    },
  });
}

describe("rerollRootKey", () => {
  beforeEach(() => {
    state.insertedKey = null;
  });

  // Root keys are system credentials. Legacy user identity links must not
  // propagate when an administrator rotates one.
  it("creates the replacement root key without an identity", async () => {
    await rerollRootKey();

    expect(state.insertedKey).toMatchObject({
      id: "key_new",
      forWorkspaceId: "ws_customer",
      identityId: null,
    });
  });
});
